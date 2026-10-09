// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"github.com/craftsail/craftsail-growth/internal/pkg/resp"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (h *Handler) questionLibraries(c *gin.Context) {
	rows, err := h.bootstrap.Libraries(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"items": rows}))
}
