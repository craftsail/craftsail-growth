// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"fmt"
	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
	"gorm.io/gorm"
	"strings"
	"time"
)

// Offline fixtures are explicitly identified in every project screen. They do
// not contact Google, create an activation event, or mutate an existing DB.
func seedSearchScenarios(ctx context.Context, db *gorm.DB, now time.Time) error {
	end, _ := webstats.FinalizedThrough(now, "")
	from := end.AddDate(0, 0, -89)
	r := &repo.Webstats{DB: db}
	projects := project.New(db)
	for _, kind := range []string{"new_site", "established"} {
		p, err := projects.Create(ctx, project.CreateInput{Name: "Quillpad " + kind, Slug: "quillpad-" + strings.ReplaceAll(kind, "_", "-"), URL: siteURL})
		if err != nil {
			return err
		}
		p.DemoScenario = kind
		p.GscSite = "sc-domain:quillpad.example"
		p.GoogleHistoryStart = from.Format("2006-01-02")
		if err = projects.Save(ctx, p); err != nil {
			return err
		}
		if err = r.ActivateProperty(ctx, p.ID, "gsc", p.GscSite, ""); err != nil {
			return err
		}
		batches := map[string]repo.SyncBatch{}
		quality := model.GoogleQuality{Known: true, Aggregations: []string{"byProperty"}}
		daily := repo.SyncBatch{Quality: quality}
		for d := from; !d.After(end); d = d.AddDate(0, 0, 1) {
			clicks, impr := 20.0, 1000.0
			if !d.Before(end.AddDate(0, 0, -27)) {
				clicks = 5
			}
			if kind == "new_site" {
				clicks = 0
				impr = 8
				if d.Weekday() == time.Monday {
					clicks = 1
				}
			}
			daily.GSCDaily = append(daily.GSCDaily, model.GscDaily{ProjectID: p.ID, Property: p.GscSite, SearchType: "web", Day: d, Clicks: clicks, Impressions: impr, CTR: clicks / impr, Position: 5, FetchedAt: now.Unix()})
			for _, slice := range []string{"page", "query", "query_page", "country", "device", "country_device", "page_country_device", "query_country_device", "query_page_country_device"} {
				batch := batches[slice]
				batch.Quality = quality
				fact := model.GscFact{ProjectID: p.ID, Property: p.GscSite, SearchType: "web", Slice: slice, Day: d, Clicks: clicks, Impressions: impr, CTR: clicks / impr, Position: 5, FetchedAt: now.Unix()}
				switch slice {
				case "page", "page_country_device":
					fact.Page = siteURL + "/blog/remote-team-rituals.html"
				case "query", "query_country_device":
					fact.Query = "meeting notes for remote teams"
				case "query_page", "query_page_country_device":
					fact.Query = "meeting notes for remote teams"
					fact.Page = siteURL + "/blog/remote-team-rituals.html"
				}
				if slice == "country" || slice == "country_device" || slice == "page_country_device" || slice == "query_country_device" || slice == "query_page_country_device" {
					fact.Country = "bra"
				}
				if slice == "device" || slice == "country_device" || slice == "page_country_device" || slice == "query_country_device" || slice == "query_page_country_device" {
					fact.Device = "mobile"
				}
				if fact.Page != "" {
					batch.Quality.Aggregations = []string{"byPage"}
				}
				batch.GSC = append(batch.GSC, fact)
				batches[slice] = batch
			}
		}
		batches["daily"] = daily
		for name, batch := range batches {
			state := model.WebSyncReport{ProjectID: p.ID, Source: "gsc", Property: p.GscSite, Report: name, SearchType: "web", Version: webstats.SyncRequestVersion, Token: fmt.Sprintf("demo-%d-%s", p.ID, name), From: from, Through: end, UpdatedAt: now.Unix()}
			if err = r.BeginSyncReport(ctx, &state); err != nil {
				return err
			}
			if err = r.ReplaceSyncBatch(ctx, state, from, end, batch); err != nil {
				return err
			}
			if err = r.FinishSyncReport(ctx, state, "completed", ""); err != nil {
				return err
			}
		}
		if err = r.UpsertImport(ctx, &model.WebImport{ProjectID: p.ID, Source: "gsc", Property: p.GscSite, State: "completed", FinalizedThrough: &end, CursorDate: &end, BoundarySource: "demo", UpdatedAt: now.Unix()}); err != nil {
			return err
		}
		totals := webstats.WindowFromGSC(daily.GSCDaily, end, 28)
		if err = r.UpsertWindow(ctx, &model.WebWindow{ProjectID: p.ID, Source: "gsc", Property: p.GscSite, WindowDays: 28, FinalizedThrough: end, Clicks: totals.Clicks, Impressions: totals.Impressions, PreviousClicks: totals.PreviousClicks, PreviousImpressions: totals.PreviousImpressions, CoveredDays: totals.CoveredDays, ComputedAt: now.Unix()}); err != nil {
			return err
		}
		if kind == "established" {
			if err = r.PromoteSearchStage(ctx, p.ID, p.GscSite, end, webstats.SyncRequestVersion); err != nil {
				return err
			}
		}
	}
	return nil
}
