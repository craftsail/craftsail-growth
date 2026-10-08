// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/craftsail/craftsail-growth/internal/pkg/resp"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
	"github.com/gin-gonic/gin"
)

func (h *Handler) getGAChannels(c *gin.Context) { h.gaList(c, "channel") }
func (h *Handler) getGALandings(c *gin.Context) { h.gaList(c, "landing") }
func (h *Handler) gaList(c *gin.Context, report string) {
	if h.web == nil {
		writeErr(c, errWeb())
		return
	}
	in, ok := searchInput(c)
	if !ok {
		return
	}
	if c.Query("format") == "csv" {
		h.gaCSV(c, report, in)
		return
	}
	out, err := h.web.ExploreGA(c.Request.Context(), c.Param("slug"), report, in)
	if err != nil {
		searchError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(out))
}
func csvText(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if strings.ContainsAny(value[:min(1, len(value))], "\t\r\n") || (len(trimmed) > 0 && strings.ContainsAny(trimmed[:1], "=+-@")) {
		return "'" + value
	}
	return value
}
func (h *Handler) gaCSV(c *gin.Context, report string, in webstats.ExploreInput) {
	file, err := os.CreateTemp("", "craftsail-ga-*.csv")
	if err != nil {
		writeErr(c, err)
		return
	}
	defer os.Remove(file.Name())
	defer file.Close()
	writer := csv.NewWriter(file)
	var context *webstats.GAExplore
	err = h.web.ExportGA(c.Request.Context(), c.Param("slug"), report, in, func(out *webstats.GAExplore) error {
		context = out
		return writer.Write([]string{"channel", "source", "medium", "landing", "sessions", "engaged_sessions", "engagement_rate", "key_events", "engagement_seconds", "seconds_per_session", "previous_sessions", "sessions_change", "from", "through", "previous_from", "previous_through", "covered_days", "previous_covered_days", "comparable", "property", "timezone", "quality", "previous_quality"})
	}, func(row webstats.GAMetric) error {
		number := func(n float64) string { return strconv.FormatFloat(n, 'f', -1, 64) }
		optional := func(n *float64) string {
			if n == nil {
				return ""
			}
			return number(*n)
		}
		quality, _ := json.Marshal(context.Quality)
		previousQuality, _ := json.Marshal(context.PreviousQuality)
		cells := []string{csvText(row.Channel), csvText(row.Source), csvText(row.Medium), csvText(row.Landing), number(row.Sessions), number(row.Engaged), optional(row.EngagementRate), number(row.KeyEvents), optional(row.Duration), optional(row.DurationPerSession), number(row.PreviousSessions), "", context.Coverage.From, context.Coverage.Through, context.PreviousCoverage.From, context.PreviousCoverage.Through, strconv.Itoa(context.Coverage.CoveredDays), strconv.Itoa(context.PreviousCoverage.CoveredDays), strconv.FormatBool(context.Comparable), context.Property, context.Timezone, string(quality), string(previousQuality)}
		if context.Comparable {
			cells[11] = number(row.SessionsChange)
		}
		if (context.Coverage.State != "covered" || !context.Quality.Known || context.Quality.Sampled || context.Quality.Thresholded || context.Quality.OtherRow || context.Quality.Restricted || context.Quality.EmptyReason) && row.CurrentRows == 0 {
			for i := 4; i <= 9; i++ {
				cells[i] = ""
			}
		}
		if (context.PreviousCoverage.State != "covered" || !context.PreviousQuality.Known || context.PreviousQuality.Sampled || context.PreviousQuality.Thresholded || context.PreviousQuality.OtherRow || context.PreviousQuality.Restricted || context.PreviousQuality.EmptyReason) && row.PreviousRows == 0 {
			cells[10] = ""
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
	c.FileAttachment(file.Name(), "ga-"+report+".csv")
}
