// SPDX-License-Identifier: AGPL-3.0-or-later

package jobs

import (
	"context"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"time"
)

// Deferred ends one bounded turn without finishing the job. The same durable
// job is claimed for a later turn, including after a process restart.
type Deferred struct{ After time.Duration }

func (d *Deferred) Error() string { return "job has more batches to process" }

func (s *Service) ResumeDue(ctx context.Context) { s.resumeDue(ctx, 0) }
func (s *Service) resumeDue(ctx context.Context, onlyID uint64) {
	s.admission.RLock()
	defer s.admission.RUnlock()
	if s.restarting {
		return
	}
	rows, err := s.jobs.Due(ctx, time.Now().Unix(), onlyID)
	if err != nil {
		return
	}
	for _, j := range rows {
		if onlyID != 0 && j.ID != onlyID {
			continue
		}
		s.mu.Lock()
		sp, ok := s.specs[j.Action]
		// An exec can be finishing its cleanup while its row becomes queued.
		if !ok || !sp.Resumable || s.cancels[j.ID] != nil || j.ProjectID == nil {
			s.mu.Unlock()
			continue
		}
		p, err := (&repo.Projects{DB: s.db}).ByID(ctx, *j.ProjectID)
		if err != nil || p == nil {
			s.mu.Unlock()
			continue
		}
		claimed, err := s.jobs.Claim(ctx, j.ID, time.Now().Unix())
		if err != nil || !claimed {
			s.mu.Unlock()
			continue
		}
		jctx, cancel := context.WithCancel(context.Background())
		s.cancels[j.ID] = cancel
		s.mu.Unlock()
		args := j.Args
		if args == nil {
			args = map[string]any{}
		}
		go s.exec(jctx, j.ID, p.Slug, j.Action, args, sp)
	}
}
func (s *Service) StartQueue(ctx context.Context) {
	go func() {
		s.ResumeDue(ctx)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.ResumeDue(ctx)
			}
		}
	}()
}
