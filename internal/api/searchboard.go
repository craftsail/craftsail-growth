// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"encoding/csv"
	"errors"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/craftsail/craftsail-growth/internal/pkg/resp"
)

func (h *Handler) getKeywords(c *gin.Context) { h.searchList(c, "query") }
func (h *Handler) getGscPages(c *gin.Context) { h.searchList(c, "page") }

func searchInput(c *gin.Context) (webstats.ExploreInput, bool) {
	var in webstats.ExploreInput
	if err := c.ShouldBindQuery(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid search parameters"})
		return in, false
	}
	return in, true
}

func searchError(c *gin.Context, err error) {
	var invalid *webstats.InvalidSearchInput
	if errors.As(err, &invalid) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": invalid.Error()})
		return
	}
	writeErr(c, err)
}

func (h *Handler) searchList(c *gin.Context, kind string) {
	if h.web == nil {
		writeErr(c, errWeb())
		return
	}
	in, ok := searchInput(c)
	if !ok {
		return
	}
	if c.Query("format") == "csv" {
		h.searchCSV(c, kind, in)
		return
	}
	out, err := h.web.ExploreSearch(c.Request.Context(), c.Param("slug"), kind, in)
	if err != nil {
		searchError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(out))
}

func (h *Handler) getSearchDetail(c *gin.Context) {
	if h.web == nil {
		writeErr(c, errWeb())
		return
	}
	in, ok := searchInput(c)
	if !ok {
		return
	}
	out, err := h.web.SearchDetail(c.Request.Context(), c.Param("slug"), c.Query("kind"), in)
	if err != nil {
		searchError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(out))
}

// Spool before sending headers so a failed query never downloads a partial CSV.
func (h *Handler) searchCSV(c *gin.Context, kind string, in webstats.ExploreInput) {
	file, err := os.CreateTemp("", "craftsail-search-*.csv")
	if err != nil {
		writeErr(c, err)
		return
	}
	defer os.Remove(file.Name())
	defer file.Close()
	writer := csv.NewWriter(file)
	var report *webstats.SearchExplore
	err = h.web.ExportSearch(c.Request.Context(), c.Param("slug"), kind, in, func(out *webstats.SearchExplore) error {
		report = out
		return writer.Write([]string{kind, "clicks", "impressions", "ctr", "position", "previous_clicks", "previous_impressions", "previous_ctr", "previous_position", "clicks_change", "from", "through", "previous_from", "previous_through", "covered_days", "previous_covered_days", "comparable", "country", "device", "search_type", "brand"})
	}, func(row webstats.SearchMetric) error {
		n := func(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
		name := csvText(row.Name)
		cells := []string{name, n(row.Clicks), n(row.Impressions), n(row.CTR), n(row.Position), n(row.PreviousClicks), n(row.PreviousImpressions), n(row.PreviousCTR), n(row.PreviousPosition), "", report.Coverage.From, report.Coverage.Through, report.PreviousCoverage.From, report.PreviousCoverage.Through, strconv.Itoa(report.Coverage.CoveredDays), strconv.Itoa(report.PreviousCoverage.CoveredDays), strconv.FormatBool(report.Comparable), report.Filters.Country, report.Filters.Device, report.Filters.SearchType, report.Filters.Brand}
		if report.Comparable {
			cells[9] = n(row.ClicksChange)
		}
		if report.Coverage.State != "covered" && row.CurrentRows == 0 {
			for i := 1; i <= 4; i++ {
				cells[i] = ""
			}
		}
		if report.PreviousCoverage.State != "covered" && row.PreviousRows == 0 {
			for i := 5; i <= 8; i++ {
				cells[i] = ""
			}
		}
		return writer.Write(cells)
	})
	writer.Flush()
	if err == nil {
		err = writer.Error()
	}
	if err != nil {
		searchError(c, err)
		return
	}
	if err := file.Close(); err != nil {
		writeErr(c, err)
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.FileAttachment(file.Name(), "search-"+kind+".csv")
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
