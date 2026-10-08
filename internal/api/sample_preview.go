// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"github.com/craftsail/craftsail-growth/internal/pkg/resp"
	"github.com/craftsail/craftsail-growth/internal/service/sample"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func (h *Handler) samplePreview(c *gin.Context) {
	var in struct {
		Repeat    int    `form:"repeat"`
		Limit     int    `form:"limit"`
		Platforms string `form:"platforms"`
	}
	if err := c.ShouldBindQuery(&in); err != nil || in.Repeat < 0 || in.Repeat > 10 || in.Limit < 0 || in.Limit > 1000 {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "invalid sampling preview")
		return
	}
	plats := []string{}
	if in.Platforms != "" {
		plats = strings.Split(in.Platforms, ",")
	}
	out, err := h.sample.Preview(c.Request.Context(), c.Param("slug"), sample.RunInput{Repeat: in.Repeat, Limit: in.Limit, Platforms: plats})
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(out))
}
