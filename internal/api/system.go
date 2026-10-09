// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"errors"
	"log"
	"net/http"

	"github.com/craftsail/craftsail-growth/internal/pkg/resp"
	"github.com/craftsail/craftsail-growth/internal/service/update"
	"github.com/gin-gonic/gin"
)

func updateError(c *gin.Context, err error) {
	code := "update.failed"
	for _, known := range []error{update.ErrBusy, update.ErrUnsupported, update.ErrPending, update.ErrTarget, update.ErrDownload, update.ErrChecksum, update.ErrArchive, update.ErrInstall, update.ErrRestart, update.ErrJobs} {
		if errors.Is(err, known) {
			code = known.Error()
			break
		}
	}
	log.Printf("system update: %v", err)
	fail(c, resp.CodeConflict, http.StatusConflict, code)
}
func (h *Handler) hasUpdater(c *gin.Context) bool {
	if h.updater == nil {
		updateError(c, update.ErrUnsupported)
		return false
	}
	return true
}
func (h *Handler) systemVersion(c *gin.Context) {
	if !h.hasUpdater(c) {
		return
	}
	state, err := h.updater.Check(c.Request.Context(), c.Query("force") == "true")
	if err != nil {
		updateError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(state))
}
func (h *Handler) applySystem(c *gin.Context, rollback bool) {
	if !h.hasUpdater(c) {
		return
	}
	var body struct {
		Version string `json:"version"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Version == "" {
		updateError(c, update.ErrTarget)
		return
	}
	if err := h.updater.Apply(c.Request.Context(), body.Version, rollback); err != nil {
		updateError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"need_restart": true}))
}
func (h *Handler) systemUpdate(c *gin.Context)   { h.applySystem(c, false) }
func (h *Handler) systemRollback(c *gin.Context) { h.applySystem(c, true) }
func (h *Handler) systemRestart(c *gin.Context) {
	if !h.hasUpdater(c) {
		return
	}
	if err := h.updater.Restart(); err != nil {
		updateError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"restarting": true}))
}
