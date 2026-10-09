// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"context"
	"fmt"
)

// FirstCheck saves technical evidence before optional AI or Google work. It
// deliberately does not draft or overwrite a project's brand and questions.
func (s *Service) FirstCheck(ctx context.Context, slug string, maxPages int) error {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return err
	}
	if p.NoSite || p.Site == "" {
		s.info("No website: add brand materials and review questions before optional AI sampling")
		return nil
	}
	// A first pass is bounded; a full crawl remains available separately.
	if maxPages <= 0 || maxPages > 5 {
		maxPages = 5
	}
	s.info("=== 1/2 crawl ===")
	result, err := s.crawl.Run(ctx, slug, maxPages)
	if err != nil && result == nil {
		return fmt.Errorf("crawl: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.info("=== 2/2 audit ===")
	if _, err := s.audit.Run(ctx, slug); err != nil {
		return fmt.Errorf("audit: %w", err)
	}
	if err != nil {
		return fmt.Errorf("%w; the audit contains access findings to review", err)
	}
	s.info("Technical check saved; review the audit evidence, then confirm brand facts and questions")
	return nil
}
