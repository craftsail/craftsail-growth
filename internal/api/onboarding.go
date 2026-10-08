// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/craftsail/craftsail-growth/internal/pkg/resp"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

func (h *Handler) getProgress(c *gin.Context) {
	out, err := h.projects.Progress(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(out))
}
func (h *Handler) confirmProgress(c *gin.Context) {
	actor := who(c)
	if actor == nil || actor.User == nil {
		fail(c, resp.CodeForbidden, http.StatusForbidden, "confirmation requires a signed-in user")
		return
	}
	var body struct {
		Kind     string `json:"kind"`
		Revision string `json:"revision"`
		AuditID  uint64 `json:"audit_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "invalid confirmation")
		return
	}
	var err error
	switch body.Kind {
	case "brand", "questions":
		err = h.projects.ConfirmReview(c.Request.Context(), c.Param("slug"), body.Kind, body.Revision, actor.User.ID)
	case "audit_helpful":
		err = h.projects.AuditHelpful(c.Request.Context(), c.Param("slug"), body.AuditID, actor.User.ID)
	default:
		err = project.ErrReviewUnavailable
	}
	if errors.Is(err, project.ErrReviewChanged) {
		fail(c, resp.CodeConflict, http.StatusConflict, err.Error())
		return
	}
	if errors.Is(err, project.ErrReviewUnavailable) {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(c, err)
		return
	}
	h.getProgress(c)
}
