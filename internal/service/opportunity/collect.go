// SPDX-License-Identifier: AGPL-3.0-or-later

package opportunity

import (
	"context"
	"log"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"github.com/craftsail/craftsail-growth/internal/service/sample"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

type liveSources struct {
	sample   *sample.Service
	audits   *repo.Audits
	web      *webstats.Service
	projects *project.Service
}

// NewLiveSources reads the four sources from the database. A failing source
// is logged and skipped so the others still show.
func NewLiveSources(s *sample.Service, a *repo.Audits, w *webstats.Service, p *project.Service) Sources {
	return liveSources{sample: s, audits: a, web: w, projects: p}
}

func (l liveSources) Collect(ctx context.Context, slug string) ([]Item, error) {
	p, err := l.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	var items []Item
	stage := "new_site"
	if rows, err := l.audits.LatestIssues(ctx, p.ID); err != nil {
		log.Println("opportunity audit source:", err)
	} else {
		items = append(items, FromAudit(rows)...)
	}
	cur, err := l.sample.Measure(ctx, slug, sample.MeasureQuery{Range: "30d"})
	if err != nil {
		log.Println("opportunity citation source:", err)
	} else if cur != nil {
		perPrompt := map[string][2]int{}
		for _, pc := range cur.PromptCharts {
			perPrompt[pc.ID] = [2]int{pc.X, pc.N}
		}
		items = append(items, FromCitation(cur.Opportunities, cur.Access, perPrompt)...)
		if prev, err := l.sample.MeasureWindow(ctx, slug, cur.Access, 60, 30); err == nil && prev != nil {
			items = append(items, FromMetric(cur.Access, prev.VisibilityX, prev.VisibilityN, cur.VisibilityX, cur.VisibilityN)...)
		}
	}
	if board, err := l.web.SearchBoard(ctx, slug); err != nil {
		log.Println("opportunity search source:", err)
	} else if board != nil {
		if board.Observation != nil {
			stage = board.Observation.Mode
		}
		items = append(items, FromSearch(board.Ops, Brand{Name: p.Name, Aliases: p.Brand.Aliases, Site: p.Site})...)
	}
	for i := range items {
		items[i].Stage = stage
	}
	return items, nil
}

// NewDB builds the service with live sources from the database.
func NewDB(db *gorm.DB) *Service {
	projects := project.New(db)
	return New(
		NewLiveSources(sample.New(db, sample.NewAsker()), &repo.Audits{DB: db}, webstats.New(db), projects),
		&repo.Tasks{DB: db},
		func(ctx context.Context, slug string) (uint64, error) {
			p, err := projects.Get(ctx, slug)
			if err != nil {
				return 0, err
			}
			return p.ID, nil
		},
	)
}
