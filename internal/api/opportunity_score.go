// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"errors"
	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/pkg/resp"
	"github.com/craftsail/craftsail-growth/internal/service/opportunity"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (h *Handler) scoreOpportunity(c *gin.Context) {
	var row model.OpportunityScore
	if err := c.ShouldBindJSON(&row); err != nil {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "invalid score")
		return
	}
	err := h.opportunity.Score(c.Request.Context(), c.Param("slug"), row)
	switch {
	case errors.Is(err, opportunity.ErrScore):
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, err.Error())
	case errors.Is(err, opportunity.ErrNotFound):
		fail(c, resp.CodeNotFound, http.StatusNotFound, err.Error())
	case err != nil:
		writeErr(c, err)
	default:
		c.JSON(http.StatusOK, resp.OK(nil))
	}
}
