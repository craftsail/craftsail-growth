// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"errors"
	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/pkg/resp"
	"github.com/craftsail/craftsail-growth/internal/service/plan"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func (h *Handler) listObservations(c *gin.Context) {
	if h.plan == nil {
		writeErr(c, errors.New("plan service unavailable"))
		return
	}
	rows, err := h.plan.Observations(c.Request.Context(), c.Param("slug"), c.Param("code"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"items": rows}))
}
func (h *Handler) releaseTask(c *gin.Context) {
	if h.plan == nil {
		writeErr(c, errors.New("plan service unavailable"))
		return
	}
	var body model.Observation
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "invalid release")
		return
	}
	row, err := h.plan.Release(c.Request.Context(), c.Param("slug"), c.Param("code"), body)
	if errors.Is(err, plan.ErrObservation) {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(row))
}
func (h *Handler) evaluateObservation(c *gin.Context) {
	if h.plan == nil {
		writeErr(c, errors.New("plan service unavailable"))
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "invalid observation ID")
		return
	}
	var body struct {
		Refresh bool   `json:"refresh"`
		Notes   string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "invalid evaluation")
		return
	}
	row, err := h.plan.Evaluate(c.Request.Context(), c.Param("slug"), c.Param("code"), id, body.Refresh, body.Notes)
	if errors.Is(err, plan.ErrObservationMissing) {
		fail(c, resp.CodeNotFound, http.StatusNotFound, err.Error())
		return
	}
	if errors.Is(err, plan.ErrObservation) {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(row))
}
