// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/pkg/dotenv"
	"github.com/craftsail/craftsail-growth/internal/pkg/resp"
	"github.com/craftsail/craftsail-growth/internal/service/account"
	"github.com/craftsail/craftsail-growth/internal/service/audit"
	"github.com/craftsail/craftsail-growth/internal/service/bootstrap"
	"github.com/craftsail/craftsail-growth/internal/service/jobs"
	"github.com/craftsail/craftsail-growth/internal/service/opportunity"
	"github.com/craftsail/craftsail-growth/internal/service/plan"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"github.com/craftsail/craftsail-growth/internal/service/report"
	"github.com/craftsail/craftsail-growth/internal/service/sample"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

// Wire builds the handler that Mount registers routes on. It returns an
// error instead of starting the server with an unusable handler when the
// database is missing, or when seeding the first admin from the environment
// fails although a password was set: letting setup stay open in that case
// would let a stranger create the first admin.
func Wire(db *gorm.DB, token string, js *jobs.Service) (*Handler, error) {
	if db == nil {
		return nil, fmt.Errorf("wire: no database connection")
	}
	boot := bootstrap.New(db)
	boot.LLM = sample.NewAsker()
	h := NewHandler(
		project.New(db),
		audit.New(db),
		boot,
		sample.New(db, sample.NewAsker()),
		plan.New(db),
		token,
	)
	h.report = report.New(db)
	if js == nil {
		js = jobs.New(db)
		jobs.Bind(js, db)
	}
	h.web = webstats.New(db)
	h.jobs = js
	h.opportunity = opportunity.NewDB(db)
	h.accounts = account.New(db)
	h.EnvRoot = dotenv.FindRoot()
	// CRAFTSAIL_GROWTH_USER / _PASSWORD seed the first admin.
	envUser, envPass := dotenv.Get("CRAFTSAIL_GROWTH_USER"), dotenv.Get("CRAFTSAIL_GROWTH_PASSWORD")
	if err := h.accounts.SeedAdmin(context.Background(), envUser, envPass); err != nil {
		if envPass != "" {
			// A password was set, so the operator clearly meant this to
			// become the first admin; failing quietly would leave the
			// users table empty and /api/setup open to anyone.
			return nil, fmt.Errorf("seed admin from env: %w", err)
		}
		log.Printf("seed admin from env: %v", err)
	}
	// With no account at all, create the default admin so the first visit
	// can sign in; the dashboard asks to change its password.
	if err := h.accounts.SeedDefault(context.Background()); err != nil {
		return nil, fmt.Errorf("create default admin: %w", err)
	}
	return h, nil
}

func (h *Handler) startJob(c *gin.Context) {
	if h.jobs == nil {
		writeErr(c, fmt.Errorf("jobs service not configured"))
		return
	}
	var body struct {
		Action string         `json:"action"`
		Params map[string]any `json:"params"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Action == "" {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, "action is required")
		c.JSON(st, env)
		return
	}
	j, err := h.jobs.Start(c.Request.Context(), c.Param("slug"), body.Action, body.Params)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"job": j}))
}

func (h *Handler) listJobs(c *gin.Context) {
	slug := c.Query("slug")
	if slug == "" && !who(c).admin() {
		fail(c, resp.CodeForbidden, http.StatusForbidden, "only an admin can list every job")
		return
	}
	if slug != "" && !h.allowProject(c, slug, false) {
		return
	}
	if h.jobs == nil {
		writeErr(c, fmt.Errorf("jobs service not configured"))
		return
	}
	rows, running, err := h.jobs.Recent(c.Request.Context(), slug, 12)
	if err != nil {
		writeErr(c, err)
		return
	}
	var rid any
	if running != nil {
		rid = running.ID
	}
	views := make([]jobView, 0, len(rows))
	for _, row := range rows {
		views = append(views, jobView{Job: row, Label: h.jobs.Label(row.Action)})
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"jobs": views, "running": rid}))
}

type jobView struct {
	model.Job
	Label string `json:"label"`
}

func (h *Handler) getJob(c *gin.Context) {
	if h.jobs == nil {
		writeErr(c, fmt.Errorf("jobs service not configured"))
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, "invalid job id")
		c.JSON(st, env)
		return
	}
	off, _ := strconv.Atoi(c.Query("offset"))
	chunk, newOff, j, err := h.jobs.Tail(c.Request.Context(), id, off)
	if err != nil {
		writeErr(c, err)
		return
	}
	if !h.allowProjectID(c, j.ProjectID, false) {
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"job": j, "log": chunk, "offset": newOff}))
}

func (h *Handler) stopJob(c *gin.Context) {
	if h.jobs == nil {
		writeErr(c, fmt.Errorf("jobs service not configured"))
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, "invalid job id")
		c.JSON(st, env)
		return
	}
	j, err := h.jobs.Get(c.Request.Context(), id)
	if err != nil {
		writeErr(c, err)
		return
	}
	if !h.allowProjectID(c, j.ProjectID, true) {
		return
	}
	if err := h.jobs.Stop(c.Request.Context(), id); err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"ok": true}))
}

func (h *Handler) buildReport(c *gin.Context) {
	if h.report == nil {
		writeErr(c, fmt.Errorf("report service not configured"))
		return
	}
	out, err := h.report.Build(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(out))
}

func (h *Handler) getReport(c *gin.Context) {
	if h.report == nil {
		writeErr(c, fmt.Errorf("report service not configured"))
		return
	}
	row, err := h.report.Latest(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writeErr(c, err)
		return
	}
	if row == nil {
		env, st := resp.Fail(resp.CodeNotFound, http.StatusNotFound, "no report yet")
		c.JSON(st, env)
		return
	}
	c.JSON(http.StatusOK, resp.OK(row))
}

func (h *Handler) patchMonitor(c *gin.Context) {
	p, err := h.projects.Get(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writeErr(c, err)
		return
	}
	var body struct {
		EveryDays  int  `json:"every_days"`
		RunsPerDay *int `json:"runs_per_day"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, "invalid request body")
		c.JSON(st, env)
		return
	}
	if body.RunsPerDay != nil {
		if *body.RunsPerDay < 1 || *body.RunsPerDay > 10 {
			env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, "runs_per_day must be between 1 and 10")
			c.JSON(st, env)
			return
		}
		p.MonitorRunsPerDay = *body.RunsPerDay
	}
	if body.EveryDays <= 0 {
		p.MonitorEveryDays = nil
		p.MonitorNextRun = nil
	} else {
		p.MonitorEveryDays = &body.EveryDays
		n := time.Now().Add(time.Duration(body.EveryDays) * 24 * time.Hour)
		p.MonitorNextRun = &n
	}
	if err := h.projects.Save(c.Request.Context(), p); err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(p))
}
