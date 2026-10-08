// SPDX-License-Identifier: AGPL-3.0-or-later

package opportunity

import (
	"context"
	"errors"
	"github.com/craftsail/craftsail-growth/internal/model"
	"math"
	"time"
)

type scoreStore interface {
	OpportunityScores(context.Context, uint64) ([]model.OpportunityScore, error)
	SaveOpportunityScore(context.Context, *model.OpportunityScore) error
}

var ErrScore = errors.New("impact, confidence and ease must be 1–10; effort hours must be 0–10000")

func (s *Service) Score(ctx context.Context, slug string, row model.OpportunityScore) error {
	if row.Impact < 1 || row.Impact > 10 || row.Confidence < 1 || row.Confidence > 10 || row.Ease < 1 || row.Ease > 10 || row.EffortHours < 0 || row.EffortHours > 10000 || math.IsNaN(row.EffortHours) || math.IsInf(row.EffortHours, 0) {
		return ErrScore
	}
	items, err := s.List(ctx, slug, ListFilter{})
	if err != nil {
		return err
	}
	found := false
	for _, it := range items {
		if it.Key == row.Key {
			found = true
			break
		}
	}
	if !found {
		return ErrNotFound
	}
	pid, err := s.projectID(ctx, slug)
	if err != nil {
		return err
	}
	store, ok := s.tasks.(scoreStore)
	if !ok {
		return errors.New("score storage unavailable")
	}
	row.ID = 0
	row.ProjectID = pid
	row.UpdatedAt = time.Now().Unix()
	return store.SaveOpportunityScore(ctx, &row)
}
func (s *Service) rank(ctx context.Context, pid uint64, items []Item) error {
	if store, ok := s.tasks.(scoreStore); ok {
		scores, err := store.OpportunityScores(ctx, pid)
		if err != nil {
			return err
		}
		by := map[string]model.OpportunityScore{}
		for _, row := range scores {
			by[row.Key] = row
		}
		for i := range items {
			if score, ok := by[items[i].Key]; ok {
				items[i].Score = &score
			}
		}
	}
	Sort(items)
	count := 0
	for i := range items {
		if count < 3 && (items[i].Status == "" || items[i].Status == model.TaskOpen || items[i].Status == model.TaskDoing || items[i].Status == model.TaskRegressed) {
			items[i].Recommended = true
			count++
		}
	}
	return nil
}
