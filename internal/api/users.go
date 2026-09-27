// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/pkg/resp"
	"github.com/craftsail/craftsail-growth/internal/service/account"
)

type grantView struct {
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Access string `json:"access"`
}

type userView struct {
	ID          uint64      `json:"id"`
	Username    string      `json:"username"`
	Role        string      `json:"role"`
	Disabled    bool        `json:"disabled"`
	LastLoginAt *int64      `json:"last_login_at"`
	Projects    []grantView `json:"projects"`
}

func (h *Handler) userView(c *gin.Context, u model.User, projects []model.Project) (userView, error) {
	v := userView{ID: u.ID, Username: u.Username, Role: u.Role, Disabled: u.Disabled, LastLoginAt: u.LastLoginAt, Projects: []grantView{}}
	if u.Role == model.RoleAdmin {
		return v, nil
	}
	grants, err := h.accounts.Grants(c.Request.Context(), u.ID)
	if err != nil {
		return v, err
	}
	for _, p := range projects {
		if a := grants[p.ID]; a != "" {
			v.Projects = append(v.Projects, grantView{Slug: p.Slug, Name: p.Name, Access: a})
		}
	}
	return v, nil
}

func (h *Handler) listUsers(c *gin.Context) {
	ctx := c.Request.Context()
	users, err := h.accounts.List(ctx)
	if err != nil {
		writeErr(c, err)
		return
	}
	projects, err := h.projects.List(ctx)
	if err != nil {
		writeErr(c, err)
		return
	}
	out := make([]userView, 0, len(users))
	for _, u := range users {
		v, err := h.userView(c, u, projects)
		if err != nil {
			writeErr(c, err)
			return
		}
		out = append(out, v)
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"items": out}))
}

func (h *Handler) createUser(c *gin.Context) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Role == "" {
		body.Role = model.RoleMember
	}
	u, err := h.accounts.CreateUser(c.Request.Context(), body.Username, body.Password, body.Role)
	if err != nil {
		accountErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"id": u.ID, "username": u.Username, "role": u.Role}))
}

func (h *Handler) patchUser(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	var body struct {
		Role     *string `json:"role"`
		Disabled *bool   `json:"disabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "invalid request body")
		return
	}
	if p := who(c); p.User != nil && p.User.ID == id && (body.Role != nil || body.Disabled != nil) {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "you cannot change your own role or disable yourself")
		return
	}
	if err := h.accounts.Update(c.Request.Context(), id, account.Patch{Role: body.Role, Disabled: body.Disabled}); err != nil {
		accountErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"ok": true}))
}

func (h *Handler) deleteUser(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	if p := who(c); p.User != nil && p.User.ID == id {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "you cannot delete your own account")
		return
	}
	if err := h.accounts.Delete(c.Request.Context(), id); err != nil {
		accountErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"ok": true}))
}

// resetPassword lets an admin set a new password for someone who forgot
// theirs. It ends that user's sessions.
func (h *Handler) resetPassword(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "invalid request body")
		return
	}
	if p := who(c); p.User != nil && p.User.ID == id {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "change your own password under Change password")
		return
	}
	if err := h.accounts.SetPassword(c.Request.Context(), id, body.Password); err != nil {
		accountErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"ok": true}))
}

// putUserAccess changes a member's access to the listed projects; send
// access "" to remove one. Projects not listed keep their access.
func (h *Handler) putUserAccess(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	var body struct {
		Items []struct {
			Slug   string `json:"slug"`
			Access string `json:"access"`
		} `json:"items"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "invalid request body")
		return
	}
	ctx := c.Request.Context()
	// Check every item first so a bad slug or level changes nothing.
	ids := make([]uint64, len(body.Items))
	for i, it := range body.Items {
		if it.Access != "" && it.Access != model.AccessView && it.Access != model.AccessEdit {
			accountErr(c, account.ErrBadAccess)
			return
		}
		p, err := h.projects.Get(ctx, it.Slug)
		if err != nil {
			writeErr(c, err)
			return
		}
		ids[i] = p.ID
	}
	for i, it := range body.Items {
		if err := h.accounts.Grant(ctx, id, ids[i], it.Access); err != nil {
			accountErr(c, err)
			return
		}
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"ok": true}))
}

// changeOwnPassword checks the current password, sets the new one, ends
// every session of this user and signs this browser in again.
func (h *Handler) changeOwnPassword(c *gin.Context) {
	p := who(c)
	if p.User == nil {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "the API token has no password")
		return
	}
	var body struct {
		Current  string `json:"current"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "invalid request body")
		return
	}
	ctx := c.Request.Context()
	// The default password is public and was just used to sign in, so
	// asking for it again proves nothing.
	if !p.User.MustChangePassword {
		if err := h.accounts.CheckPassword(ctx, p.User.ID, body.Current); err != nil {
			if errors.Is(err, account.ErrBadCredentials) {
				fail(c, resp.CodeBadRequest, http.StatusBadRequest, "the current password is wrong")
				return
			}
			writeErr(c, err)
			return
		}
	}
	if err := h.accounts.SetPassword(ctx, p.User.ID, body.Password); err != nil {
		accountErr(c, err)
		return
	}
	tok, err := h.accounts.StartSession(ctx, p.User.ID)
	if err != nil {
		writeErr(c, err)
		return
	}
	h.setSession(c, tok)
	c.JSON(http.StatusOK, resp.OK(gin.H{"ok": true}))
}

func userID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "invalid user id")
		return 0, false
	}
	return id, true
}

// accountErr maps account errors: bad input is 400, a missing user 404, a
// taken name 409; anything else is a server error.
func accountErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, account.ErrUserNotFound):
		fail(c, resp.CodeNotFound, http.StatusNotFound, err.Error())
	case errors.Is(err, account.ErrUserExists):
		fail(c, resp.CodeConflict, http.StatusConflict, err.Error())
	case account.IsValidation(err), errors.Is(err, account.ErrLastAdmin), errors.Is(err, account.ErrBadAccess):
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, err.Error())
	default:
		writeErr(c, err)
	}
}
