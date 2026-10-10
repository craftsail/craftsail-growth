# Playbook Status Implementation Plan (Phase 2)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The help-center playbooks show the current project's state: which SEO and AI stage it is in, ✓/✗ on each step that can be checked, and how many pass signals are met. The user can choose the product stage and confirm manual signals there. The overview only gets a small link.

**Architecture:** A new read-only service `internal/service/playbook` gathers facts from existing services (project, audit, sample, plan, report, webstats) into a plain `Facts` struct, and pure functions in `judge.go` turn facts into a `Status`. Two small writes (product stage, manual signal confirmations) go through `repo.Playbook`. One API route returns the status, two update it. The frontend loads it once in the help page and passes it to the playbook renderer through `Kit`.

**Tech Stack:** Go (gin, gorm, SQLite default + MySQL 8), React 18 + TypeScript.

**Spec:** `docs/superpowers/specs/2026-10-10-playbook-guided-growth-design.md`. Guidance never goes on the overview beyond one link.

**Commits:** The user asked not to commit. Every task ends with a `git status` checkpoint. Do not commit.

**Deviations from the spec, decided while planning (tell the user):**
1. `search_stage` promotion is **not** changed. It gates statistical alerts on sample size; tying it to the playbook would loosen those safeguards. The playbook judges the SEO stage itself: `seoGrow` when at least 6 of the 8 signals are met (automatic or confirmed by hand), or when the existing search stage is already `established`.
2. `sitemapStable` becomes a manual signal: no per-sitemap index-rate history is stored.
3. The "engines" step is done when a sample run has succeeded: key test results are never stored.

---

## File map

| File | Change | Responsibility |
|---|---|---|
| `internal/model/onboarding.go` | Modify | `ProjectProgress.ProductStage` |
| `internal/model/playbook.go` | Create | `PlaybookConfirmation` |
| `internal/model/migrate.go` | Modify | Register the new model |
| `internal/repo/playbook.go` | Create | Stage and confirmation writes; index facts query |
| `internal/repo/playbook_test.go` | Create | SQLite tests |
| `internal/integration/database_test.go` | Modify | Same writes on SQLite and MySQL 8 |
| `internal/service/webstats/playbook.go` | Create | `PlaybookSearch`: search facts that need package-private helpers |
| `internal/service/webstats/playbook_test.go` | Create | Visible-share weeks |
| `internal/service/playbook/judge.go` | Create | Pure judges: checks, signals, stages |
| `internal/service/playbook/judge_test.go` | Create | Table tests |
| `internal/service/playbook/thresholds_test.go` | Create | Go thresholds equal the TS catalog |
| `internal/service/playbook/service.go` | Create | Gather facts; SetStage; Confirm |
| `internal/service/playbook/service_test.go` | Create | SQLite end-to-end |
| `internal/api/playbook.go`, `playbook_test.go` | Create | Routes |
| `internal/api/handler.go`, `handler_extra.go` | Modify | Field, wiring, routes |
| `web/src/api.ts` | Modify | Types and calls |
| `web/src/features/playbook/catalog.ts` | Modify | `check` on steps, `sitemapStable` manual, `seoPassSignals` |
| `web/src/features/help/kit.tsx` | Modify | `PlaybookCtx` in `Kit` |
| `web/src/features/help/help.tsx` | Modify | Load status, pass it in |
| `web/src/features/help/playbook.tsx` | Modify | Render status, stage choice, confirmations |
| `web/src/features/overview/overview.tsx` | Modify | One link |
| `web/src/i18n/locales/{en,zh,pt}.ts` | Modify | New and changed strings |

---

### Task 1: Model and repo writes

**Files:**
- Modify: `internal/model/onboarding.go` (struct `ProjectProgress`)
- Create: `internal/model/playbook.go`
- Modify: `internal/model/migrate.go` (the `models` slice)
- Create: `internal/repo/playbook.go`, `internal/repo/playbook_test.go`
- Modify: `internal/integration/database_test.go`

- [ ] **Step 1: Add the field and model**

In `ProjectProgress`, after `FirstValueRef`:

```go
	// ProductStage is the user's answer to "which stage is your product in"
	// (s0, s1, s2; empty when not answered). It only changes playbook hints.
	ProductStage string `gorm:"size:8;not null;default:''" json:"product_stage"`
```

Create `internal/model/playbook.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package model

// PlaybookConfirmation records that a user checked a pass signal by hand,
// for signals that no Google API reports. One row per project and signal.
type PlaybookConfirmation struct {
	ProjectID   uint64 `gorm:"primaryKey;autoIncrement:false" json:"project_id"`
	Signal      string `gorm:"primaryKey;size:32" json:"signal"`
	ConfirmedAt int64  `json:"confirmed_at"`
	ConfirmedBy uint64 `json:"confirmed_by"`
}
```

In `migrate.go`, change `&ProjectProgress{},` to `&ProjectProgress{}, &PlaybookConfirmation{},`.

- [ ] **Step 2: Write the failing repo test**

`internal/repo/playbook_test.go` (use this package's existing `testDB(t)` from `webstats_test.go`):

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"testing"

	"github.com/craftsail/craftsail-growth/internal/model"
)

func TestPlaybookStageAndConfirmations(t *testing.T) {
	ctx := context.Background()
	r := &Playbook{DB: testDB(t)}
	if err := r.SetStage(ctx, 7, "s1"); err != nil {
		t.Fatal(err)
	}
	if err := r.SetStage(ctx, 7, "s2"); err != nil {
		t.Fatal(err)
	}
	row, err := (&ProjectProgress{DB: r.DB}).Get(ctx, 7)
	if err != nil || row.ProductStage != "s2" {
		t.Fatalf("stage = %q, %v", row.ProductStage, err)
	}
	for _, on := range []bool{true, true} { // confirming twice is an upsert
		if err := r.Confirm(ctx, 7, "crawlUp", 3, on); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Confirm(ctx, 7, "brandFilter", 3, true); err != nil {
		t.Fatal(err)
	}
	if err := r.Confirm(ctx, 7, "brandFilter", 3, false); err != nil {
		t.Fatal(err)
	}
	got, err := r.Confirmations(ctx, 7)
	if err != nil || len(got) != 1 || !got["crawlUp"] {
		t.Fatalf("confirmations = %v, %v", got, err)
	}
}

func TestPlaybookIndexFacts(t *testing.T) {
	ctx := context.Background()
	r := &Webstats{DB: testDB(t)}
	ts := func(v int64) *int64 { return &v }
	rows := []model.IndexURL{
		{ProjectID: 1, Property: "sc-domain:x.test", KeyHash: "a", URL: "https://x.test/a", Verdict: "PASS", PublishedAt: ts(1000), FirstIndexedAt: ts(2000), LastSuccessAt: ts(5000)},
		{ProjectID: 1, Property: "sc-domain:x.test", KeyHash: "b", URL: "https://x.test/b", Verdict: "NEUTRAL", PublishedAt: ts(50), LastSuccessAt: ts(6000)},
		{ProjectID: 1, Property: "sc-domain:x.test", KeyHash: "c", URL: "https://x.test/c", Verdict: "PASS"},
		{ProjectID: 1, Property: "other", KeyHash: "d", URL: "https://x.test/d", Verdict: "PASS", LastSuccessAt: ts(9000)},
	}
	if err := r.DB.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	f, err := r.PlaybookIndex(ctx, 1, "sc-domain:x.test", 500)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.NewPages) != 1 || f.NewPages[0].KeyHash != "a" || f.Indexed != 2 || f.LastSuccess == nil || *f.LastSuccess != 6000 {
		t.Fatalf("facts = %+v", f)
	}
}
```

Before running, open `internal/model/indexing.go` and confirm the `IndexURL` field names (`URL`, `KeyHash`, `Verdict`, `PublishedAt`, `FirstIndexedAt`, `FirstImpressionAt`, `LastSuccessAt`) and that timestamps are unix seconds `*int64`. If a field differs, adjust the test and Step 4 to the real names.

- [ ] **Step 3: Run it to verify it fails**

Run: `go test ./internal/repo -run 'TestPlaybook' -count=1`
Expected: FAIL, `undefined: Playbook` / `PlaybookIndex`.

- [ ] **Step 4: Implement `internal/repo/playbook.go`**

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"database/sql"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/craftsail/craftsail-growth/internal/model"
)

// Playbook stores the two things a user tells the playbook: the product
// stage and which manual pass signals they have checked.
type Playbook struct{ DB *gorm.DB }

func (r *Playbook) SetStage(ctx context.Context, pid uint64, stage string) error {
	if err := (&ProjectProgress{DB: r.DB}).ensure(ctx, pid); err != nil {
		return err
	}
	return r.DB.WithContext(ctx).Model(&model.ProjectProgress{}).Where("project_id = ?", pid).Update("product_stage", stage).Error
}

func (r *Playbook) Confirm(ctx context.Context, pid uint64, signal string, user uint64, on bool) error {
	if !on {
		return r.DB.WithContext(ctx).Where("project_id = ? AND `signal` = ?", pid, signal).Delete(&model.PlaybookConfirmation{}).Error
	}
	row := model.PlaybookConfirmation{ProjectID: pid, Signal: signal, ConfirmedAt: time.Now().Unix(), ConfirmedBy: user}
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
}

func (r *Playbook) Confirmations(ctx context.Context, pid uint64) (map[string]bool, error) {
	var rows []model.PlaybookConfirmation
	if err := r.DB.WithContext(ctx).Where("project_id = ?", pid).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(rows))
	for _, row := range rows {
		out[row.Signal] = true
	}
	return out, nil
}

// IndexFacts is what the playbook reads from the URL inventory of one
// Search Console property.
type IndexFacts struct {
	NewPages    []model.IndexURL // publish date at or after since
	Indexed     int64            // last verdict PASS
	LastSuccess *int64           // latest successful inspection
}

func (r *Webstats) PlaybookIndex(ctx context.Context, pid uint64, property string, since int64) (*IndexFacts, error) {
	q := func() *gorm.DB {
		return r.DB.WithContext(ctx).Model(&model.IndexURL{}).Where("project_id = ? AND property = ?", pid, property)
	}
	out := &IndexFacts{}
	if err := q().Where("published_at >= ?", since).Find(&out.NewPages).Error; err != nil {
		return nil, err
	}
	if err := q().Where("verdict = ?", "PASS").Count(&out.Indexed).Error; err != nil {
		return nil, err
	}
	var last sql.NullInt64
	if err := q().Select("MAX(last_success_at)").Row().Scan(&last); err != nil {
		return nil, err
	}
	if last.Valid {
		out.LastSuccess = &last.Int64
	}
	return out, nil
}
```

`signal` is quoted because it is a reserved word in MySQL 8. Backticks also work in SQLite.

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test ./internal/repo -run 'TestPlaybook' -count=1`
Expected: PASS.

- [ ] **Step 6: Add the same writes to the MySQL acceptance test**

In `TestDatabaseAcceptance`, after the `index_history` line:

```go
			t.Run("playbook", func(t *testing.T) {
				ctx := context.Background()
				r := &repo.Playbook{DB: db}
				if err := r.SetStage(ctx, 42, "s1"); err != nil {
					t.Fatal(err)
				}
				for _, on := range []bool{true, true, false, true} {
					if err := r.Confirm(ctx, 42, "crawlUp", 1, on); err != nil {
						t.Fatal(err)
					}
				}
				got, err := r.Confirmations(ctx, 42)
				if err != nil || !got["crawlUp"] || len(got) != 1 {
					t.Fatalf("confirmations = %v, %v", got, err)
				}
			})
```

Run: `go test ./internal/integration -run TestDatabaseAcceptance/sqlite -count=1`
Expected: PASS. Then, if Docker is available, `make test-mysql`; expected: PASS. If Docker is not available, say so in the report; do not skip silently.

- [ ] **Step 7: Checkpoint** — `gofmt -l internal && git status --short`

---

### Task 2: Search facts in package webstats

The judges need four search facts that rely on unexported webstats helpers (`indexProperty`, `searchObservation`, `grainCoverage`, `reportCoverage`, `sumOfficial`). Collect them in one exported method.

**Files:**
- Create: `internal/service/webstats/playbook.go`, `internal/service/webstats/playbook_test.go`

- [ ] **Step 1: Write the failing test**

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

func TestVisibleShareWeeks(t *testing.T) {
	ctx := context.Background()
	s := New(testDB(t))
	p, err := project.New(s.rows.DB).Create(ctx, project.CreateInput{Name: "Share", URL: "https://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	prop, _ := GSCPropertyKey(p.Site)
	end := time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC) // Sunday
	start := end.AddDate(0, 0, -34)                       // five full weeks
	week := func(d time.Time) int { return int(d.Sub(start).Hours() / 24 / 7) }
	err = s.syncReport(ctx, p.ID, "gsc", prop, "daily", "web", start, end, 35, 1, func(_ context.Context, part dateChunk) (repo.SyncBatch, error) {
		var rows []model.GscDaily
		for d := part.start; !d.After(part.end); d = d.AddDate(0, 0, 1) {
			rows = append(rows, model.GscDaily{ProjectID: p.ID, Property: prop, SearchType: "web", Day: d, Clicks: 10, Impressions: 100})
		}
		return repo.SyncBatch{GSCDaily: rows}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Visible query clicks per day: 2, 3, 4, 5, 6 in weeks 1..5 → share 20%..60%.
	err = s.syncReport(ctx, p.ID, "gsc", prop, "query", "web", start, end, 35, 1, func(_ context.Context, part dateChunk) (repo.SyncBatch, error) {
		var rows []model.GscFact
		for d := part.start; !d.After(part.end); d = d.AddDate(0, 0, 1) {
			rows = append(rows, model.GscFact{ProjectID: p.ID, Property: prop, Slice: "query", SearchType: "web", Day: d,
				KeyHash: fmt.Sprintf("q-%s", d.Format("0102")), Query: "q", Clicks: float64(2 + week(d))})
		}
		return repo.SyncBatch{GSC: rows}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.visibleShare(ctx, p.ID, prop, end)
	if err != nil {
		t.Fatal(err)
	}
	want := []float64{0.2, 0.3, 0.4, 0.5, 0.6}
	if len(got) != 5 {
		t.Fatalf("weeks = %d", len(got))
	}
	for i, w := range want {
		if got[i] == nil || *got[i] < w-1e-9 || *got[i] > w+1e-9 {
			t.Fatalf("week %d = %v, want %v", i, got[i], w)
		}
	}
	// A week without query coverage is unknown, not zero.
	got, err = s.visibleShare(ctx, p.ID, prop, end.AddDate(0, 0, 7))
	if err != nil || got[4] != nil {
		t.Fatalf("uncovered week = %v, %v", got[4], err)
	}
}
```

Check `syncReport`'s query report name in `internal/service/webstats/official.go` (search for the call that syncs the query slice). If the report is not named `"query"`, use the real name here and in Step 3.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/service/webstats -run TestVisibleShareWeeks -count=1`
Expected: FAIL, `s.visibleShare undefined`.

- [ ] **Step 3: Implement `internal/service/webstats/playbook.go`**

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
)

// PlaybookSearch is what the playbook needs from search data. Unknown
// values stay nil; the playbook shows them as "no data", never as zero.
type PlaybookSearch struct {
	Established          bool             `json:"established"`
	NewPages             []model.IndexURL `json:"-"`
	Indexed              int64            `json:"indexed"`
	LastIndexSuccess     *int64           `json:"last_index_success"`
	PagesWithImpressions *int64           `json:"pages_with_impressions"`
	// VisibleShare is clicks on visible queries / all clicks for the last
	// five complete Monday–Sunday weeks, oldest first.
	VisibleShare  []*float64 `json:"visible_share"`
	GAThresholded *bool      `json:"ga_thresholded"`
}

func (s *Service) PlaybookSearch(ctx context.Context, p *model.Project) (*PlaybookSearch, error) {
	out := &PlaybookSearch{}
	now := s.now()
	idx, err := s.rows.PlaybookIndex(ctx, p.ID, s.indexProperty(ctx, p), now.AddDate(0, 0, -28).Unix())
	if err != nil {
		return nil, err
	}
	out.NewPages, out.Indexed, out.LastIndexSuccess = idx.NewPages, idx.Indexed, idx.LastSuccess
	obs, _, err := s.searchObservation(ctx, p)
	if err != nil {
		return nil, err
	}
	out.Established = obs.Mode == "established"
	out.PagesWithImpressions = obs.PagesWithImpressions
	_, through, property, err := s.searchWindow(ctx, p)
	if err != nil {
		return nil, err
	}
	if out.VisibleShare, err = s.visibleShare(ctx, p.ID, property, through); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.GA4Property) != "" {
		if ga := s.officialProperty(ctx, p, "ga4"); ga != "" {
			cov, err := s.reportCoverage(ctx, p.ID, "ga4", ga, "daily", "", through.AddDate(0, 0, -27), through)
			if err != nil {
				return nil, err
			}
			if cov.State == "covered" {
				t := cov.Quality.Thresholded
				out.GAThresholded = &t
			}
		}
	}
	return out, nil
}

// visibleShare never fills gaps: a week without full daily and query
// coverage, or with no clicks, is nil.
func (s *Service) visibleShare(ctx context.Context, projectID uint64, property string, through time.Time) ([]*float64, error) {
	weekEnd := dateOnly(through).AddDate(0, 0, -int(through.Weekday()))
	from := weekEnd.AddDate(0, 0, -34)
	daily, err := s.rows.ListGscDaily(ctx, projectID, property, from, weekEnd)
	if err != nil {
		return nil, err
	}
	queries, err := s.rows.ListGscSlice(ctx, projectID, property, "query", from, weekEnd)
	if err != nil {
		return nil, err
	}
	out := make([]*float64, 0, 5)
	for i := 4; i >= 0; i-- {
		end := weekEnd.AddDate(0, 0, -7*i)
		start := end.AddDate(0, 0, -6)
		dc, err := s.grainCoverage(ctx, projectID, property, "daily", start, end)
		if err != nil {
			return nil, err
		}
		qc, err := s.grainCoverage(ctx, projectID, property, "query", start, end)
		if err != nil {
			return nil, err
		}
		total := sumOfficial(daily, start, end).clicks
		if dc.State != "covered" || qc.State != "covered" || total <= 0 {
			out = append(out, nil)
			continue
		}
		var visible float64
		for _, q := range queries {
			day := dateOnly(q.Day)
			if (q.SearchType == "" || q.SearchType == "web") && !day.Before(start) && !day.After(end) {
				visible += q.Clicks
			}
		}
		share := visible / total
		out = append(out, &share)
	}
	return out, nil
}
```

Check before compiling: `searchWindow` returns `(from, through, property, err)`; `dateOnly` takes and returns `time.Time`; `ListGscSlice`'s `to` is inclusive. Adjust if the real signatures differ.

- [ ] **Step 4: Run the test**

Run: `go test ./internal/service/webstats -count=1`
Expected: PASS (whole package, to catch interference).

- [ ] **Step 5: Checkpoint** — `gofmt -l internal && git status --short`

---

### Task 3: Judges

**Files:**
- Create: `internal/service/playbook/judge.go`, `judge_test.go`, `thresholds_test.go`

- [ ] **Step 1: Write the failing tests**

`judge_test.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package playbook

import (
	"testing"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

var now = time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC) // Wednesday

func i64(v int64) *int64     { return &v }
func f64(v float64) *float64 { return &v }
func day(d int) int64        { return now.Unix() - int64(d)*86400 }

func page(published, indexedAfter int, impressions bool) model.IndexURL {
	p := model.IndexURL{PublishedAt: i64(day(published))}
	if indexedAfter >= 0 {
		p.FirstIndexedAt = i64(day(published) + int64(indexedAfter)*86400)
	}
	if impressions {
		p.FirstImpressionAt = i64(day(published) + 86400)
	}
	return p
}

func TestNewFast(t *testing.T) {
	cases := []struct {
		name  string
		pages []model.IndexURL
		want  string
	}{
		{"fast with impressions", []model.IndexURL{page(20, 1, true), page(15, 2, true), page(10, 3, false)}, "met"},
		{"slow", []model.IndexURL{page(20, 9, true), page(15, 8, true), page(10, -1, true)}, "unmet"},
		{"mostly no impressions", []model.IndexURL{page(20, 1, false), page(15, 1, false), page(10, 1, true)}, "unmet"},
		{"too few", []model.IndexURL{page(20, 1, true), page(15, 1, true)}, "no_data"},
		{"young pages are not judged", []model.IndexURL{page(20, 1, true), page(15, 1, true), page(1, -1, false)}, "no_data"},
	}
	for _, c := range cases {
		got := newFast(Facts{Now: now, Search: &webstats.PlaybookSearch{NewPages: c.pages}})
		if got.State != c.want {
			t.Errorf("%s: %s, want %s", c.name, got.State, c.want)
		}
	}
	if newFast(Facts{Now: now}).State != "no_data" {
		t.Error("no search facts must be no_data")
	}
}

func TestSearchSignals(t *testing.T) {
	share := func(v ...float64) []*float64 {
		out := []*float64{}
		for _, x := range v {
			if x < 0 {
				out = append(out, nil)
			} else {
				out = append(out, f64(x))
			}
		}
		return out
	}
	cases := []struct {
		name string
		got  Signal
		want string
	}{
		{"coverage met", imprCoverageSignal(&webstats.PlaybookSearch{Indexed: 10, PagesWithImpressions: i64(6)}), "met"},
		{"coverage unmet", imprCoverageSignal(&webstats.PlaybookSearch{Indexed: 10, PagesWithImpressions: i64(4)}), "unmet"},
		{"coverage nothing indexed", imprCoverageSignal(&webstats.PlaybookSearch{PagesWithImpressions: i64(4)}), "no_data"},
		{"coverage unknown", imprCoverageSignal(&webstats.PlaybookSearch{Indexed: 10}), "no_data"},
		{"share rising", visibleShareSignal(&webstats.PlaybookSearch{VisibleShare: share(0.1, 0.15, 0.2, 0.25, 0.3)}), "met"},
		{"share high", visibleShareSignal(&webstats.PlaybookSearch{VisibleShare: share(-1, -1, -1, -1, 0.55)}), "met"},
		{"share flat", visibleShareSignal(&webstats.PlaybookSearch{VisibleShare: share(0.1, 0.15, 0.15, 0.25, 0.3)}), "unmet"},
		{"share gap", visibleShareSignal(&webstats.PlaybookSearch{VisibleShare: share(0.1, -1, 0.2, 0.25, 0.3)}), "no_data"},
		{"share latest unknown", visibleShareSignal(&webstats.PlaybookSearch{VisibleShare: share(0.6, -1)}), "no_data"},
		{"ga clean", gaThresholdSignal(&webstats.PlaybookSearch{GAThresholded: new(bool)}), "met"},
		{"ga unknown", gaThresholdSignal(&webstats.PlaybookSearch{}), "no_data"},
		{"recognized", recognizedSignal(27, 30), "met"},
		{"not recognized", recognizedSignal(15, 30), "unmet"},
		{"recognition small sample", recognizedSignal(20, 20), "no_data"},
	}
	for _, c := range cases {
		if c.got.State != c.want {
			t.Errorf("%s: %s, want %s", c.name, c.got.State, c.want)
		}
	}
}

func TestStagesAndChecks(t *testing.T) {
	base := Facts{Now: now, Layers: map[string]string{}, Rules: map[string]bool{}, Confirmed: map[string]bool{}, Search: &webstats.PlaybookSearch{}}

	s := Judge(base)
	if s.SeoStage != "seoNew" || s.AiStage != "aiReach" || s.Checks["first_check"].State != "todo" {
		t.Fatalf("empty project: %+v", s)
	}
	if s.Signals["crawlUp"].State != "manual" {
		t.Fatalf("manual signal: %+v", s.Signals["crawlUp"])
	}

	noSite := base
	noSite.NoSite, noSite.Search = true, nil
	if s := Judge(noSite); s.SeoStage != "" || s.AiStage != "aiKnown" {
		t.Fatalf("no site: seo %q ai %q", s.SeoStage, s.AiStage)
	}

	est := base
	est.Search = &webstats.PlaybookSearch{Established: true}
	if Judge(est).SeoStage != "seoGrow" {
		t.Fatal("established search stage must count as passed")
	}

	six := base
	six.Confirmed = map[string]bool{"sitemapStable": true, "discoveredDown": true, "crawlUp": true, "brandFilter": true}
	six.Search = &webstats.PlaybookSearch{Indexed: 10, PagesWithImpressions: i64(8), GAThresholded: new(bool)}
	if s := Judge(six); s.SeoMet != 6 || s.SeoStage != "seoGrow" {
		t.Fatalf("six signals: met %d stage %s", s.SeoMet, s.SeoStage)
	}

	ai := base
	ai.AuditDone = true
	ai.Layers = map[string]string{"access": "ok", "discover": "ok", "understand": "warn"}
	ai.RecognitionX, ai.RecognitionN = 28, 30
	s = Judge(ai)
	if s.AiStage != "aiRecommend" || s.Checks["audit_layers_ok"].State != "todo" || s.Checks["audit_access_ok"].State != "done" {
		t.Fatalf("ai: %s %+v", s.AiStage, s.Checks)
	}
	ai.Layers["discover"] = "blocked"
	if Judge(ai).AiStage != "aiReach" {
		t.Fatal("a blocked layer does not pass")
	}

	week := base
	monday := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	week.ReportOn = &monday
	week.Search = &webstats.PlaybookSearch{LastIndexSuccess: i64(day(8))}
	s = Judge(week)
	if s.Checks["report_this_week"].State != "done" || s.Checks["index_checked_7d"].State != "todo" {
		t.Fatalf("week checks: %+v", s.Checks)
	}
	sunday := monday.AddDate(0, 0, -1)
	week.ReportOn = &sunday
	if Judge(week).Checks["report_this_week"].State != "todo" {
		t.Fatal("last week's report does not count")
	}
	if Judge(noSite).Checks["index_checked_7d"].State != "no_data" {
		t.Fatal("no site has no index check")
	}
}

func TestObservationDue(t *testing.T) {
	o := model.Observation{FollowupThrough: "2026-10-05", Metric: "technical"}
	if !observationDue(o, now) {
		t.Fatal("technical window ended two days ago")
	}
	o.Metric = "clicks"
	if observationDue(o, now) {
		t.Fatal("search metrics wait for data to settle")
	}
	o.Results = []model.ObservationResult{{}}
	if observationDue(o, now.AddDate(0, 0, 10)) {
		t.Fatal("evaluated observation is not due")
	}
}
```

`thresholds_test.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package playbook

import (
	"os"
	"regexp"
	"strconv"
	"testing"
)

// The help center prints the thresholds from the TS catalog; the judges
// use the Go constants. They must be the same numbers.
func TestThresholdsMatchCatalog(t *testing.T) {
	src, err := os.ReadFile("../../../web/src/features/playbook/catalog.ts")
	if err != nil {
		t.Fatal(err)
	}
	block := regexp.MustCompile(`(?s)export const THRESHOLDS = \{(.*?)\} as const`).FindSubmatch(src)
	if block == nil {
		t.Fatal("THRESHOLDS not found in catalog.ts")
	}
	ts := map[string]int{}
	for _, m := range regexp.MustCompile(`(\w+):\s*(\d+)`).FindAllSubmatch(block[1], -1) {
		v, _ := strconv.Atoi(string(m[2]))
		ts[string(m[1])] = v
	}
	want := map[string]int{
		"newPageDays": newPageDays, "imprCoverage": imprCoverage, "visibleShare": visibleShare, "weeks": weeks,
		"recognitionN": recognitionN, "recognitionLower": recognitionLower, "seoPassSignals": seoPassSignals,
	}
	for k, v := range want {
		if ts[k] != v {
			t.Errorf("%s: catalog %d, Go %d", k, ts[k], v)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/service/playbook -count=1`
Expected: FAIL, undefined `Judge`, `Facts`, ….

- [ ] **Step 3: Implement `judge.go`**

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package playbook

import (
	"math"
	"sort"
	"time"

	"github.com/craftsail/craftsail-growth/internal/service/metrics"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

// Thresholds mirror THRESHOLDS in web/src/features/playbook/catalog.ts,
// which the help center prints. TestThresholdsMatchCatalog keeps them equal.
const (
	newPageDays      = 3
	imprCoverage     = 50
	visibleShare     = 50
	weeks            = 4
	recognitionN     = 30
	recognitionLower = 50
	seoPassSignals   = 6
)

// Facts is everything the judges read, gathered once per request.
type Facts struct {
	Now                  time.Time
	NoSite               bool
	ProductStage         string
	FirstCheck           bool
	GoogleSelected       bool
	EngineAnswered       bool
	AuditDone            bool
	Layers               map[string]string // access, discover, understand, cite → ok | warn | fail | blocked
	Rules                map[string]bool   // rule codes that fired in the latest audit
	BrandFilled          bool
	QuestionsConfirmed   bool
	Competitors          int
	RecognitionX         int // branded answers naming the brand, last 30 days
	RecognitionN         int
	RecognitionN7        int // branded answers in the last 7 days
	ReportOn             *time.Time
	OpenTasks            int
	DueObservations      int
	Search               *webstats.PlaybookSearch // nil without a site
	Confirmed            map[string]bool
}

type Check struct {
	State string `json:"state"` // done | todo | no_data
	At    *int64 `json:"at,omitempty"`
}

type Signal struct {
	State     string   `json:"state"` // met | unmet | no_data | manual
	Value     *float64 `json:"value,omitempty"`
	Confirmed bool     `json:"confirmed"`
}

type Status struct {
	ProductStage string            `json:"product_stage"`
	SeoStage     string            `json:"seo_stage"` // seoNew | seoGrow | "" without a site
	AiStage      string            `json:"ai_stage"`  // aiReach | aiKnown | aiRecommend
	SeoMet       int               `json:"seo_met"`
	SeoNeeded    int               `json:"seo_needed"`
	Checks       map[string]Check  `json:"checks"`
	Signals      map[string]Signal `json:"signals"`
}

// ManualSignals have no API behind them; the user confirms them.
var ManualSignals = map[string]bool{"sitemapStable": true, "discoveredDown": true, "crawlUp": true, "brandFilter": true}

var seoSignals = []string{"newFast", "sitemapStable", "discoveredDown", "crawlUp", "imprCoverage", "visibleShare", "brandFilter", "gaThreshold"}

// Judge turns facts into the playbook status. The SEO stage passes on the
// book's signals, or when search data already qualified the property as
// established; the AI stages pass in order.
func Judge(f Facts) Status {
	sig := judgeSignals(f)
	out := Status{ProductStage: f.ProductStage, SeoNeeded: seoPassSignals, Checks: judgeChecks(f), Signals: sig}
	for _, id := range seoSignals {
		if sig[id].State == "met" {
			out.SeoMet++
		}
	}
	if !f.NoSite {
		out.SeoStage = "seoNew"
		if out.SeoMet >= seoPassSignals || (f.Search != nil && f.Search.Established) {
			out.SeoStage = "seoGrow"
		}
	}
	out.AiStage = "aiReach"
	if f.NoSite || (sig["accessPass"].State == "met" && sig["discoverPass"].State == "met") {
		out.AiStage = "aiKnown"
		if sig["recognized"].State == "met" {
			out.AiStage = "aiRecommend"
		}
	}
	return out
}

func done(ok bool) Check {
	if ok {
		return Check{State: "done"}
	}
	return Check{State: "todo"}
}

func judgeChecks(f Facts) map[string]Check {
	layer := func(key string) bool { return f.AuditDone && f.Layers[key] == "ok" }
	c := map[string]Check{
		"first_check":          done(f.FirstCheck),
		"google_connected":     done(f.GoogleSelected),
		"engines_answered":     done(f.EngineAnswered),
		"audit_layers_ok":      done(layer("access") && layer("discover") && layer("understand")),
		"sitemap_ok":           done(f.AuditDone && !f.Rules["NO_SITEMAP"] && !f.Rules["SITEMAP_NOT_DECLARED"]),
		"audit_access_ok":      done(layer("access")),
		"llms_published":       done(f.AuditDone && !f.Rules["NO_LLMS_TXT"]),
		"brand_facts":          done(f.BrandFilled),
		"recognition_n":        done(f.RecognitionN7 >= recognitionN),
		"questions_confirmed":  done(f.QuestionsConfirmed),
		"competitors_3":        done(f.Competitors >= 3),
		"experiment_open":      done(f.OpenTasks > 0),
		"experiments_reviewed": done(f.DueObservations == 0),
		"report_this_week":     done(f.ReportOn != nil && !f.ReportOn.Before(weekStart(f.Now))),
		"index_checked_7d":     {State: "no_data"},
	}
	if f.Search != nil {
		last := f.Search.LastIndexSuccess
		idx := done(last != nil && f.Now.Unix()-*last <= 7*86400)
		idx.At = last
		c["index_checked_7d"] = idx
	}
	return c
}

// weekStart is Monday 00:00 UTC of t's week.
func weekStart(t time.Time) time.Time {
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7))
}

var noData = Signal{State: "no_data"}

func met(ok bool, v *float64) Signal {
	if ok {
		return Signal{State: "met", Value: v}
	}
	return Signal{State: "unmet", Value: v}
}

func val(v float64) *float64 { return &v }

func judgeSignals(f Facts) map[string]Signal {
	s := map[string]Signal{
		"newFast":      newFast(f),
		"imprCoverage": imprCoverageSignal(f.Search),
		"visibleShare": visibleShareSignal(f.Search),
		"gaThreshold":  gaThresholdSignal(f.Search),
		"accessPass":   layerSignal(f, "access"),
		"discoverPass": layerSignal(f, "discover"),
		"recognized":   recognizedSignal(f.RecognitionX, f.RecognitionN),
	}
	for id := range ManualSignals {
		s[id] = Signal{State: "manual"}
		if f.Confirmed[id] {
			s[id] = Signal{State: "met", Confirmed: true}
		}
	}
	return s
}

// newFast: median days from publish to first observed indexing over new
// pages old enough to judge (unindexed pages older than the threshold count
// as too slow), and at least half of them already have impressions. Value
// is the median in days.
func newFast(f Facts) Signal {
	if f.Search == nil {
		return noData
	}
	var delays []float64
	seen := 0
	for _, p := range f.Search.NewPages {
		if p.PublishedAt == nil {
			continue
		}
		switch {
		case p.FirstIndexedAt != nil:
			delays = append(delays, math.Max(0, float64(*p.FirstIndexedAt-*p.PublishedAt)/86400))
		case float64(f.Now.Unix()-*p.PublishedAt)/86400 > newPageDays:
			delays = append(delays, math.Inf(1))
		default:
			continue
		}
		if p.FirstImpressionAt != nil {
			seen++
		}
	}
	if len(delays) < 3 {
		return noData
	}
	sort.Float64s(delays)
	median := delays[len(delays)/2]
	var v *float64
	if !math.IsInf(median, 1) {
		v = val(median)
	}
	return met(median <= newPageDays && seen*2 >= len(delays), v)
}

func imprCoverageSignal(s *webstats.PlaybookSearch) Signal {
	if s == nil || s.Indexed == 0 || s.PagesWithImpressions == nil {
		return noData
	}
	pct := math.Min(100, 100*float64(*s.PagesWithImpressions)/float64(s.Indexed))
	return met(pct >= imprCoverage, val(pct))
}

// visibleShareSignal: the latest week's share reaches the threshold, or the
// share rose in each of the last `weeks` weeks. Value is the latest share in %.
func visibleShareSignal(s *webstats.PlaybookSearch) Signal {
	if s == nil || len(s.VisibleShare) == 0 || s.VisibleShare[len(s.VisibleShare)-1] == nil {
		return noData
	}
	pct := 100 * *s.VisibleShare[len(s.VisibleShare)-1]
	if pct >= visibleShare {
		return met(true, val(pct))
	}
	if len(s.VisibleShare) < weeks+1 {
		return noData
	}
	tail := s.VisibleShare[len(s.VisibleShare)-weeks-1:]
	for i := 1; i < len(tail); i++ {
		if tail[i-1] == nil || tail[i] == nil {
			return noData
		}
		if *tail[i] <= *tail[i-1] {
			return met(false, val(pct))
		}
	}
	return met(true, val(pct))
}

func gaThresholdSignal(s *webstats.PlaybookSearch) Signal {
	if s == nil || s.GAThresholded == nil {
		return noData
	}
	return met(!*s.GAThresholded, nil)
}

func layerSignal(f Facts, key string) Signal {
	if !f.AuditDone {
		return noData
	}
	return met(f.Layers[key] == "ok", nil)
}

// recognizedSignal uses the lower end of the 95% Wilson interval, so a
// small lucky sample does not pass. Value is that lower end in %.
func recognizedSignal(x, n int) Signal {
	if n < recognitionN {
		return noData
	}
	ci := metrics.Wilson(x, n)
	if ci == nil {
		return noData
	}
	lo := 100 * ci.Lo
	return met(lo >= recognitionLower, val(lo))
}

// observationDue: the window has ended and no result was recorded. Mirror
// the readiness rule in plan.Evaluate (internal/service/plan/observation.go);
// if that rule differs from the one below, use the real one and update the test.
func observationDue(o model.Observation, now time.Time) bool {
	if len(o.Results) > 0 || o.FollowupThrough == "" {
		return false
	}
	end, err := time.Parse("2006-01-02", o.FollowupThrough)
	if err != nil {
		return false
	}
	wait := 24 * time.Hour
	if o.Metric == "clicks" || o.Metric == "impressions" {
		wait = 80 * time.Hour
	}
	return !now.Before(end.Add(wait))
}
```

Add `"github.com/craftsail/craftsail-growth/internal/model"` to the imports (used by `observationDue`). Note `IndexURL.PublishedAt` is a pointer; the test helper sets it.

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/service/playbook -count=1`
Expected: `TestThresholdsMatchCatalog` FAILS on `seoPassSignals` (catalog has no such key until Task 6). Everything else PASSES. Leave that one failing; Task 6 fixes it. If any other test fails, fix the judge, not the test, unless the test contradicts the rule text in this plan.

- [ ] **Step 5: Checkpoint** — `gofmt -l internal && git status --short`

---

### Task 4: Service: gather facts, write stage and confirmations

**Files:**
- Create: `internal/service/playbook/service.go`, `internal/service/playbook/service_test.go`

- [ ] **Step 1: Write the failing test**

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package playbook

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := model.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestStatusFromStoredData(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	s := New(db)
	s.Now = func() time.Time { return time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC) }
	p, err := project.New(db).Create(ctx, project.CreateInput{Name: "Quill", URL: "https://quill.test/"})
	if err != nil {
		t.Fatal(err)
	}

	st, err := s.Status(ctx, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if st.SeoStage != "seoNew" || st.AiStage != "aiReach" || st.Checks["first_check"].State != "todo" {
		t.Fatalf("new project: %+v", st)
	}

	audit := model.Audit{ProjectID: p.ID, PageCount: 3, Layers: []map[string]any{
		{"key": "access", "status": "ok"}, {"key": "discover", "status": "ok"}, {"key": "understand", "status": "warn"},
	}}
	if err := db.Create(&audit).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.AuditIssue{ProjectID: p.ID, AuditID: audit.ID, Code: "NO_LLMS_TXT"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.SetStage(ctx, p.Slug, "s0"); err != nil {
		t.Fatal(err)
	}
	if err := s.Confirm(ctx, p.Slug, "crawlUp", 1, true); err != nil {
		t.Fatal(err)
	}
	st, err = s.Status(ctx, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if st.ProductStage != "s0" || st.AiStage != "aiKnown" || st.Checks["first_check"].State != "done" ||
		st.Checks["llms_published"].State != "todo" || st.Checks["audit_access_ok"].State != "done" ||
		st.Signals["crawlUp"].State != "met" || st.SeoMet != 1 {
		t.Fatalf("after audit: %+v", st)
	}

	if err := s.SetStage(ctx, p.Slug, "s9"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad stage: %v", err)
	}
	if err := s.Confirm(ctx, p.Slug, "newFast", 1, true); !errors.Is(err, ErrInvalid) {
		t.Fatalf("automatic signal cannot be confirmed: %v", err)
	}
}
```

Check `model.Audit` and `model.AuditIssue` field names (`PageCount`, `NoSite`, `Layers`, `AuditID`, `Code`) in `internal/model/audit.go` and adjust the seed if they differ. If `repo.(*Audits).LatestIssues` reads issues by the latest audit's id, the seed above matches; if it reads them another way, seed accordingly.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/service/playbook -run TestStatusFromStoredData -count=1`
Expected: FAIL, `undefined: New`.

- [ ] **Step 3: Implement `service.go`**

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package playbook

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/plan"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"github.com/craftsail/craftsail-growth/internal/service/report"
	"github.com/craftsail/craftsail-growth/internal/service/sample"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

var ErrInvalid = errors.New("unknown product stage or signal")

// Service reports where a project stands in the playbooks. It only reads
// data other features store, apart from the product stage and the manual
// signal confirmations.
type Service struct {
	db       *gorm.DB
	projects *project.Service
	audits   *repo.Audits
	sample   *sample.Service
	plan     *plan.Service
	reports  *report.Service
	web      *webstats.Service
	rows     *repo.Playbook
	// Now overrides time.Now in tests.
	Now func() time.Time
}

func New(db *gorm.DB) *Service {
	return &Service{
		db: db, projects: project.New(db), audits: &repo.Audits{DB: db}, sample: sample.New(db, nil),
		plan: plan.New(db), reports: report.New(db), web: webstats.New(db), rows: &repo.Playbook{DB: db},
	}
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) Status(ctx context.Context, slug string) (*Status, error) {
	f, err := s.facts(ctx, slug)
	if err != nil {
		return nil, err
	}
	out := Judge(*f)
	return &out, nil
}

func (s *Service) SetStage(ctx context.Context, slug, stage string) error {
	switch stage {
	case "", "s0", "s1", "s2":
	default:
		return ErrInvalid
	}
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return err
	}
	return s.rows.SetStage(ctx, p.ID, stage)
}

func (s *Service) Confirm(ctx context.Context, slug, signal string, user uint64, on bool) error {
	if !ManualSignals[signal] {
		return ErrInvalid
	}
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return err
	}
	return s.rows.Confirm(ctx, p.ID, signal, user, on)
}

func (s *Service) facts(ctx context.Context, slug string) (*Facts, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	prog, err := s.projects.Progress(ctx, slug)
	if err != nil {
		return nil, err
	}
	f := &Facts{
		Now: s.now().UTC(), NoSite: p.NoSite, ProductStage: prog.ProductStage, QuestionsConfirmed: prog.QuestionsConfirmed,
		GoogleSelected: strings.TrimSpace(p.GscSite) != "", BrandFilled: brandFilled(p.Brand),
		Layers: map[string]string{}, Rules: map[string]bool{},
	}
	a, _, err := s.audits.Latest(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	if a != nil && !a.NoSite && a.PageCount > 0 {
		f.FirstCheck, f.AuditDone = true, true
		for _, l := range a.Layers {
			key, _ := l["key"].(string)
			status, _ := l["status"].(string)
			if b, _ := l["blocked_by"].(string); b != "" {
				status = "blocked"
			}
			f.Layers[key] = status
		}
		issues, err := s.audits.LatestIssues(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		for _, i := range issues {
			f.Rules[i.Code] = true
		}
	}
	runs, err := s.sample.Runs(ctx, slug, 50)
	if err != nil {
		return nil, err
	}
	for _, r := range runs {
		if r.Status == "completed" && r.Succeeded > 0 {
			f.EngineAnswered = true
			break
		}
	}
	comps, err := (&repo.Competitors{DB: s.db}).List(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	for _, c := range comps {
		if model.CompetitorConfirmed(c) {
			f.Competitors++
		}
	}
	month, err := s.sample.Measure(ctx, slug, sample.MeasureQuery{Range: "30d"})
	if err != nil {
		return nil, err
	}
	if month.Recognition != nil {
		f.RecognitionN = month.RecognitionN
		f.RecognitionX = int(math.Round(*month.Recognition / 100 * float64(month.RecognitionN)))
	}
	week, err := s.sample.Measure(ctx, slug, sample.MeasureQuery{Range: "7d"})
	if err != nil {
		return nil, err
	}
	f.RecognitionN7 = week.RecognitionN
	if r, err := s.reports.Latest(ctx, slug); err != nil {
		return nil, err
	} else if r != nil {
		on := r.ReportOn
		f.ReportOn = &on
	}
	tasks, err := (&repo.Tasks{DB: s.db}).List(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	for _, t := range tasks {
		if t.Status == "open" || t.Status == "doing" {
			f.OpenTasks++
		}
	}
	obs, err := s.plan.Observations(ctx, slug, "")
	if err != nil {
		return nil, err
	}
	for _, o := range obs {
		if observationDue(o, f.Now) {
			f.DueObservations++
		}
	}
	if !p.NoSite {
		if f.Search, err = s.web.PlaybookSearch(ctx, p); err != nil {
			return nil, err
		}
	}
	if f.Confirmed, err = s.rows.Confirmations(ctx, p.ID); err != nil {
		return nil, err
	}
	return f, nil
}

func brandFilled(b model.Brand) bool {
	if model.IsPendingFact(b.Definition) || model.IsPendingFact(b.TargetUsers) {
		return false
	}
	for _, a := range b.Aliases {
		if !model.IsPendingFact(a) {
			return true
		}
	}
	return false
}
```

Points to verify while compiling, and fix to the real API if they differ (report each one):
- `sample.New(db, nil)` accepts a nil asker; `Runs` and `Measure` never call it. If `nil` does not type-check, pass `sample.NewAsker()`.
- `Measure` on a project with no samples returns an empty view, not an error. If it returns a sentinel "no data" error, treat that error as zero answers.
- `report.Latest` returns `(nil, nil)` without a report; `ReportOn` is a `time.Time`.
- `repo.Audits.Latest` returns `(*model.Audit, []model.AuditPage, error)`.

- [ ] **Step 4: Run all playbook tests**

Run: `go test ./internal/service/playbook -count=1`
Expected: all PASS except `TestThresholdsMatchCatalog` (fixed in Task 6).

- [ ] **Step 5: Checkpoint** — `gofmt -l internal && go vet ./internal/... && git status --short`

---

### Task 5: API routes

**Files:**
- Create: `internal/api/playbook.go`, `internal/api/playbook_test.go`
- Modify: `internal/api/handler.go` (Handler struct, `Mount`), `internal/api/handler_extra.go` (`Wire`)

- [ ] **Step 1: Write the failing test**

Read `internal/api/testutil_test.go` and an existing test such as `onboarding_test.go` first; reuse `testDB`, `testHandler`, `testEngine`, `call` and the way they create a project and decode `resp.OK` JSON. Then:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/craftsail/craftsail-growth/internal/service/playbook"
)

func TestPlaybookRoutes(t *testing.T) {
	db := testDB(t)
	h := testHandler(t, db)
	h.playbook = playbook.New(db)
	r := testEngine(h)
	slug := createTestProject(t, r) // use the helper existing tests use; name it accordingly

	w := call(r, "GET", "/api/projects/"+slug+"/playbook", "", testToken)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"seo_stage":"seoNew"`) {
		t.Fatalf("get: %d %s", w.Code, w.Body.String())
	}
	w = call(r, "PUT", "/api/projects/"+slug+"/playbook/stage", `{"product_stage":"s1"}`, testToken)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"product_stage":"s1"`) {
		t.Fatalf("stage: %d %s", w.Code, w.Body.String())
	}
	w = call(r, "PUT", "/api/projects/"+slug+"/playbook/stage", `{"product_stage":"s7"}`, testToken)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad stage: %d", w.Code)
	}
	w = call(r, "PUT", "/api/projects/"+slug+"/playbook/signals/brandFilter", `{"confirmed":true}`, testToken)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"brandFilter":{"state":"met"`) {
		t.Fatalf("confirm: %d %s", w.Code, w.Body.String())
	}
	w = call(r, "PUT", "/api/projects/"+slug+"/playbook/signals/newFast", `{"confirmed":true}`, testToken)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("automatic signal: %d", w.Code)
	}
}
```

Also add one permission case in the style of `permissions_test.go`: a view-only user gets 200 on GET and 403 on both PUTs.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/api -run 'TestPlaybook' -count=1`
Expected: FAIL, `h.playbook undefined`.

- [ ] **Step 3: Implement**

`internal/api/playbook.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/craftsail/craftsail-growth/internal/pkg/resp"
	"github.com/craftsail/craftsail-growth/internal/service/playbook"
)

func (h *Handler) getPlaybook(c *gin.Context) {
	if h.playbook == nil {
		writeErr(c, errors.New("playbook not configured"))
		return
	}
	out, err := h.playbook.Status(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, resp.OK(out))
}

func (h *Handler) putPlaybookStage(c *gin.Context) {
	var body struct {
		ProductStage string `json:"product_stage"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "invalid product stage")
		return
	}
	h.playbookWrite(c, h.playbook.SetStage(c.Request.Context(), c.Param("slug"), body.ProductStage))
}

func (h *Handler) putPlaybookSignal(c *gin.Context) {
	var body struct {
		Confirmed bool `json:"confirmed"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, "invalid confirmation")
		return
	}
	var user uint64
	if a := who(c); a != nil && a.User != nil {
		user = a.User.ID
	}
	h.playbookWrite(c, h.playbook.Confirm(c.Request.Context(), c.Param("slug"), c.Param("signal"), user, body.Confirmed))
}

// playbookWrite answers a write with the new status, so the page updates
// from one response.
func (h *Handler) playbookWrite(c *gin.Context, err error) {
	if errors.Is(err, playbook.ErrInvalid) {
		fail(c, resp.CodeBadRequest, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeErr(c, err)
		return
	}
	h.getPlaybook(c)
}
```

In `handler.go`: add `playbook *playbook.Service` to `Handler` (and the import). In `Mount`, next to the `progress` routes:

```go
	r("GET", "/projects/:slug/playbook", permView, h.getPlaybook)
	r("PUT", "/projects/:slug/playbook/stage", permEdit, h.putPlaybookStage)
	r("PUT", "/projects/:slug/playbook/signals/:signal", permEdit, h.putPlaybookSignal)
```

Place the two PUT lines where the other `permEdit` project routes are registered if `Mount` groups them by permission. In `Wire` (`handler_extra.go`), after `h.web = webstats.New(db)`: `h.playbook = playbook.New(db)`.

- [ ] **Step 4: Run**

Run: `go test ./internal/api -count=1 && go vet ./... && test -z "$(gofmt -l .)"`
Expected: PASS, no vet or fmt output.

- [ ] **Step 5: Checkpoint** — `git status --short`

---

### Task 6: Catalog and text

**Files:**
- Modify: `web/src/features/playbook/catalog.ts`
- Modify: `web/src/i18n/locales/{en,zh,pt}.ts` (`playbook` block, `overview` block)

- [ ] **Step 1: Catalog changes**

1. In `THRESHOLDS`, add `seoPassSignals: 6,` after `indexDrop: 5,`.
2. Export the check ids and add an optional `check` to `Step`:

```ts
// Done checks the server can judge for a step (internal/service/playbook).
export const CHECKS = [
  "first_check", "google_connected", "engines_answered", "audit_layers_ok", "sitemap_ok", "index_checked_7d",
  "audit_access_ok", "llms_published", "brand_facts", "recognition_n", "questions_confirmed", "competitors_3",
  "experiment_open", "experiments_reviewed", "report_this_week",
] as const;
export type CheckId = (typeof CHECKS)[number];

type Step = { id: string; topic: PlaybookTopic; cadence: Cadence; page: { path: string; label: Key }; books: readonly BookRef[]; check?: CheckId };
```

3. Add `check` to these steps (append `, check: "<id>"` inside each object):

| step | check |
|---|---|
| project | first_check |
| google | google_connected |
| engines | engines_answered |
| seoLayers | audit_layers_ok |
| seoSitemap | sitemap_ok |
| seoInspect | index_checked_7d |
| aiBots | audit_access_ok |
| aiLlms | llms_published |
| knownFacts | brand_facts |
| knownRecognition | recognition_n |
| recQuestions | questions_confirmed |
| recCompetitors | competitors_3 |
| weekPick | experiment_open |
| weekReview | experiments_reviewed |
| weekReport | report_this_week |

4. Change `{ id: "sitemapStable", topic: "seoNew", auto: true }` to `auto: false`, and update the `S0`/signals comment if it lists automatic signals.

- [ ] **Step 2: Changed strings (all three locales)**

| key | en | zh | pt |
|---|---|---|---|
| `playbook.auto` | `Automatic` | `自动` | `Automática` |
| `playbook.passNote` | `These thresholds are our starting assumptions, not Google or engine rules. We will tune them with real projects and write the result back into the book.` | `这些阈值是我们的先验判断，不是 Google 或 AI 的规则。之后会用真实项目校准，再写回书里。` | `Estes limites são nossas hipóteses iniciais, não regras do Google ou dos mecanismos. Vamos ajustá-los com projetos reais e registrar o resultado no livro.` |
| `playbook.steps.engines.done` | `At least one engine has answered in a sample run.` | `至少一个引擎在采样中成功回答过。` | `Pelo menos um mecanismo respondeu numa amostragem.` |
| `playbook.steps.aiLlms.done` | `llms.txt is published.` | `llms.txt 已发布。` | `O llms.txt está publicado.` |
| `playbook.signals.sitemapStable.rule` | `In Search Console's Sitemaps report, each sitemap's indexed share is at least {sitemapRate}% for {weeks} weeks in a row, moving no more than {sitemapSwing} points between weeks` | `在 Search Console 的站点地图报告里看：各 sitemap 的已编入索引占比连续 {weeks} 周不低于 {sitemapRate}%，周间波动不超过 {sitemapSwing} 个点` | `No relatório Sitemaps do Search Console, a parcela indexada de cada sitemap fica em pelo menos {sitemapRate}% por {weeks} semanas seguidas, variando no máximo {sitemapSwing} pontos entre semanas` |
| `playbook.signals.newFast.rule` | `Median indexing time of new pages in the last 28 days is at most {newPageDays} days, and most of them have impressions. Needs publish dates on the Indexing tab.` | `近 28 天新页收录时长中位数不超过 {newPageDays} 天，且大多数已有展示。需要在收录页填写发布日期。` | `A mediana do tempo de indexação das páginas novas nos últimos 28 dias é de no máximo {newPageDays} dias, e a maioria tem impressões. Requer datas de publicação na aba Indexação.` |

- [ ] **Step 3: New strings**

Add inside `playbook` (en shown; zh and pt below, same keys):

```ts
    status: {
      current: "You are at this stage",
      passed: "Passed",
      upcoming: "Not there yet",
      now: "Now",
      met: "Met",
      unmet: "Not met",
      noData: "No data yet",
      manual: "Confirm by hand",
      confirm: "I checked this",
      done: "Done",
      todo: "Not done",
      progress: "{met} of {total} signals met; {needed} pass this stage.",
      yours: "Your project now",
      productStage: "Your product stage",
      unset: "Not chosen yet",
      s0Warn: "You chose S0. This channel is slow at your stage: do the one-time steps, let monitoring run, and spend your week finding users.",
      noSite: "This project has no website, so the SEO playbook does not apply.",
      loadError: "Could not load this project's progress. The playbook still reads the same.",
      saveError: "Could not save. Reload and try again.",
    },
    values: {
      newFast: "median {v} days",
      imprCoverage: "{v}%",
      visibleShare: "{v}%",
      recognized: "lower bound {v}%",
    },
```

zh:

```ts
    status: {
      current: "你当前在这一阶段",
      passed: "已过关",
      upcoming: "还没到这一阶段",
      now: "现在",
      met: "已满足",
      unmet: "未满足",
      noData: "暂无数据",
      manual: "待人工确认",
      confirm: "我已核对",
      done: "已完成",
      todo: "未完成",
      progress: "已满足 {met}/{total} 条，满 {needed} 条算过关。",
      yours: "你的项目现在",
      productStage: "你的产品阶段",
      unset: "还没选",
      s0Warn: "你选了 S0。这个渠道在你的阶段很慢：做完一次性的步骤，让监测自动跑，一周的时间去找用户。",
      noSite: "这个项目没有网站，SEO 剧本不适用。",
      loadError: "没能读取这个项目的进度，剧本内容不受影响。",
      saveError: "没能保存，请刷新后重试。",
    },
    values: {
      newFast: "中位数 {v} 天",
      imprCoverage: "{v}%",
      visibleShare: "{v}%",
      recognized: "下界 {v}%",
    },
```

pt:

```ts
    status: {
      current: "Você está neste estágio",
      passed: "Concluído",
      upcoming: "Ainda não chegou aqui",
      now: "Agora",
      met: "Atendido",
      unmet: "Não atendido",
      noData: "Sem dados ainda",
      manual: "Confirmar manualmente",
      confirm: "Verifiquei",
      done: "Feito",
      todo: "Não feito",
      progress: "{met} de {total} sinais atendidos; com {needed} este estágio está concluído.",
      yours: "Seu projeto agora",
      productStage: "Estágio do seu produto",
      unset: "Ainda não escolhido",
      s0Warn: "Você escolheu S0. Este canal é lento no seu estágio: faça os passos únicos, deixe o monitoramento rodar e use a semana para encontrar usuários.",
      noSite: "Este projeto não tem site, então o roteiro de SEO não se aplica.",
      loadError: "Não foi possível carregar o progresso deste projeto. O roteiro continua o mesmo.",
      saveError: "Não foi possível salvar. Recarregue e tente de novo.",
    },
    values: {
      newFast: "mediana de {v} dias",
      imprCoverage: "{v}%",
      visibleShare: "{v}%",
      recognized: "limite inferior {v}%",
    },
```

In the `overview` block of each locale, add `playbookLink`: en `"New here? Read the playbooks →"`, zh `"刚开始？看看剧本 →"`, pt `"Começando? Veja os roteiros →"`.

- [ ] **Step 4: Verify**

Run: `go test ./internal/service/playbook -count=1 && cd web && ./node_modules/.bin/tsc --noEmit && npm run check:playbook`
Expected: all Go tests PASS (thresholds now match), tsc clean, check ok.

- [ ] **Step 5: Checkpoint** — `git status --short`

---

### Task 7: Show the status in the help center

**Files:**
- Modify: `web/src/api.ts`
- Modify: `web/src/features/help/kit.tsx`, `help.tsx`, `playbook.tsx`

- [ ] **Step 1: API client in `api.ts`** (after `confirmProgress` / the progress functions)

```ts
export type PlaybookCheck = { state: "done" | "todo" | "no_data"; at?: number };
export type PlaybookSignal = { state: "met" | "unmet" | "no_data" | "manual"; value?: number; confirmed: boolean };
export type ProductStage = "" | "s0" | "s1" | "s2";
export type PlaybookStatus = {
  product_stage: ProductStage;
  seo_stage: "" | "seoNew" | "seoGrow";
  ai_stage: "aiReach" | "aiKnown" | "aiRecommend";
  seo_met: number;
  seo_needed: number;
  checks: Record<string, PlaybookCheck>;
  signals: Record<string, PlaybookSignal>;
};
export function getPlaybook(slug: string) {
  return request<PlaybookStatus>(`/api/projects/${slug}/playbook`);
}
export function putProductStage(slug: string, product_stage: ProductStage) {
  return request<PlaybookStatus>(`/api/projects/${slug}/playbook/stage`, { method: "PUT", body: JSON.stringify({ product_stage }) });
}
export function putPlaybookSignal(slug: string, signal: string, confirmed: boolean) {
  return request<PlaybookStatus>(`/api/projects/${slug}/playbook/signals/${signal}`, { method: "PUT", body: JSON.stringify({ confirmed }) });
}
```

- [ ] **Step 2: `Kit` carries the status (`kit.tsx`)**

Add `import type { PlaybookStatus, ProductStage } from "../../api";` and:

```tsx
// PlaybookCtx is the current project's progress for the playbook topics.
// status is null while loading, outside a project, or after an error.
export type PlaybookCtx = {
  status: PlaybookStatus | null;
  error: string;
  canEdit: boolean;
  setStage: (stage: ProductStage) => void;
  confirm: (signal: string, on: boolean) => void;
};
```

Add `playbook: PlaybookCtx;` to `Kit`, add a `playbook: PlaybookCtx` parameter to `makeKit`, and return it in the object.

- [ ] **Step 3: Load it in `help.tsx`**

```tsx
import { getPlaybook, putPlaybookSignal, putProductStage, type PlaybookStatus } from "../../api";
import { useAccess } from "../../app/access";
```

Inside `Help()`, before `kit`:

```tsx
  const { canEdit } = useAccess();
  const [status, setStatus] = useState<PlaybookStatus | null>(null);
  const [statusErr, setStatusErr] = useState("");
  useEffect(() => {
    if (!slug) return;
    let live = true;
    getPlaybook(slug).then((s) => live && setStatus(s)).catch(() => live && setStatusErr("playbook.status.loadError"));
    return () => { live = false; };
  }, [slug]);
  const playbook = useMemo(() => ({
    status, error: statusErr ? t(statusErr as Key) : "", canEdit: canEdit && !!slug,
    setStage: (stage: PlaybookStatus["product_stage"]) =>
      putProductStage(slug, stage).then(setStatus).catch(() => setStatusErr("playbook.status.saveError")),
    confirm: (signal: string, on: boolean) =>
      putPlaybookSignal(slug, signal, on).then(setStatus).catch(() => setStatusErr("playbook.status.saveError")),
  }), [status, statusErr, canEdit, slug, t]);
```

Change `makeKit(slug, go, t)` to `makeKit(slug, go, t, playbook)` and add `playbook` to that `useMemo`'s dependencies. Check `useAccess()` returns `canEdit` for the current project (see `features/settings/schedule.tsx`).

- [ ] **Step 4: Render it in `playbook.tsx`**

Add imports: `IconCircleCheck, IconCircleDashed, IconMinus` from `@tabler/icons-react`; `type PlaybookStatus` from `../../api`; `CHECKS`-free usage is fine.

Add these helpers inside `playbookBodies`, after `where`:

```tsx
  const { status, error, canEdit, setStage, confirm } = k.playbook;
  const order: Record<PlaybookTopic, number> = { start: 0, seoNew: 1, seoGrow: 2, aiReach: 1, aiKnown: 2, aiRecommend: 3, weekly: 0 };
  const stageOf = (id: PlaybookTopic): string | undefined =>
    id.startsWith("seo") ? status?.seo_stage : id.startsWith("ai") ? status?.ai_stage : undefined;
  // banner says where the project is relative to this stage topic.
  const banner = (id: PlaybookTopic) => {
    const cur = stageOf(id);
    if (!status || order[id] === 0) return null;
    if (id.startsWith("seo") && !cur) return <Tip>{t("playbook.status.noSite")}</Tip>;
    const rel = order[id] < order[cur as PlaybookTopic] ? "passed" : order[id] === order[cur as PlaybookTopic] ? "current" : "upcoming";
    const cls = rel === "current" ? "bg-primary-100 text-primary-700" : rel === "passed" ? "bg-emerald-100 text-emerald-700" : "bg-gray-100 text-gray-700";
    return <p><span className={"rounded-full px-2.5 py-0.5 text-xs font-medium " + cls}>{t(`playbook.status.${rel}`)}</span></p>;
  };
  const s0 = (id: PlaybookTopic) =>
    status?.product_stage === "s0" && id !== "start" && id !== "weekly" ? <Warn>{t("playbook.status.s0Warn")}</Warn> : null;
  const mark = (check?: string) => {
    const c = check ? status?.checks[check] : undefined;
    if (!c) return null;
    const label = t(c.state === "done" ? "playbook.status.done" : c.state === "todo" ? "playbook.status.todo" : "playbook.status.noData");
    const Icon = c.state === "done" ? IconCircleCheck : c.state === "todo" ? IconCircleDashed : IconMinus;
    const color = c.state === "done" ? "text-emerald-700" : "text-gray-400";
    return <span className={"mr-1.5 inline-flex items-center gap-1 align-middle text-xs font-medium " + color}><Icon size={16} aria-hidden />{label}</span>;
  };
```

In `steps`, change the `<strong>` line to:

```tsx
            <div>{mark("check" in s ? s.check : undefined)}<strong>{t(`playbook.steps.${s.id}.do`, THRESHOLDS)}</strong></div>
```

Replace `pass` with a version that adds a "Now" column when status exists:

```tsx
  const signalNow = (id: string) => {
    const sg = status?.signals[id];
    if (!sg) return "";
    const word = t(sg.state === "met" ? "playbook.status.met" : sg.state === "unmet" ? "playbook.status.unmet" : sg.state === "manual" ? "playbook.status.manual" : "playbook.status.noData");
    const v = sg.value != null && (id === "newFast" || id === "imprCoverage" || id === "visibleShare" || id === "recognized")
      ? ` · ${t(`playbook.values.${id}`, { v: Math.round(sg.value) })}` : "";
    const manual = (sg.state === "manual" || sg.confirmed) && canEdit ? (
      <label className="ml-2 inline-flex items-center gap-1 text-xs text-gray-600">
        <input type="checkbox" checked={sg.confirmed} onChange={(e) => confirm(id, e.target.checked)} />{t("playbook.status.confirm")}
      </label>
    ) : null;
    const cls = sg.state === "met" ? "bg-emerald-100 text-emerald-700" : sg.state === "unmet" ? "bg-amber-100 text-amber-700" : "bg-gray-100 text-gray-700";
    return <><span className={"rounded-full px-2 py-0.5 text-xs font-medium " + cls}>{word}{v}</span>{manual}</>;
  };
  const pass = (id: PlaybookTopic) => {
    const list = SIGNALS.filter((s) => s.topic === id);
    if (!list.length) return null;
    const head = [t("playbook.signal"), t("playbook.rule"), t("playbook.judged")];
    if (status) head.push(t("playbook.status.now"));
    return <>
      <h3>{t("playbook.pass")}</h3>
      {status && id === "seoNew" && status.seo_stage && <p>{t("playbook.status.progress", { met: status.seo_met, total: list.length, needed: status.seo_needed })}</p>}
      <Table head={head} rows={list.map((s) => {
        const row: ReactNode[] = [t(`playbook.signals.${s.id}.name`), t(`playbook.signals.${s.id}.rule`, THRESHOLDS), s.auto ? t("playbook.auto") : t("playbook.manual")];
        if (status) row.push(signalNow(s.id));
        return row;
      })} />
      <p>{t("playbook.passNote")}</p>
    </>;
  };
```

Add a project summary and the stage picker to `start`. Replace the `stages` constant's first line (`<h3>{t("playbook.stage.title")}</h3>`) with:

```tsx
    {status && <>
      <h3>{t("playbook.status.yours")}</h3>
      <p>
        {t("playbook.status.productStage")}{sep}
        {canEdit ? (
          <select className="input inline-block w-auto py-1" value={status.product_stage}
            onChange={(e) => setStage(e.target.value as PlaybookStatus["product_stage"])} aria-label={t("playbook.status.productStage")}>
            <option value="">{t("playbook.status.unset")}</option>
            {(["s0", "s1", "s2"] as const).map((s) => <option key={s} value={s}>{t(`playbook.stage.${s}.name`)}</option>)}
          </select>
        ) : status.product_stage ? t(`playbook.stage.${status.product_stage}.name`) : t("playbook.status.unset")}
      </p>
      <p>
        {status.seo_stage && <>SEO{sep}{topic(status.seo_stage, t(`helpTopics.${status.seo_stage}.label`))}{" · "}</>}
        AI{sep}{topic(status.ai_stage, t(`helpTopics.${status.ai_stage}.label`))}
      </p>
    </>}
    <h3>{t("playbook.stage.title")}</h3>
```

Finally, in `body`, render the error, banner and S0 warning right after the `<h2>`:

```tsx
    <h2>{t(`helpTopics.${id}.label`)}</h2>
    {error && <p className="text-sm text-red-700">{error}</p>}
    {banner(id)}
    {s0(id)}
```

Notes:
- Colors follow AGENTS.md: primary for "current", emerald/amber/gray badges with words for states; never color alone.
- `"check" in s` narrows the `as const` union; if TypeScript still complains, read `(s as { check?: string }).check`.
- `start` and `weekly` have `order` 0 and get no banner.

- [ ] **Step 5: Verify**

Run: `cd web && ./node_modules/.bin/tsc --noEmit && npm run check:cjk && npm run check:playbook && npm run build`
Expected: all pass.

- [ ] **Step 6: Checkpoint** — `git status --short`

---

### Task 8: One link on the overview

**Files:**
- Modify: `web/src/features/overview/overview.tsx`

- [ ] **Step 1: Add the link**

After the `<FirstCheck … />` line:

```tsx
      <Link to={`/p/${slug}/help#start`} className="self-end text-sm text-primary-600 hover:underline">{t("overview.playbookLink")}</Link>
```

`Link` is already imported. Nothing else on the overview changes.

- [ ] **Step 2: Verify** — `cd web && ./node_modules/.bin/tsc --noEmit && npm run build`

- [ ] **Step 3: Checkpoint** — `git status --short`

---

### Task 9: Full verification

- [ ] **Step 1: Backend**

Run: `test -z "$(gofmt -l .)" && go vet ./... && go test ./... -count=1 && scripts/license-headers.sh`
Expected: all pass. Then `make test-mysql` if Docker is available (report if not).

- [ ] **Step 2: Frontend**

Run: `cd web && ./node_modules/.bin/tsc --noEmit && npm run check:cjk && npm run check:playbook && npm run build`

- [ ] **Step 3: Browser**

`make build`, generate a fresh demo database into a temp dir (`go run ./scripts/demo -lang en -db <tmp>/data/craftsail-growth.db`, copy `config/default.example.toml` to `<tmp>/config/`), run `./craftsail-growth ui --no-open --port 8799` from `<tmp>`, log in as `demo` / `quillpad-demo-2026`. Extend the existing Playwright script `/tmp/pw/cg-help.mjs` (or write `/tmp/pw/cg-status.mjs` with the same login) to check, in en/zh/pt:

1. Overview of `quillpad` shows the "Read the playbooks →" link and nothing else new; it opens `help#start`.
2. Start here shows "Your project now" with a product-stage select and SEO/AI stage links.
3. Choosing S0 persists across reload and shows the S0 warning on SEO and AI topics.
4. `quillpad-new-site`: SEO: new site shows "You are at this stage"; `quillpad-established`: SEO: new site shows "Passed" and SEO: growth shows "You are at this stage".
5. Steps with checks show Done / Not done with an icon; reading-only steps show no mark.
6. The SEO pass table has a "Now" column; ticking "I checked this" on a manual signal changes it to Met and the "x of 8" count goes up by one; unticking reverts.
7. A view-only user (create one with `users` admin page or the API) sees states but no select and no checkboxes.
8. No page errors, no literal `{…}` or raw keys.

- [ ] **Step 4: Report** what passed, what failed, the three spec deviations listed at the top, and that nothing is committed.
