// SPDX-License-Identifier: AGPL-3.0-or-later

package project

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/pkg/slug"
	"github.com/craftsail/craftsail-growth/internal/repo"
)

var (
	ErrNameRequired          = errors.New("a project without a site needs a brand name")
	ErrSlugTaken             = errors.New("project slug is taken")
	ErrInvalidSlug           = errors.New("invalid project slug")
	ErrNotFound              = errors.New("project not found")
	ErrInvalidSearchSettings = errors.New("search mode must be auto, new_site or established; minimum impressions must be 100–1000000")
	ErrInvalidSite           = errors.New("invalid website URL")
)

type CreateInput struct {
	URL       string
	Name      string
	Slug      string
	NoSite    bool
	Materials string
	MaxPages  int
	Force     bool
}

type Service struct {
	projects *repo.Projects
}

func New(db *gorm.DB) *Service {
	return &Service{projects: &repo.Projects{DB: db}}
}

func (s *Service) List(ctx context.Context) ([]model.Project, error) {
	return s.projects.List(ctx)
}

func (s *Service) Save(ctx context.Context, p *model.Project) error {
	return s.projects.Save(ctx, p)
}

// ByID returns the project with that ID, or ErrNotFound.
func (s *Service) ByID(ctx context.Context, id uint64) (*model.Project, error) {
	p, err := s.projects.ByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrNotFound
	}
	return p, nil
}

func (s *Service) Get(ctx context.Context, sl string) (*model.Project, error) {
	if !slug.Valid(sl) {
		return nil, ErrInvalidSlug
	}
	p, err := s.projects.BySlug(ctx, sl)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrNotFound
	}
	return p, nil
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*model.Project, error) {
	in.URL = strings.TrimSpace(in.URL)
	in.Name = strings.TrimSpace(in.Name)
	in.Slug = strings.TrimSpace(in.Slug)
	if in.MaxPages <= 0 {
		in.MaxPages = 25
	}

	noSite := in.NoSite || in.URL == ""
	site := ""
	if noSite {
		if in.Name == "" {
			return nil, ErrNameRequired
		}
	} else {
		var err error
		site, err = normalizeSite(in.URL)
		if err != nil {
			return nil, err
		}
		in.URL = site
	}

	gotSlug := in.Slug
	if gotSlug == "" {
		if noSite {
			gotSlug = slug.Slugify(in.Name)
		} else {
			host := strings.TrimPrefix(hostOf(site), "www.")
			label := host
			if i := strings.Index(host, "."); i > 0 {
				label = host[:i]
			}
			gotSlug = slug.Slugify(label)
		}
	}
	if !slug.Valid(gotSlug) {
		return nil, ErrInvalidSlug
	}

	name := in.Name
	if name == "" {
		name = gotSlug
	}

	existing, err := s.projects.BySlug(ctx, gotSlug)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if !in.Force {
			return nil, fmt.Errorf("%w：%s", ErrSlugTaken, gotSlug)
		}
		if err := s.projects.DeleteBySlug(ctx, gotSlug); err != nil {
			return nil, err
		}
	}

	p := &model.Project{
		Slug:      gotSlug,
		Name:      name,
		Site:      site,
		Market:    model.MarketAll,
		NoSite:    noSite,
		Brand:     model.Brand{Aliases: []string{}, Products: []string{}},
		Platforms: model.DefaultPlatforms(),
		PagesSeed: []string{},
		PagesMax:  in.MaxPages,
		Targets:   model.DefaultTargets(),
		Materials: in.Materials,
		Notes:     "prompts, competitors and aliases are filled by bootstrap",
	}
	if err := s.projects.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

type UpdateInput struct {
	SearchMode           *string
	SearchMinImpressions *int
	URL                  *string
	Name                 *string
	NoSite               *bool
	Materials            *string
	MaxPages             *int
	GscSite              *string
	GA4Property          *string
}

func (s *Service) Update(ctx context.Context, sl string, in UpdateInput) (*model.Project, error) {
	p, err := s.Get(ctx, sl)
	if err != nil {
		return nil, err
	}
	if in.SearchMode != nil {
		switch *in.SearchMode {
		case "auto", "new_site", "established":
			p.SearchMode = *in.SearchMode
		default:
			return nil, ErrInvalidSearchSettings
		}
	}
	if in.SearchMinImpressions != nil {
		if *in.SearchMinImpressions < 100 || *in.SearchMinImpressions > 1000000 {
			return nil, ErrInvalidSearchSettings
		}
		p.SearchMinImpressions = *in.SearchMinImpressions
	}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, ErrNameRequired
		}
		p.Name = name
	}
	if in.Materials != nil {
		p.Materials = *in.Materials
	}
	if in.MaxPages != nil && *in.MaxPages > 0 {
		p.PagesMax = *in.MaxPages
	}
	if in.GscSite != nil {
		p.GscSite = strings.TrimSpace(*in.GscSite)
	}
	if in.GA4Property != nil {
		p.GA4Property = strings.TrimPrefix(strings.TrimSpace(*in.GA4Property), "properties/")
	}
	if in.NoSite != nil && *in.NoSite {
		p.NoSite = true
		p.Site = ""
	} else if in.URL != nil {
		site, err := normalizeSite(*in.URL)
		if err != nil {
			return nil, err
		}
		if site == "" {
			if p.Name == "" {
				return nil, ErrNameRequired
			}
			p.NoSite = true
			p.Site = ""
		} else {
			p.NoSite = false
			p.Site = site
		}
	}
	if p.NoSite && strings.TrimSpace(p.Name) == "" {
		return nil, ErrNameRequired
	}
	if err := s.projects.Save(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func normalizeSite(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", ErrInvalidSite
	}
	return strings.TrimRight(raw, "/"), nil
}

func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return u.Hostname()
}
