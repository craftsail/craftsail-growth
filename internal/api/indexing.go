// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/pkg/resp"
	"github.com/gin-gonic/gin"
)

func indexPagination(c *gin.Context) (int, int, bool) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, sizeErr := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if err != nil || sizeErr != nil || page < 1 || page > 1000000 || size < 1 || size > 200 {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "invalid pagination")
		return 0, 0, false
	}
	return page, size, true
}
func (h *Handler) getIndexInventory(c *gin.Context) {
	if h.web == nil {
		writeErr(c, errWeb())
		return
	}
	page, size, ok := indexPagination(c)
	if !ok {
		return
	}
	state := c.Query("state")
	switch state {
	case "", "pending", "indexed", "other", "error", "due":
	default:
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "invalid index state")
		return
	}
	group := c.Query("group")
	switch group {
	case "", "homepage", "article", "documentation", "product", "document", "page":
	default:
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "invalid page group")
		return
	}
	query := strings.TrimSpace(c.Query("q"))
	if len(query) > 2000 {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "URL filter too long")
		return
	}
	out, err := h.web.IndexInventory(c.Request.Context(), c.Param("slug"), query, state, page, size, group)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(out))
}
func (h *Handler) getIndexHistory(c *gin.Context) {
	if h.web == nil {
		writeErr(c, errWeb())
		return
	}
	page, size, ok := indexPagination(c)
	if !ok {
		return
	}
	url := strings.TrimSpace(c.Query("url"))
	if url == "" || len(url) > 10000 {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "URL is required")
		return
	}
	out, err := h.web.IndexHistory(c.Request.Context(), c.Param("slug"), url, page, size)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(out))
}

func (h *Handler) setIndexPublished(c *gin.Context) {
	if h.web == nil {
		writeErr(c, errWeb())
		return
	}
	var body struct {
		URL         string `json:"url"`
		PublishedAt *int64 `json:"published_at"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.URL == "" {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "invalid publication record")
		return
	}
	if body.PublishedAt != nil && (*body.PublishedAt < 0 || *body.PublishedAt > time.Now().Unix()) {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "publication time must be in the past")
		return
	}
	if err := h.web.SetURLPublished(c.Request.Context(), c.Param("slug"), body.URL, body.PublishedAt); err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"ok": true}))
}
func (h *Handler) addIndexSitemap(c *gin.Context) {
	if h.web == nil {
		writeErr(c, errWeb())
		return
	}
	var body struct {
		URL string `json:"url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.URL) > 8192 {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "invalid sitemap URL")
		return
	}
	if err := h.web.AddSitemap(c.Request.Context(), c.Param("slug"), body.URL); err != nil {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"ok": true}))
}
