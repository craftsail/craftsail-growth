// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrSyncBudget = errors.New("Google sync batch budget reached; more work remains")

// BatchOptions bounds one turn of a sync job. RefreshBefore stays fixed across
// continuations so already refreshed dates do not starve historical gaps.
type BatchOptions struct {
	RefreshBefore time.Time
	MaxRequests   int
	MaxBatches    int
	MaxDuration   time.Duration
}
type syncBudget struct {
	mu                sync.Mutex
	options           BatchOptions
	requests, batches int
	committed         int
	exhausted         bool
}
type budgetKey struct{}

func withSyncBudget(ctx context.Context, options BatchOptions) (context.Context, context.CancelFunc, *syncBudget) {
	if options.RefreshBefore.IsZero() {
		options.RefreshBefore = time.Now()
	}
	if options.MaxRequests <= 0 {
		options.MaxRequests = 120
	}
	if options.MaxBatches <= 0 {
		options.MaxBatches = 24
	}
	if options.MaxDuration <= 0 {
		options.MaxDuration = 2 * time.Minute
	}
	b := &syncBudget{options: options}
	ctx, cancel := context.WithTimeoutCause(ctx, options.MaxDuration, ErrSyncBudget)
	return context.WithValue(ctx, budgetKey{}, b), cancel, b
}
func budgetFrom(ctx context.Context) *syncBudget {
	b, _ := ctx.Value(budgetKey{}).(*syncBudget)
	return b
}
func takeSyncBudget(ctx context.Context, request bool) error {
	if errors.Is(context.Cause(ctx), ErrSyncBudget) {
		return ErrSyncBudget
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	b := budgetFrom(ctx)
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.exhausted || request && b.requests >= b.options.MaxRequests || !request && b.batches >= b.options.MaxBatches {
		b.exhausted = true
		return ErrSyncBudget
	}
	if request {
		b.requests++
	} else {
		b.batches++
	}
	return nil
}
func syncBudgetError(ctx context.Context, err error) error {
	if errors.Is(context.Cause(ctx), ErrSyncBudget) {
		return ErrSyncBudget
	}
	return err
}
func (b *syncBudget) spent(ctx context.Context) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.exhausted || errors.Is(context.Cause(ctx), ErrSyncBudget)
}

// RunBatch does one bounded turn; callers can persist and resume the same job
// when Pending is true. Google/API errors are returned and never auto-retried.
func (s *Service) RunBatch(ctx context.Context, slug string, options BatchOptions) (*RunResult, error) {
	batchCtx, cancel, budget := withSyncBudget(ctx, options)
	defer cancel()
	result, err := s.run(batchCtx, slug)
	if result == nil && errors.Is(err, context.DeadlineExceeded) && errors.Is(context.Cause(batchCtx), ErrSyncBudget) {
		result = &RunResult{Pending: true}
		err = ErrSyncBudget
	}
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	if result != nil {
		result.Requests = budget.requests
		result.Batches = budget.batches
		result.Pending = result.Pending || budget.spent(batchCtx)
	}
	err = withoutBudgetError(err)
	if err == nil && budget.spent(batchCtx) && budget.committed == 0 {
		err = errors.New("Google sync reached the batch budget without saving a complete report or index result; previous data retained. Automatic continuation stopped")
	}
	return result, err
}

// Keep real failures when another report also exhausted the batch budget.
func withoutBudgetError(err error) error {
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		var kept []error
		for _, child := range joined.Unwrap() {
			if e := withoutBudgetError(child); e != nil {
				kept = append(kept, e)
			}
		}
		return errors.Join(kept...)
	}
	if errors.Is(err, ErrSyncBudget) {
		return nil
	}
	return err
}
