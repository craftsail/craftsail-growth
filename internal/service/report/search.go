// SPDX-License-Identifier: AGPL-3.0-or-later

package report

import (
	"context"
	"fmt"
	"strings"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

// writeWebInsight: Google data is supporting context. It never feeds the AI
// visibility numbers above.
func (s *Service) writeWebInsight(ctx context.Context, b *strings.Builder, p *model.Project) {
	payload, err := webstats.New(s.db).Insight(ctx, p)
	if err != nil || payload == nil || (payload["gsc_clicks"] == (*float64)(nil) && payload["ga_sessions"] == (*float64)(nil)) {
		s.writeMonitor(ctx, b, p.ID)
		return
	}
	b.WriteString("## Google Search and GA4 (supporting data)\n\n")
	fmt.Fprintf(b, "- Window: %s to %s\n", insightStr(payload["from"]), insightStr(payload["to"]))
	fmt.Fprintf(b, "- Search Console: %s impressions, %s clicks\n", insightNum(payload["gsc_impressions"]), insightNum(payload["gsc_clicks"]))
	fmt.Fprintf(b, "- GA4: %s sessions, %s key events; sessions from AI referrers at least %s (in-app opens often send no referrer)\n",
		insightNum(payload["ga_sessions"]), insightNum(payload["ga_key_events"]), insightNum(payload["ai_sessions"]))
	b.WriteString("- Clicks, impressions and sessions are official daily totals. AI-referrer sessions come from source rows and are a lower bound.\n")
	s.writeMonitor(ctx, b, p.ID)
}

func (s *Service) writeMonitor(ctx context.Context, b *strings.Builder, projectID uint64) {
	rows := &repo.Webstats{DB: s.db}
	wrote := false
	for _, source := range []string{"gsc", "ga4"} {
		imp, err := rows.GetImport(ctx, projectID, source)
		if err != nil || imp == nil || imp.Property == "" {
			continue
		}
		win, err := rows.LatestWindow(ctx, projectID, source, imp.Property)
		if err != nil || win == nil {
			continue
		}
		if !wrote {
			b.WriteString("\n### Import status\n\n")
			wrote = true
		}
		name, calendar := "Search Console", "Pacific time"
		if source == "ga4" {
			name, calendar = "GA4", "property time zone"
		}
		through := ""
		if imp.FinalizedThrough != nil {
			through = imp.FinalizedThrough.UTC().Format("2006-01-02")
		}
		fmt.Fprintf(b, "- %s property `%s`: final data through %s (%s). %d of the last %d days have daily totals.\n",
			name, imp.Property, through, calendar, win.CoveredDays, win.WindowDays)
		if source == "gsc" {
			fmt.Fprintf(b, "- Daily totals: %s impressions, %s clicks. Previous window: %s impressions, %s clicks.\n",
				insightNum(win.Impressions), insightNum(win.Clicks), insightNum(win.PreviousImpressions), insightNum(win.PreviousClicks))
		} else {
			fmt.Fprintf(b, "- Daily totals: %s sessions. Previous window: %s sessions.\n", insightNum(win.Sessions), insightNum(win.PreviousSessions))
		}
	}
	if wrote {
		b.WriteString("- Totals come from date-only requests, never from summing query rows.\n\n")
	} else {
		b.WriteString("\n")
	}
}

func (s *Service) writeCrawlSearch(ctx context.Context, b *strings.Builder, p *model.Project) {
	board, _ := webstats.New(s.db).SearchBoard(ctx, p.Slug)
	writeSearchSection(b, board)
}

func writeSearchSection(b *strings.Builder, board *webstats.SearchBoard) {
	if board == nil || (!board.Period.Measured && len(board.Keywords) == 0) {
		return
	}
	b.WriteString("### Search Console, last 28 days\n\n")
	b.WriteString("> Query rows exclude anonymized queries, so they add up to less than the daily totals.\n\n")
	if board.Period.Measured {
		fmt.Fprintf(b, "%.0f clicks, %.0f impressions, average position %.1f, CTR %.2f%%",
			board.Period.Clicks, board.Period.Impressions, board.Period.Position, board.Period.CTR*100)
		if board.Period.HasPrevious && board.Period.ClicksDelta != nil {
			fmt.Fprintf(b, "; clicks %+.1f%% versus the previous window", *board.Period.ClicksDelta)
		}
		b.WriteString(".\n\n")
	}
	if len(board.Keywords) > 0 {
		b.WriteString("| Query | Position | Clicks | Impressions |\n|---|---:|---:|---:|\n")
		for i, k := range board.Keywords {
			if i >= 10 {
				break
			}
			fmt.Fprintf(b, "| %s | %.1f | %.0f | %.0f |\n", cell(k.Query), k.Position, k.Clicks, k.Impressions)
		}
		b.WriteString("\n")
	}
}
