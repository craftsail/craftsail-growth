// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/craftsail/craftsail-growth/internal/pkg/resp"
)

func (h *Handler) getKeywords(c *gin.Context) {
	if h.web == nil {
		writeErr(c, errWeb())
		return
	}
	b, err := h.web.SearchBoard(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"items": b.Keywords, "period": b.Period}))
}

func (h *Handler) getGscPages(c *gin.Context) {
	if h.web == nil {
		writeErr(c, errWeb())
		return
	}
	b, err := h.web.SearchBoard(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"items": b.Pages, "period": b.Period}))
}

func (h *Handler) saveKeyword(c *gin.Context) {
	if h.web == nil {
		writeErr(c, errWeb())
		return
	}
	var body struct {
		Query string `json:"query"`
		Notes string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "query is required"})
		return
	}
	if err := h.web.SaveKeyword(c.Request.Context(), c.Param("slug"), body.Query, body.Notes); err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"ok": true}))
}

func (h *Handler) listSavedKeywords(c *gin.Context) {
	if h.web == nil {
		writeErr(c, errWeb())
		return
	}
	rows, err := h.web.ListSaved(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"items": rows}))
}

func errWeb() error { return errString("webstats service not configured") }

type errString string

func (e errString) Error() string { return string(e) }
