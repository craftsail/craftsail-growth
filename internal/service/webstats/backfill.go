// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
)

type dateChunk struct {
	start time.Time
	end   time.Time
}

func chunks(source string, lastEnd, now time.Time) []dateChunk {
	now = dateOnly(now)
	endCap := now.AddDate(0, 0, -3)
	start := now.AddDate(0, -16, 0)
	if source == "ga" || strings.HasPrefix(source, "ga/") {
		endCap = now.AddDate(0, 0, -1)
		start = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if !lastEnd.IsZero() {
		inc := endCap.AddDate(0, 0, -9)
		if inc.After(endCap) {
			return nil
		}
		return []dateChunk{{start: inc, end: endCap}}
	}
	var out []dateChunk
	for cur := start; !cur.After(endCap); {
		end := cur.AddDate(0, 0, 6)
		if end.After(endCap) {
			end = endCap
		}
		out = append(out, dateChunk{start: cur, end: end})
		cur = end.AddDate(0, 0, 1)
	}
	return out
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func (s *Service) SyncFacts(ctx context.Context, projectID uint64, token, gscSite, gaProp string, now time.Time) (string, error) {
	var notes []string
	gscKey, _ := GSCPropertyKey(gscSite)
	gaKey, _ := GAPropertyKey(gaProp)
	if gscSite != "" {
		for _, slice := range gscSlices() {
			for _, searchType := range slice.Types {
				key := "gsc/" + slice.Name + "/" + searchType
				last, err := s.rows.GetSync(ctx, projectID, key)
				if err != nil {
					return strings.Join(notes, "；"), err
				}
				parts := chunks("gsc", last, now)
				if slice.Name == "hour" {
					parts = chunks("gsc", now, now)
				}
				for _, part := range parts {
					rows, err := s.client().FetchGSCFacts(ctx, token, gscSite, slice.Name, searchType, part.start.Format("2006-01-02"), part.end.Format("2006-01-02"))
					if err != nil {
						if errors.Is(err, ErrSliceUnsupported) {
							notes = append(notes, key+" is not supported")
							break
						}
						notes = append(notes, key+": "+err.Error())
						break
					}
					for i := range rows {
						rows[i].ProjectID = projectID
						rows[i].Property = gscKey
						rows[i].FetchedAt = now.Unix()
					}
					if err := s.rows.UpsertGscFacts(ctx, rows); err != nil {
						return strings.Join(notes, "；"), err
					}
					if err := s.rows.PutSync(ctx, projectID, key, part.end); err != nil {
						return strings.Join(notes, "；"), err
					}
				}
			}
		}
	}
	if gaProp != "" {
		for _, report := range []string{"session", "page", "event", "hour"} {
			key := "ga/" + report
			last, err := s.rows.GetSync(ctx, projectID, key)
			if err != nil {
				return strings.Join(notes, "；"), err
			}
			parts := chunks("ga", last, now)
			if report == "hour" {
				parts = chunks("ga", now, now)
			}
			for _, part := range parts {
				rows, err := s.client().FetchGAFacts(ctx, token, gaProp, report, part.start.Format("2006-01-02"), part.end.Format("2006-01-02"))
				if err != nil {
					notes = append(notes, key+": "+err.Error())
					break
				}
				for i := range rows {
					rows[i].ProjectID = projectID
					rows[i].Property = gaKey
					rows[i].FetchedAt = now.Unix()
				}
				if err := s.rows.UpsertGaFacts(ctx, rows); err != nil {
					return strings.Join(notes, "；"), err
				}
				if err := s.rows.PutSync(ctx, projectID, key, part.end); err != nil {
					return strings.Join(notes, "；"), err
				}
			}
		}
	}
	return strings.Join(notes, "；"), nil
}

func (s *Service) archivePage(ctx context.Context, projectID uint64, source, report, request, body string, now time.Time) {
	if strings.TrimSpace(body) == "" {
		return
	}
	_ = s.rows.AppendRaw(ctx, &model.GoogleRaw{
		ProjectID: projectID,
		Source:    source,
		Report:    report,
		Request:   request,
		Body:      body,
		FetchedAt: now.Unix(),
	})
}
