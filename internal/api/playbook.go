// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/craftsail/craftsail-growth/internal/pkg/resp"
	"github.com/craftsail/craftsail-growth/internal/service/playbook"
)

func (h *Handler) getPlaybook(c *gin.Context) {
	if h.playbook == nil {
		writeErr(c, errors.New("playbook not configured"))
		return
	}
	out, err := h.playbook.Status(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(out))
}

func (h *Handler) putPlaybookStage(c *gin.Context) {
	if h.playbook == nil {
		writeErr(c, errors.New("playbook not configured"))
		return
	}
	var body struct {
		ProductStage string `json:"product_stage"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "invalid product stage")
		return
	}
	h.playbookWrite(c, h.playbook.SetStage(c.Request.Context(), c.Param("slug"), body.ProductStage))
}

func (h *Handler) putPlaybookSignal(c *gin.Context) {
	if h.playbook == nil {
		writeErr(c, errors.New("playbook not configured"))
		return
	}
	var body struct {
		Confirmed bool `json:"confirmed"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "invalid confirmation")
		return
	}
	var user uint64
	if a := who(c); a != nil && a.User != nil {
		user = a.User.ID
	}
	h.playbookWrite(c, h.playbook.Confirm(c.Request.Context(), c.Param("slug"), c.Param("signal"), user, body.Confirmed))
}

// playbookWrite answers a write with the new status, so the page updates
// from one response.
func (h *Handler) playbookWrite(c *gin.Context, err error) {
	if errors.Is(err, playbook.ErrInvalid) {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(c, err)
		return
	}
	h.getPlaybook(c)
}
