// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"context"
	"fmt"
	"strings"

	"github.com/craftsail/craftsail-growth/internal/service/opportunity"
	"github.com/craftsail/craftsail-growth/internal/service/sample"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

// PeriodStepNames is the single step table for one period.
func PeriodStepNames() []string {
	return []string{"crawl", "audit", "sample", "webstats", "verify", "opportunities", "report"}
}

// Period runs crawl -> audit -> sample -> webstats -> verify ->
// opportunities -> report. Crawl and audit failures stop the period; later
// step failures are collected and returned together.
func (s *Service) Period(ctx context.Context, slug string, opt Options) error {
	var errs []string
	note := func(name string, err error) {
		if err == nil {
			return
		}
		msg := name + ": " + err.Error()
		s.info(msg)
		errs = append(errs, msg)
	}
	names := PeriodStepNames()
	step := 0
	banner := func(name string) {
		step++
		s.info(fmt.Sprintf("=== %d/%d %s ===", step, len(names), name))
	}

	banner("crawl")
	if _, err := s.crawl.Run(ctx, slug, opt.MaxPages); err != nil {
		return err
	}
	banner("audit")
	if _, err := s.audit.Run(ctx, slug); err != nil {
		return err
	}

	banner("sample")
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return err
	}
	qs, _ := s.questions.List(ctx, p.ID)
	if len(qs) == 0 {
		s.info("prompt library is empty; deriving brand facts, competitors and prompts")
		if _, err := s.bootstrap.Run(ctx, slug, opt.SkipLLM); err != nil {
			note("bootstrap", err)
		} else if _, err := s.audit.Run(ctx, slug); err != nil {
			note("re-audit", err)
		}
		qs, _ = s.questions.List(ctx, p.ID)
	}
	switch {
	case opt.NoSample:
		s.info("skipped: --no-sample")
	case len(qs) == 0:
		s.info("skipped: prompt library is empty")
	default:
		runs := p.MonitorRunsPerDay
		if runs <= 0 {
			runs = 3
		}
		trigger := "manual"
		if opt.Scheduled {
			trigger = "schedule"
		}
		if _, err := s.sample.Run(ctx, slug, sample.RunInput{Limit: opt.Limit, FillTo: runs, Trigger: trigger}); err != nil {
			note("sample", err)
		}
	}

	banner("webstats")
	if strings.TrimSpace(p.GscSite) != "" || strings.TrimSpace(p.GA4Property) != "" || (!p.NoSite && strings.TrimSpace(p.Site) != "" && webstats.UserConnected()) {
		if result, err := s.web.Run(ctx, slug); err != nil {
			s.web.RecordFailure(ctx, slug, err)
			note("webstats", err)
		} else if result != nil && result.Pending {
			s.info("Google sync batch saved; use Sync Google to continue importing the remaining history")
		}
	} else {
		s.info("skipped: Google is not connected for this project")
	}

	banner("verify")
	if _, err := s.verify.Run(ctx, slug, false); err != nil {
		note("verify", err)
	}
	banner("opportunities")
	if items, err := s.opportunity.List(ctx, slug, opportunity.ListFilter{}); err != nil {
		note("opportunities", err)
	} else {
		s.info(fmt.Sprintf("%d opportunities", len(items)))
	}
	banner("report")
	if _, err := s.report.Build(ctx, slug); err != nil {
		note("report", err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	s.info("period complete")
	return nil
}
