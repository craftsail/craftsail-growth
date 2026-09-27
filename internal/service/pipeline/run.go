// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"context"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/pkg/httputil"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/audit"
	"github.com/craftsail/craftsail-growth/internal/service/bootstrap"
	"github.com/craftsail/craftsail-growth/internal/service/crawl"
	"github.com/craftsail/craftsail-growth/internal/service/opportunity"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"github.com/craftsail/craftsail-growth/internal/service/report"
	"github.com/craftsail/craftsail-growth/internal/service/sample"
	"github.com/craftsail/craftsail-growth/internal/service/verify"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

type Options struct {
	MaxPages  int
	Limit     int
	NoSample  bool
	SkipLLM   bool
	Scheduled bool // started by the scheduler; recorded on the sample run
}

type Service struct {
	db          *gorm.DB
	Log         func(string)
	projects    *project.Service
	crawl       *crawl.Service
	audit       *audit.Service
	bootstrap   *bootstrap.Service
	sample      *sample.Service
	verify      *verify.Service
	opportunity *opportunity.Service
	report      *report.Service
	web         *webstats.Service
	questions   *repo.Questions
}

func New(db *gorm.DB) *Service {
	boot := bootstrap.New(db)
	boot.LLM = sample.NewAsker()
	return &Service{
		db: db, projects: project.New(db),
		crawl: crawl.New(db, httputil.New()), audit: audit.New(db),
		bootstrap: boot, sample: sample.New(db, sample.NewAsker()),
		verify: verify.New(db), opportunity: opportunity.NewDB(db), report: report.New(db),
		web: webstats.New(db), questions: &repo.Questions{DB: db},
	}
}

func (s *Service) info(msg string) {
	if s.Log != nil {
		s.Log(msg)
	}
}

// NewProject creates a project and runs its first period.
func (s *Service) NewProject(ctx context.Context, in project.CreateInput, opt Options) (string, error) {
	s.info("create project")
	p, err := s.projects.Create(ctx, in)
	if err != nil {
		return "", err
	}
	if err := s.Period(ctx, p.Slug, opt); err != nil {
		return p.Slug, err
	}
	s.info("next: review the derived brand facts and prompts before trusting the numbers")
	return p.Slug, nil
}
