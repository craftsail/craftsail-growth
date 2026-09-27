// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"crypto/hmac"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/pkg/resp"
	"github.com/craftsail/craftsail-growth/internal/service/account"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

const sessionCookie = "craftsail_growth_session"

var accountSessionTTL = account.SessionTTL

// principal is who is calling: a signed-in user, or the API token.
type principal struct {
	User    *model.User
	Service bool
}

func (p *principal) admin() bool {
	return p != nil && (p.Service || (p.User != nil && p.User.Role == model.RoleAdmin))
}

func (p *principal) name() string {
	if p == nil {
		return ""
	}
	if p.Service {
		return "api-token"
	}
	return p.User.Username
}

const principalKey = "principal"

func who(c *gin.Context) *principal {
	v, _ := c.Get(principalKey)
	p, _ := v.(*principal)
	return p
}

// resolve finds the caller from the API token or the session cookie. A
// non-nil error means the lookup itself failed (e.g. the database is down),
// which callers must tell apart from "not signed in".
func (h *Handler) resolve(c *gin.Context) (*principal, error) {
	candidates := []string{c.GetHeader("X-Craftsail-Growth-Token")}
	if c.Request.Method == http.MethodGet {
		// ?token= is for links and browser navigation, which are always
		// GET; a mutating request must use a header so a URL logged
		// somewhere (proxy logs, browser history) cannot be replayed.
		candidates = append(candidates, c.Query("token"))
	}
	if h.token != "" {
		for _, v := range candidates {
			if v != "" && hmac.Equal([]byte(v), []byte(h.token)) {
				return &principal{Service: true}, nil
			}
		}
	}
	ck, err := c.Cookie(sessionCookie)
	if err != nil || ck == "" {
		return nil, nil
	}
	u, err := h.accounts.UserForToken(c.Request.Context(), ck)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, nil
	}
	return &principal{User: u}, nil
}

// isSafeMethod reports whether a method never changes state, so the JSON
// content-type check in auth, login and setup does not apply to it.
func isSafeMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions
}

// requireJSON rejects a mutating request whose body is not JSON. A browser
// form or an <img>/<script> tag on another site can make the user's browser
// send a same-site-cookie request with a text/plain or empty content type,
// but cannot set an arbitrary one, so this blocks that CSRF path without
// needing a token. It does not apply to the API token, which no ordinary
// web page can present.
func requireJSON(c *gin.Context) bool {
	if strings.HasPrefix(c.GetHeader("Content-Type"), "application/json") {
		return true
	}
	fail(c, resp.CodeBadRequest, http.StatusUnsupportedMediaType, "send JSON")
	return false
}

func (h *Handler) setSession(c *gin.Context, tok string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: sessionCookie, Value: tok, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode,
		Secure: c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https",
		MaxAge: int(accountSessionTTL.Seconds()),
	})
}

func clearSession(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
}

func fail(c *gin.Context, code, status int, msg string) {
	env, _ := resp.Fail(code, status, msg)
	c.AbortWithStatusJSON(status, env)
}

type perm string

const (
	permAuthed perm = "authed" // any signed-in caller; the handler checks further
	permAdmin  perm = "admin"
	permView   perm = "view" // needs :slug; view or edit access
	permEdit   perm = "edit" // needs :slug; edit access
)

// route registers a handler with its permission. Every /api route except
// login, logout, setup and session goes through here; a test enforces it.
func (h *Handler) route(g *gin.RouterGroup, method, path string, p perm, fn gin.HandlerFunc) {
	if h.perms == nil {
		h.perms = map[string]perm{}
	}
	h.perms[method+" "+g.BasePath()+path] = p
	g.Handle(method, path, h.guard(p), fn)
}

func (h *Handler) guard(p perm) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch p {
		case permAdmin:
			if !who(c).admin() {
				fail(c, resp.CodeForbidden, http.StatusForbidden, "only an admin can do this")
				return
			}
		case permView, permEdit:
			if !h.allowProject(c, c.Param("slug"), p == permEdit) {
				return
			}
		}
		c.Next()
	}
}

// allowProject checks the caller's access to a project by slug and writes
// the error response when access is missing. Unshared projects answer 404
// so their names do not leak.
func (h *Handler) allowProject(c *gin.Context, slug string, edit bool) bool {
	p := who(c)
	if p.admin() {
		return true
	}
	proj, err := h.projects.Get(c.Request.Context(), slug)
	if err != nil {
		writeErr(c, err)
		c.Abort()
		return false
	}
	access, err := h.accounts.Access(c.Request.Context(), p.User, proj.ID)
	if err != nil {
		writeErr(c, err)
		c.Abort()
		return false
	}
	if access == "" {
		fail(c, resp.CodeNotFound, http.StatusNotFound, "project not found")
		return false
	}
	if edit && access != model.AccessEdit {
		fail(c, resp.CodeForbidden, http.StatusForbidden, "you have view access to this project")
		return false
	}
	return true
}

// allowProjectID is allowProject for handlers that start from a job.
// Jobs without a project are admin-only.
func (h *Handler) allowProjectID(c *gin.Context, projectID *uint64, edit bool) bool {
	if who(c).admin() {
		return true
	}
	if projectID == nil {
		fail(c, resp.CodeForbidden, http.StatusForbidden, "only an admin can do this")
		return false
	}
	proj, err := h.projects.ByID(c.Request.Context(), *projectID)
	if errors.Is(err, project.ErrNotFound) {
		fail(c, resp.CodeNotFound, http.StatusNotFound, "job not found")
		return false
	}
	if err != nil {
		writeErr(c, err)
		c.Abort()
		return false
	}
	return h.allowProject(c, proj.Slug, edit)
}
