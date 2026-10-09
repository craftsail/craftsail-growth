// SPDX-License-Identifier: AGPL-3.0-or-later

package bootstrap

import (
	"context"
	"strings"

	"github.com/craftsail/craftsail-growth/internal/model"
)

// Brand returns the project with its structured brand facts. Placeholder
// values ("TBD" and the legacy Chinese marker) come back empty so a form can
// show them as blanks.
func (s *Service) Brand(ctx context.Context, slug string) (*model.Project, error) {
	p, _, err := s.BrandForReview(ctx, slug)
	return p, err
}

func (s *Service) BrandForReview(ctx context.Context, slug string) (*model.Project, string, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, "", err
	}
	revision := model.BrandReviewRevision(p)
	p.Brand = cleanBrand(p.Brand)
	return p, revision, nil
}

// SaveBrand stores the structured brand facts, then regenerates the facts
// card that llms.txt and JSON-LD are built from. It returns the new card.
// Fields the form does not edit (offers, business goal, uncertain) are kept.
func (s *Service) SaveBrand(ctx context.Context, slug, name string, b model.Brand) (string, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return "", err
	}
	if n := strings.TrimSpace(name); n != "" {
		p.Name = n
	}
	keep := p.Brand
	b = cleanBrand(b)
	if b.Offers == nil {
		b.Offers = keep.Offers
	}
	if b.BusinessGoal == "" {
		b.BusinessGoal = keep.BusinessGoal
	}
	if b.Uncertain == nil {
		b.Uncertain = keep.Uncertain
	}
	p.Brand = b
	if err := s.projRepo.Save(ctx, p); err != nil {
		return "", err
	}
	comps, err := s.competitors.List(ctx, p.ID)
	if err != nil {
		return "", err
	}
	md := RenderFacts(BrandFactsFromProject(p), p.Site, comps)
	if err := s.facts.Upsert(ctx, p.ID, md); err != nil {
		return "", err
	}
	return md, nil
}

func cleanBrand(b model.Brand) model.Brand {
	one := func(s string) string {
		if isPending(s) {
			return ""
		}
		return strings.TrimSpace(s)
	}
	list := func(xs []string) []string {
		out := []string{}
		for _, x := range xs {
			if v := one(x); v != "" {
				out = append(out, v)
			}
		}
		return out
	}
	b.Aliases, b.Products = list(b.Aliases), list(b.Products)
	b.Disambiguation, b.Suitable, b.Unsuitable = list(b.Disambiguation), list(b.Suitable), list(b.Unsuitable)
	b.Industry, b.TargetUsers, b.Definition = one(b.Industry), one(b.TargetUsers), one(b.Definition)
	nums := []model.KeyNumber{}
	for _, n := range b.KeyNumbers {
		n.Fact, n.Value, n.Source = one(n.Fact), one(n.Value), one(n.Source)
		if n.Fact != "" || n.Value != "" {
			nums = append(nums, n)
		}
	}
	b.KeyNumbers = nums
	return b
}
