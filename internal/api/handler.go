// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

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

type Handler struct {
	projects    *project.Service
	audit       *audit.Service
	bootstrap   *bootstrap.Service
	sample      *sample.Service
	plan        *plan.Service
	report      *report.Service
	jobs        *jobs.Service
	web         *webstats.Service
	opportunity *opportunity.Service
	token       string
	accounts    *account.Service
	perms       map[string]perm // "METHOD /api/path" -> permission, filled by route()
	EnvRoot     string
}

func NewHandler(projects *project.Service, auditSvc *audit.Service, boot *bootstrap.Service, samp *sample.Service, pl *plan.Service, token string) *Handler {
	return &Handler{projects: projects, audit: auditSvc, bootstrap: boot, sample: samp, plan: pl, token: strings.TrimSpace(token)}
}

func Mount(engine *gin.Engine, h *Handler) {
	api := engine.Group("/api")
	api.POST("/login", h.login)
	api.POST("/logout", h.logout)
	api.POST("/setup", h.setup)
	api.GET("/session", h.session)
	api.Use(h.auth)
	r := func(method, path string, p perm, fn gin.HandlerFunc) { h.route(api, method, path, p, fn) }

	// Signed-in callers; handlers filter or check further.
	r("GET", "/projects", permAuthed, h.listProjects)
	r("GET", "/jobs", permAuthed, h.listJobs)
	r("GET", "/jobs/:id", permAuthed, h.getJob)
	r("POST", "/jobs/:id/stop", permAuthed, h.stopJob)
	r("POST", "/me/password", permAuthed, h.changeOwnPassword)

	// Workspace: admins only.
	r("POST", "/projects", permAdmin, h.createProject)
	r("PATCH", "/projects/:slug", permAdmin, h.patchProject)
	r("GET", "/keys", permAdmin, h.getKeys)
	r("PUT", "/keys", permAdmin, h.putKeys)
	r("POST", "/keys/verify", permAdmin, h.verifyKey)
	r("GET", "/google/connect", permAdmin, h.googleConnect)
	r("GET", "/google/callback", permAdmin, h.googleCallback)
	r("POST", "/google/disconnect", permAdmin, h.googleDisconnect)
	r("GET", "/projects/:slug/google", permAdmin, h.googleStatus)
	r("POST", "/projects/:slug/google", permAdmin, h.googleSave)
	r("GET", "/users", permAdmin, h.listUsers)
	r("POST", "/users", permAdmin, h.createUser)
	r("PATCH", "/users/:id", permAdmin, h.patchUser)
	r("DELETE", "/users/:id", permAdmin, h.deleteUser)
	r("POST", "/users/:id/password", permAdmin, h.resetPassword)
	r("PUT", "/users/:id/access", permAdmin, h.putUserAccess)

	// Project, read.
	r("GET", "/projects/:slug", permView, h.getProject)
	r("GET", "/projects/:slug/progress", permView, h.getProgress)
	r("POST", "/projects/:slug/progress", permEdit, h.confirmProgress)
	r("GET", "/projects/:slug/audit", permView, h.getAudit)
	r("GET", "/projects/:slug/audit/issues", permView, h.getAuditIssues)
	r("GET", "/projects/:slug/audit.md", permView, h.exportAuditMD)
	r("GET", "/projects/:slug/brand", permView, h.getBrand)
	r("GET", "/projects/:slug/questions", permView, h.getQuestions)
	r("GET", "/projects/:slug/competitors", permView, h.getCompetitors)
	r("GET", "/projects/:slug/samples", permView, h.listSamples)
	r("GET", "/projects/:slug/runs", permView, h.listRuns)
	r("GET", "/projects/:slug/measure", permView, h.getMeasure)
	r("GET", "/projects/:slug/sample-sheet", permView, h.sampleSheet)
	r("GET", "/projects/:slug/webstats", permView, h.getWebstats)
	r("GET", "/projects/:slug/indexing", permView, h.getIndexInventory)
	r("PUT", "/projects/:slug/indexing/published", permEdit, h.setIndexPublished)
	r("POST", "/projects/:slug/indexing/sitemaps", permEdit, h.addIndexSitemap)
	r("GET", "/projects/:slug/indexing/history", permView, h.getIndexHistory)
	r("GET", "/projects/:slug/keywords", permView, h.getKeywords)
	r("GET", "/projects/:slug/gsc-pages", permView, h.getGscPages)
	r("GET", "/projects/:slug/search-detail", permView, h.getSearchDetail)
	r("GET", "/projects/:slug/ga-channels", permView, h.getGAChannels)
	r("GET", "/projects/:slug/ga-landings", permView, h.getGALandings)
	r("GET", "/projects/:slug/saved-keywords", permView, h.listSavedKeywords)
	r("GET", "/projects/:slug/opportunities", permView, h.listOpportunities)
	r("GET", "/projects/:slug/report", permView, h.getReport)

	// Project, change.
	r("POST", "/projects/:slug/jobs", permEdit, h.startJob)
	r("POST", "/projects/:slug/audit", permEdit, h.auditProject)
	r("PUT", "/projects/:slug/brand", permEdit, h.putBrand)
	r("PUT", "/projects/:slug/questions", permEdit, h.putQuestions)
	r("PUT", "/projects/:slug/competitors", permEdit, h.putCompetitors)
	r("PATCH", "/projects/:slug/samples/:id", permEdit, h.overrideSample)
	r("POST", "/projects/:slug/runs/:id/retry", permEdit, h.retryRun)
	r("POST", "/projects/:slug/samples/import", permEdit, h.importSamples)
	r("POST", "/projects/:slug/webstats", permEdit, h.runWebstats)
	r("POST", "/projects/:slug/saved-keywords", permEdit, h.saveKeyword)
	r("POST", "/projects/:slug/opportunities/accept", permEdit, h.acceptOpportunity)
	r("POST", "/projects/:slug/opportunities/dismiss", permEdit, h.dismissOpportunity)
	r("PATCH", "/projects/:slug/tasks/:code", permEdit, h.patchTask)
	r("POST", "/projects/:slug/report", permEdit, h.buildReport)
	r("PATCH", "/projects/:slug/monitor", permEdit, h.patchMonitor)

}

// accountsReady reports whether the account service is wired up, and fails
// the request with 500 instead of letting a nil h.accounts panic.
func (h *Handler) accountsReady(c *gin.Context) bool {
	if h.accounts != nil {
		return true
	}
	fail(c, resp.CodeInternal, http.StatusInternalServerError, "accounts not configured")
	return false
}

// auth lets a request through when the caller is known: a signed-in session
// cookie, or the API token (a header for any method, or ?token= for GET).
// A mutating request authenticated by cookie must send a JSON body, which
// blocks the classic cross-site form/CSRF path; the API token is exempt
// since no ordinary web page can present it.
func (h *Handler) auth(c *gin.Context) {
	if !h.accountsReady(c) {
		return
	}
	p, err := h.resolve(c)
	if err != nil {
		fail(c, resp.CodeInternal, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	if p == nil {
		fail(c, resp.CodeUnauthorized, http.StatusUnauthorized, "not signed in")
		return
	}
	if !p.Service && !isSafeMethod(c.Request.Method) && !requireJSON(c) {
		return
	}
	// The default admin's password is public; nothing but replacing it is
	// allowed until then.
	if p.User != nil && p.User.MustChangePassword && c.FullPath() != "/api/me/password" {
		fail(c, resp.CodePasswordChange, http.StatusForbidden, "change the default password first")
		return
	}
	c.Set(principalKey, p)
	c.Next()
}

func (h *Handler) login(c *gin.Context) {
	if !h.accountsReady(c) || !requireJSON(c) {
		return
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "invalid request body")
		return
	}
	if empty, _ := h.accounts.NeedsSetup(c.Request.Context()); empty {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "set up the account first")
		return
	}
	tok, err := h.accounts.SignIn(c.Request.Context(), body.Username, body.Password)
	if errors.Is(err, account.ErrBadCredentials) {
		fail(c, resp.CodeUnauthorized, http.StatusUnauthorized, err.Error())
		return
	}
	if err != nil {
		writeErr(c, err)
		return
	}
	h.setSession(c, tok)
	c.JSON(http.StatusOK, resp.OK(gin.H{"ok": true}))
}

func (h *Handler) setup(c *gin.Context) {
	if !h.accountsReady(c) || !requireJSON(c) {
		return
	}
	ctx := c.Request.Context()
	if empty, err := h.accounts.NeedsSetup(ctx); err != nil || !empty {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "the account is already set up; sign in")
		return
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "invalid request body")
		return
	}
	u, err := h.accounts.CreateUser(ctx, body.Username, body.Password, model.RoleAdmin)
	if err != nil {
		if account.IsValidation(err) {
			fail(c, resp.CodeBadRequest, http.StatusBadRequest, err.Error())
		} else {
			writeErr(c, err)
		}
		return
	}
	tok, err := h.accounts.StartSession(ctx, u.ID)
	if err != nil {
		writeErr(c, err)
		return
	}
	h.setSession(c, tok)
	c.JSON(http.StatusOK, resp.OK(gin.H{"ok": true}))
}

func (h *Handler) logout(c *gin.Context) {
	if !h.accountsReady(c) {
		return
	}
	if ck, err := c.Cookie(sessionCookie); err == nil && ck != "" {
		_ = h.accounts.SignOut(c.Request.Context(), ck)
	}
	clearSession(c)
	c.JSON(http.StatusOK, resp.OK(gin.H{"ok": true}))
}

// session tells the dashboard who is signed in and what they may do.
func (h *Handler) session(c *gin.Context) {
	if !h.accountsReady(c) {
		return
	}
	empty, err := h.accounts.NeedsSetup(c.Request.Context())
	if err != nil {
		fail(c, resp.CodeInternal, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	if empty {
		c.JSON(http.StatusOK, resp.OK(gin.H{"ok": false, "setup": true}))
		return
	}
	p, err := h.resolve(c)
	if err != nil {
		fail(c, resp.CodeInternal, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	if p == nil {
		c.JSON(http.StatusOK, resp.OK(gin.H{"ok": false, "setup": false}))
		return
	}
	user := gin.H{"id": 0, "username": p.name(), "role": model.RoleAdmin}
	if p.User != nil {
		user = gin.H{"id": p.User.ID, "username": p.User.Username, "role": p.User.Role, "must_change_password": p.User.MustChangePassword}
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"ok": true, "user": user}))
}

// projectRow is a project with the caller's access, so the dashboard
// knows which pages to show read-only.
type projectRow struct {
	model.Project
	Access string `json:"access"`
}

// listProjects returns the projects the caller can see.
func (h *Handler) listProjects(c *gin.Context) {
	ctx := c.Request.Context()
	items, err := h.projects.List(ctx)
	if err != nil {
		writeErr(c, err)
		return
	}
	p := who(c)
	out := []projectRow{}
	if p.admin() {
		for _, it := range items {
			out = append(out, projectRow{it, model.AccessEdit})
		}
		c.JSON(http.StatusOK, resp.OK(gin.H{"items": out}))
		return
	}
	grants, err := h.accounts.Grants(ctx, p.User.ID)
	if err != nil {
		writeErr(c, err)
		return
	}
	for _, it := range items {
		if a := grants[it.ID]; a != "" {
			out = append(out, projectRow{it, a})
		}
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"items": out}))
}

func (h *Handler) createProject(c *gin.Context) {
	var body struct {
		URL       string `json:"url"`
		Name      string `json:"name"`
		Slug      string `json:"slug"`
		NoSite    bool   `json:"no_site"`
		Materials string `json:"materials"`
		MaxPages  int    `json:"max_pages"`
		Force     bool   `json:"force"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, "invalid request body")
		c.JSON(st, env)
		return
	}
	p, err := h.projects.Create(c.Request.Context(), project.CreateInput{
		URL:       body.URL,
		Name:      body.Name,
		Slug:      body.Slug,
		NoSite:    body.NoSite,
		Materials: body.Materials,
		MaxPages:  body.MaxPages,
		Force:     body.Force,
	})
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(projectRow{*p, model.AccessEdit}))
}

func (h *Handler) getProject(c *gin.Context) {
	p, err := h.projects.Get(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(p))
}

func (h *Handler) patchProject(c *gin.Context) {
	var body struct {
		SearchMode           *string `json:"search_mode"`
		SearchMinImpressions *int    `json:"search_min_impressions"`
		URL                  *string `json:"url"`
		Site                 *string `json:"site"`
		Name                 *string `json:"name"`
		NoSite               *bool   `json:"no_site"`
		Materials            *string `json:"materials"`
		MaxPages             *int    `json:"max_pages"`
		GscSite              *string `json:"gsc_site"`
		GA4Property          *string `json:"ga4_property"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, "invalid request body")
		c.JSON(st, env)
		return
	}
	url := body.URL
	if url == nil {
		url = body.Site
	}
	p, err := h.projects.Update(c.Request.Context(), c.Param("slug"), project.UpdateInput{
		URL: url, Name: body.Name, NoSite: body.NoSite,
		Materials: body.Materials, MaxPages: body.MaxPages,
		GscSite: body.GscSite, GA4Property: body.GA4Property, SearchMode: body.SearchMode, SearchMinImpressions: body.SearchMinImpressions,
	})
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(p))
}

func (h *Handler) auditProject(c *gin.Context) {
	if h.audit == nil {
		writeErr(c, errors.New("audit service not configured"))
		return
	}
	rep, err := h.audit.Run(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(rep))
}

func (h *Handler) getAuditIssues(c *gin.Context) {
	if h.audit == nil {
		writeErr(c, errors.New("audit service not configured"))
		return
	}
	items, refs, err := h.audit.Issues(c.Request.Context(), c.Param("slug"), audit.IssueFilter{
		Severity: c.Query("severity"), Layer: c.Query("layer"), Code: c.Query("code"), Surface: c.Query("surface"),
	})
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"items": items, "references": refs}))
}

func (h *Handler) getAudit(c *gin.Context) {
	if h.audit == nil {
		writeErr(c, errors.New("audit service not configured"))
		return
	}
	rep, err := h.audit.Latest(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writeErr(c, err)
		return
	}
	if rep == nil {
		env, st := resp.Fail(resp.CodeNotFound, http.StatusNotFound, "no audit yet")
		c.JSON(st, env)
		return
	}
	c.JSON(http.StatusOK, resp.OK(rep))
}

func (h *Handler) exportAuditMD(c *gin.Context) {
	if h.audit == nil {
		writeErr(c, errors.New("audit service not configured"))
		return
	}
	slug := c.Param("slug")
	md, err := h.audit.ExportMarkdown(c.Request.Context(), slug)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+slug+`-site-audit.md"`)
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(md))
}

func (h *Handler) getBrand(c *gin.Context) {
	p, revision, err := h.bootstrap.BrandForReview(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writeErr(c, err)
		return
	}
	md, _ := h.bootstrap.Facts(c.Request.Context(), c.Param("slug"))
	c.JSON(http.StatusOK, resp.OK(gin.H{"name": p.Name, "site": p.Site, "brand": p.Brand, "facts_markdown": md, "review_revision": revision}))
}

func (h *Handler) putBrand(c *gin.Context) {
	var body struct {
		Name  string      `json:"name"`
		Brand model.Brand `json:"brand"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, "invalid request body")
		c.JSON(st, env)
		return
	}
	// The brand name is also the project name, and renaming a project is for
	// admins; editors keep the current name.
	if !who(c).admin() {
		p, err := h.projects.Get(c.Request.Context(), c.Param("slug"))
		if err != nil {
			writeErr(c, err)
			return
		}
		body.Name = p.Name
	}
	md, err := h.bootstrap.SaveBrand(c.Request.Context(), c.Param("slug"), body.Name, body.Brand)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"facts_markdown": md}))
}

func (h *Handler) getQuestions(c *gin.Context) {
	if h.bootstrap == nil {
		writeErr(c, errors.New("bootstrap service not configured"))
		return
	}
	qs, err := h.bootstrap.Questions(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writeErr(c, err)
		return
	}
	p, err := h.projects.Get(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"items": qs, "brand": p.Name, "aliases": p.Brand.Aliases, "site": p.Site, "review_revision": model.QuestionsReviewRevision(qs)}))
}

func (h *Handler) putQuestions(c *gin.Context) {
	if h.bootstrap == nil {
		writeErr(c, errors.New("bootstrap service not configured"))
		return
	}
	var body struct {
		Items []model.Question `json:"items"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, "invalid request body")
		c.JSON(st, env)
		return
	}
	if err := h.bootstrap.SaveQuestions(c.Request.Context(), c.Param("slug"), body.Items); err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"ok": true}))
}

func (h *Handler) getCompetitors(c *gin.Context) {
	if h.bootstrap == nil {
		writeErr(c, errors.New("bootstrap service not configured"))
		return
	}
	rows, err := h.bootstrap.Competitors(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"items": rows}))
}

func (h *Handler) putCompetitors(c *gin.Context) {
	if h.bootstrap == nil {
		writeErr(c, errors.New("bootstrap service not configured"))
		return
	}
	var body struct {
		Items []model.Competitor `json:"items"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, "invalid request body")
		c.JSON(st, env)
		return
	}
	if err := h.bootstrap.SaveCompetitors(c.Request.Context(), c.Param("slug"), body.Items); err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"ok": true}))
}

func keyItems() []map[string]any {
	return append(sample.KeyStatus(), webstats.KeyCard())
}

func (h *Handler) getKeys(c *gin.Context) {
	c.JSON(http.StatusOK, resp.OK(gin.H{"items": keyItems()}))
}

func (h *Handler) putKeys(c *gin.Context) {
	var body struct {
		Updates map[string]string `json:"updates"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.Updates) == 0 {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, "updates are required")
		c.JSON(st, env)
		return
	}
	for k, v := range body.Updates {
		if strings.HasSuffix(k, "_BASE") {
			body.Updates[k] = sample.EffectiveBaseUpdate(k, v)
		}
	}
	if err := dotenv.Write(dotenv.FindRoot(), body.Updates); err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"items": keyItems()}))
}

func (h *Handler) verifyKey(c *gin.Context) {
	var body struct {
		Code string `json:"code"`
		Key  string `json:"key"`
		Base string `json:"base"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Code) == "" {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, "code is required")
		c.JSON(st, env)
		return
	}
	if strings.TrimSpace(body.Code) == "google" {
		if err := webstats.Verify(body.Key); err != nil {
			env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, err.Error())
			c.JSON(st, env)
			return
		}
		c.JSON(http.StatusOK, resp.OK(gin.H{"ok": true}))
		return
	}
	if err := sample.Verify(body.Code, body.Key, body.Base); err != nil {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, err.Error())
		c.JSON(st, env)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"ok": true}))
}

func (h *Handler) runWebstats(c *gin.Context) {
	if h.web == nil {
		writeErr(c, errors.New("webstats service not configured"))
		return
	}
	res, err := h.web.Run(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(res))
}

func (h *Handler) getWebstats(c *gin.Context) {
	if h.web == nil {
		writeErr(c, errors.New("webstats service not configured"))
		return
	}
	snap, err := h.web.Snapshot(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(snap))
}

func (h *Handler) listSamples(c *gin.Context) {
	if h.sample == nil {
		writeErr(c, errors.New("sample service not configured"))
		return
	}
	rows, err := h.sample.List(c.Request.Context(), c.Param("slug"), c.Query("platform"), c.Query("qid"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"items": rows}))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (h *Handler) getMeasure(c *gin.Context) {
	if h.sample == nil {
		writeErr(c, errors.New("sample service not configured"))
		return
	}
	view, err := h.sample.Measure(c.Request.Context(), c.Param("slug"), sample.MeasureQuery{
		Access:   c.Query("access"),
		Range:    c.Query("range"),
		Platform: c.Query("platform"),
		Tag:      firstNonEmpty(c.Query("tags"), c.Query("tag")),
		QID:      c.Query("qid"),
		Sort:     c.Query("sort"),
	})
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(view))
}

func (h *Handler) sampleSheet(c *gin.Context) {
	if h.sample == nil {
		writeErr(c, errors.New("sample service not configured"))
		return
	}
	md, err := h.sample.Sheet(c.Request.Context(), c.Param("slug"), c.Query("intent"), 0)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"markdown": md}))
}

func (h *Handler) importSamples(c *gin.Context) {
	if h.sample == nil {
		writeErr(c, errors.New("sample service not configured"))
		return
	}
	var body struct {
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Text) == "" {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, "text is required")
		c.JSON(st, env)
		return
	}
	res, err := h.sample.ImportMarkdown(c.Request.Context(), c.Param("slug"), body.Text)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(res))
}

func (h *Handler) listOpportunities(c *gin.Context) {
	if h.opportunity == nil {
		writeErr(c, errors.New("opportunity service not configured"))
		return
	}
	items, err := h.opportunity.List(c.Request.Context(), c.Param("slug"), opportunity.ListFilter{
		Source: c.Query("source"), Status: c.Query("status"),
	})
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"items": items, "hints": opportunityHints()}))
}

// opportunityHints explains sources that cannot produce items yet.
func opportunityHints() []gin.H {
	for _, p := range sample.Providers {
		if p.Search && sample.Available(p.Code) {
			return []gin.H{}
		}
	}
	return []gin.H{{
		"code": "no_search_engine",
		"text": "AI citation gaps need an engine that searches the web and returns sources, such as Perplexity or Doubao with its content plugin. The engines connected now answer from memory, so this source stays empty.",
	}}
}

func (h *Handler) acceptOpportunity(c *gin.Context) {
	h.materializeOpportunity(c, func(ctx context.Context, slug, key string, reviewed bool) (*model.Task, error) {
		if reviewed && who(c).User != nil {
			return h.opportunity.AcceptReviewed(ctx, slug, key, who(c).User.ID)
		}
		return h.opportunity.Accept(ctx, slug, key)
	})
}

func (h *Handler) dismissOpportunity(c *gin.Context) {
	h.materializeOpportunity(c, func(ctx context.Context, slug, key string, _ bool) (*model.Task, error) {
		return h.opportunity.Dismiss(ctx, slug, key)
	})
}

func (h *Handler) materializeOpportunity(c *gin.Context, do func(ctx context.Context, slug, key string, reviewed bool) (*model.Task, error)) {
	if h.opportunity == nil {
		writeErr(c, errors.New("opportunity service not configured"))
		return
	}
	var body struct {
		Key      string `json:"key"`
		Reviewed bool   `json:"reviewed"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Key) == "" {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, "key is required")
		c.JSON(st, env)
		return
	}
	t, err := do(c.Request.Context(), c.Param("slug"), body.Key, body.Reviewed)
	switch {
	case errors.Is(err, opportunity.ErrAccepted):
		env, st := resp.Fail(resp.CodeConflict, http.StatusConflict, err.Error())
		c.JSON(st, env)
	case errors.Is(err, opportunity.ErrNotFound):
		env, st := resp.Fail(resp.CodeNotFound, http.StatusNotFound, err.Error())
		c.JSON(st, env)
	case err != nil:
		writeErr(c, err)
	default:
		c.JSON(http.StatusOK, resp.OK(t))
	}
}

func (h *Handler) patchTask(c *gin.Context) {
	if h.plan == nil {
		writeErr(c, errors.New("plan service not configured"))
		return
	}
	var body struct {
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Status == "" {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, "status is required")
		c.JSON(st, env)
		return
	}
	t, err := h.plan.SetStatus(c.Request.Context(), c.Param("slug"), c.Param("code"), body.Status, body.Note)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(t))
}

func writeErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, project.ErrInvalidSearchSettings), errors.Is(err, project.ErrNameRequired), errors.Is(err, project.ErrInvalidSlug), errors.Is(err, project.ErrInvalidSite):
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, err.Error())
		c.JSON(st, env)
	case errors.Is(err, project.ErrSlugTaken):
		env, st := resp.Fail(resp.CodeConflict, http.StatusConflict, err.Error())
		c.JSON(st, env)
	case errors.Is(err, project.ErrNotFound), errors.Is(err, jobs.ErrNotFound):
		env, st := resp.Fail(resp.CodeNotFound, http.StatusNotFound, err.Error())
		c.JSON(st, env)
	case errors.Is(err, jobs.ErrBusy):
		env, st := resp.Fail(resp.CodeLocked, http.StatusConflict, err.Error())
		c.JSON(st, env)
	case errors.Is(err, jobs.ErrUnknownAction), errors.Is(err, jobs.ErrNotRunning):
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, err.Error())
		c.JSON(st, env)
	default:
		env, st := resp.Fail(resp.CodeInternal, http.StatusInternalServerError, err.Error())
		c.JSON(st, env)
	}
}

func (h *Handler) listRuns(c *gin.Context) {
	if h.sample == nil {
		writeErr(c, errors.New("sample service not configured"))
		return
	}
	rows, err := h.sample.Runs(c.Request.Context(), c.Param("slug"), 50)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"items": rows}))
}

// retryRun re-asks the failed calls of a run through the job system, so it
// cannot overlap another job on the same project.
func (h *Handler) retryRun(c *gin.Context) {
	if h.jobs == nil {
		writeErr(c, errors.New("jobs not configured"))
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, "invalid run id")
		c.JSON(st, env)
		return
	}
	j, err := h.jobs.Start(c.Request.Context(), c.Param("slug"), "sample", map[string]any{"retry_run": int(id)})
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(j))
}

func (h *Handler) overrideSample(c *gin.Context) {
	if h.sample == nil {
		writeErr(c, errors.New("sample service not configured"))
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, "invalid sample id")
		c.JSON(st, env)
		return
	}
	var body sample.OverrideInput
	if err := c.ShouldBindJSON(&body); err != nil || (body.Mentioned == nil && body.Negative == nil) {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, "mentioned or negative is required")
		c.JSON(st, env)
		return
	}
	sm, err := h.sample.Override(c.Request.Context(), c.Param("slug"), id, body)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(sm))
}
