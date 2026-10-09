// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"github.com/craftsail/craftsail-growth/internal/pkg/resp"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (h *Handler) mapLanding(c *gin.Context) {
	if h.web == nil {
		writeErr(c, errWeb())
		return
	}
	in, ok := searchInput(c)
	if !ok {
		return
	}
	out, err := h.web.MapLanding(c.Request.Context(), c.Param("slug"), in)
	if err != nil {
		searchError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(out))
}
