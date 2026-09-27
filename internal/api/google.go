// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/craftsail/craftsail-growth/internal/pkg/dotenv"
	"github.com/craftsail/craftsail-growth/internal/pkg/resp"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

func (h *Handler) googleConnect(c *gin.Context) {
	if !webstats.OAuthConfigured() {
		dest := "/settings?google=need-client"
		if slug := strings.TrimSpace(c.Query("slug")); slug != "" {
			dest = "/p/" + url.PathEscape(slug) + "/settings/google?google=need-client"
		}
		c.Redirect(http.StatusFound, dest)
		return
	}
	nonce := randomNonce()
	slug := strings.TrimSpace(c.Query("slug"))
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "craftsail_growth_google_state",
		Value:    nonce,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})
	state := nonce
	if slug != "" {
		state = nonce + "." + slug
	}
	c.Redirect(http.StatusFound, webstats.AuthCodeURL(os.Getenv("GOOGLE_OAUTH_CLIENT_ID"), googleRedirect(c), state))
}

func (h *Handler) googleCallback(c *gin.Context) {
	if msg := strings.TrimSpace(c.Query("error")); msg != "" {
		c.String(http.StatusBadRequest, "Google authorization did not complete: %s", msg)
		return
	}
	cookie, err := c.Cookie("craftsail_growth_google_state")
	if err != nil || cookie == "" {
		c.String(http.StatusBadRequest, "The authorization state expired. Start the Google connection again from Settings.")
		return
	}
	state := c.Query("state")
	nonce, slug, _ := strings.Cut(state, ".")
	if nonce == "" || nonce != cookie {
		c.String(http.StatusBadRequest, "The authorization state does not match. Start the Google connection again from Settings.")
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{Name: "craftsail_growth_google_state", Value: "", Path: "/", MaxAge: -1})
	tok, err := (&webstats.Client{}).ExchangeCode(
		os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
		os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
		c.Query("code"),
		googleRedirect(c),
	)
	if err != nil {
		c.String(http.StatusBadGateway, "Could not exchange the Google token: %s", err.Error())
		return
	}
	if tok.RefreshToken != "" {
		if err := dotenv.Write(h.EnvRoot, map[string]string{"GOOGLE_REFRESH_TOKEN": tok.RefreshToken}); err != nil {
			c.String(http.StatusInternalServerError, "Could not save the authorization: %s", err.Error())
			return
		}
	}
	dest := "/settings"
	if slug != "" {
		dest = "/p/" + url.PathEscape(slug) + "/settings/google?pick=1"
	}
	c.Redirect(http.StatusFound, dest)
}

func (h *Handler) googleStatus(c *gin.Context) {
	slug := c.Param("slug")
	p, err := h.projects.Get(c.Request.Context(), slug)
	if err != nil || p == nil {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusNotFound, "project not found")
		c.JSON(st, env)
		return
	}
	out := gin.H{
		"oauth_configured": webstats.OAuthConfigured(),
		"connected":        webstats.UserConnected(),
		"needs_reconnect":  false,
		"email":            "",
		"redirect_uri":     googleRedirect(c),
		"gsc_site":         p.GscSite,
		"ga4_property":     p.GA4Property,
		"gsc_choices":      []webstats.GSCChoice{},
		"ga_choices":       []webstats.GAChoice{},
	}
	if !webstats.UserConnected() {
		c.JSON(http.StatusOK, resp.OK(out))
		return
	}
	tok, err := (&webstats.Client{}).UserAccessToken()
	if err != nil {
		out["needs_reconnect"] = true
		out["connected"] = false
		c.JSON(http.StatusOK, resp.OK(out))
		return
	}
	out["email"] = googleEmail(tok)
	if c.Query("pick") == "1" || strings.TrimSpace(p.GscSite) == "" {
		gsc, ga, listErr := (&webstats.Client{}).ListChoices(tok, h.projectSite(c, slug))
		if gsc != nil {
			out["gsc_choices"] = gsc
		}
		if ga != nil {
			out["ga_choices"] = ga
		}
		if listErr != nil {
			out["list_error"] = listErr.Error()
		}
	}
	c.JSON(http.StatusOK, resp.OK(out))
}

func (h *Handler) googleSave(c *gin.Context) {
	slug := c.Param("slug")
	var body struct {
		GscSite     string `json:"gsc_site"`
		GA4Property string `json:"ga4_property"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, "invalid request")
		c.JSON(st, env)
		return
	}
	tok, err := (&webstats.Client{}).UserAccessToken()
	if err != nil {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, "Google needs to be reconnected")
		c.JSON(st, env)
		return
	}
	gscChoices, gaChoices, _ := (&webstats.Client{}).ListChoices(tok, h.projectSite(c, slug))
	in := project.UpdateInput{}
	if body.GscSite != "" {
		ok := false
		for _, ch := range gscChoices {
			if ch.SiteURL == body.GscSite && ch.Selectable {
				ok = true
				break
			}
		}
		if !ok {
			env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, "this property is not available to the connected Google account")
			c.JSON(st, env)
			return
		}
		site := body.GscSite
		in.GscSite = &site
	}
	if body.GA4Property != "" {
		ok := false
		for _, ch := range gaChoices {
			if ch.ID == body.GA4Property {
				ok = true
				break
			}
		}
		if !ok {
			env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, "this GA4 property is not available to the connected Google account")
			c.JSON(st, env)
			return
		}
		id := body.GA4Property
		in.GA4Property = &id
	}
	if in.GscSite == nil && in.GA4Property == nil {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, "choose a property first")
		c.JSON(st, env)
		return
	}
	p, err := h.projects.Update(c.Request.Context(), slug, in)
	if err != nil {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusBadRequest, err.Error())
		c.JSON(st, env)
		return
	}
	c.JSON(http.StatusOK, resp.OK(p))
}

func (h *Handler) googleDisconnect(c *gin.Context) {
	if err := dotenv.Write(h.EnvRoot, map[string]string{"GOOGLE_REFRESH_TOKEN": ""}); err != nil {
		env, st := resp.Fail(resp.CodeBadRequest, http.StatusInternalServerError, err.Error())
		c.JSON(st, env)
		return
	}
	c.JSON(http.StatusOK, resp.OK(gin.H{"ok": true}))
}

func googleEmail(token string) string {
	b, err := (&webstats.Client{}).UserInfo(token)
	if err != nil {
		return ""
	}
	var obj struct {
		Email string `json:"email"`
	}
	_ = json.Unmarshal(b, &obj)
	return obj.Email
}

func (h *Handler) projectSite(c *gin.Context, slug string) string {
	p, err := h.projects.Get(c.Request.Context(), slug)
	if err != nil || p == nil || p.NoSite {
		return ""
	}
	return strings.TrimSpace(p.Site)
}

func googleRedirect(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host + "/api/google/callback"
}

func randomNonce() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
