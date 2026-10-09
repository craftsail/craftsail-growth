// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MySQL's default text collation can conflate URL-prefix properties that differ
// in path case. Keep exact property identity and URL discovery on both databases.
func (r *Webstats) indexBinary(column string) string {
	if r.DB.Dialector.Name() == "mysql" {
		return "CAST(" + column + " AS BINARY)"
	}
	return column + " COLLATE BINARY"
}

type IndexCandidate struct {
	URL                   string
	FromCrawl, FromSearch bool
}

// IndexCandidates does not cap discovery. The inspection cap applies only after
// scope and freshness filters, so the first URLs cannot starve the rest.
func (r *Webstats) IndexCandidates(ctx context.Context, projectID uint64, property string) ([]IndexCandidate, error) {
	var pages, search []string
	db := r.DB.WithContext(ctx)
	if err := db.Model(&model.Page{}).Where("project_id = ?", projectID).Distinct().Pluck("url", &pages).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&model.GscFact{}).Where("project_id = ? AND "+r.indexBinary("property")+" = ? AND slice IN ? AND page <> ''", projectID, property, []string{"page", "query_page"}).Select("DISTINCT " + r.indexBinary("page") + " AS url").Scan(&search).Error; err != nil {
		return nil, err
	}
	out := make([]IndexCandidate, 0, len(pages)+len(search))
	for _, u := range pages {
		out = append(out, IndexCandidate{URL: u, FromCrawl: true})
	}
	for _, u := range search {
		out = append(out, IndexCandidate{URL: u, FromSearch: true})
	}
	return out, nil
}

func (r *Webstats) DiscoverIndexURLs(ctx context.Context, rows []model.IndexURL) error {
	// Merge each source independently, retaining earlier provenance and scheduling.
	for _, source := range []string{"from_crawl", "from_search", "from_sitemap"} {
		batch := []model.IndexURL{}
		for _, row := range rows {
			if source == "from_crawl" && row.FromCrawl || source == "from_search" && row.FromSearch || source == "from_sitemap" && row.FromSitemap {
				row.KeyHash = model.RowKey(row.Property, row.URL)
				batch = append(batch, row)
			}
		}
		if len(batch) == 0 {
			continue
		}
		if err := r.DB.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "project_id"}, {Name: "key_hash"}}, DoUpdates: clause.AssignmentColumns([]string{"last_seen_at", source, "content_group"})}).CreateInBatches(&batch, upsertBatchSize).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *Webstats) DueIndexURLs(ctx context.Context, projectID uint64, property string, now int64, limit int) ([]model.IndexURL, error) {
	var out []model.IndexURL
	err := r.DB.WithContext(ctx).Where("project_id = ? AND "+r.indexBinary("property")+" = ? AND next_inspect_at <= ?", projectID, property, now).
		Order("CASE WHEN last_attempt_at = 0 THEN 0 ELSE 1 END").Order("next_inspect_at").Order("last_attempt_at").Order("id").Limit(limit).Find(&out).Error
	return out, err
}

// RecordInspection atomically saves history, latest successful result and the
// next due time. Failure changes neither the verdict nor first observed indexing.
func (r *Webstats) RecordInspection(ctx context.Context, target model.IndexURL, result *model.GscIndex, failure string, now time.Time) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		history := model.IndexInspection{ProjectID: target.ProjectID, KeyHash: target.KeyHash, Property: target.Property, URL: target.URL, CheckedAt: now.Unix(), Error: failure}
		interval := 6 * time.Hour
		updates := map[string]any{"last_attempt_at": now.Unix(), "last_error": failure}
		if result != nil {
			row := *result
			row.ID = 0
			row.ProjectID = target.ProjectID
			row.KeyHash = target.KeyHash
			row.Property = target.Property
			row.URL = target.URL
			row.FetchedAt = now.Unix()
			history.Result = &row
			if err := (&Webstats{DB: tx}).UpsertIndex(ctx, []model.GscIndex{row}); err != nil {
				return err
			}
			updates["last_success_at"] = now.Unix()
			updates["verdict"] = row.Verdict
			updates["last_error"] = ""
			interval = 24 * time.Hour
			if row.Verdict == "PASS" {
				interval = 7 * 24 * time.Hour
				updates["first_indexed_at"] = gorm.Expr("COALESCE(first_indexed_at, ?)", now.Unix())
			}
		}
		updates["next_inspect_at"] = now.Add(interval).Unix()
		if err := tx.Create(&history).Error; err != nil {
			return err
		}
		res := tx.Model(&model.IndexURL{}).Where("id = ? AND project_id = ? AND key_hash = ?", target.ID, target.ProjectID, target.KeyHash).Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

type IndexInventory struct {
	Sitemaps          []model.SitemapScan `json:"sitemaps"`
	Quota             *model.GoogleQuota  `json:"quota"`
	PublishedMature   int64               `json:"published_mature"`
	IndexedWithinWeek int64               `json:"indexed_within_week"`

	Property  string           `json:"property"`
	Items     []model.IndexURL `json:"items"`
	Total     int64            `json:"total"`
	Known     int64            `json:"known"`
	Inspected int64            `json:"inspected"`
	Indexed   int64            `json:"indexed"`
	Due       int64            `json:"due"`
	Page      int              `json:"page"`
	PageSize  int              `json:"page_size"`
}

func (r *Webstats) IndexInventory(ctx context.Context, projectID uint64, property, query, state string, page, size int, now int64, groups ...string) (*IndexInventory, error) {
	out := &IndexInventory{Property: property, Items: []model.IndexURL{}, Page: page, PageSize: size}
	base := func() *gorm.DB {
		return r.DB.WithContext(ctx).Model(&model.IndexURL{}).Where("project_id = ? AND "+r.indexBinary("property")+" = ?", projectID, property)
	}
	for _, c := range []struct {
		condition string
		count     *int64
	}{{"1 = 1", &out.Known}, {"last_success_at > 0", &out.Inspected}, {"verdict = 'PASS'", &out.Indexed}} {
		if err := base().Where(c.condition).Count(c.count).Error; err != nil {
			return nil, err
		}
	}
	if err := base().Where("next_inspect_at <= ?", now).Count(&out.Due).Error; err != nil {
		return nil, err
	}
	if err := base().Where("published_at IS NOT NULL AND published_at <= ?", now-7*86400).Count(&out.PublishedMature).Error; err != nil {
		return nil, err
	}
	if err := base().Where("published_at IS NOT NULL AND published_at <= ? AND first_indexed_at >= published_at AND first_indexed_at <= published_at + ?", now-7*86400, 7*86400).Count(&out.IndexedWithinWeek).Error; err != nil {
		return nil, err
	}
	db := base()
	if len(groups) > 0 && groups[0] != "" {
		db = db.Where("content_group = ?", groups[0])
	}
	if query != "" {
		escaped := strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(strings.ToLower(query))
		db = db.Where("LOWER(url) LIKE ? ESCAPE '!'", "%"+escaped+"%")
	}
	switch state {
	case "pending":
		db = db.Where("last_success_at = 0")
	case "indexed":
		db = db.Where("verdict = ?", "PASS")
	case "other":
		db = db.Where("last_success_at > 0 AND verdict <> ?", "PASS")
	case "error":
		db = db.Where("last_error <> ''")
	case "due":
		db = db.Where("next_inspect_at <= ?", now)
	}
	if err := db.Count(&out.Total).Error; err != nil {
		return nil, err
	}
	if err := db.Order("url").Order("id").Offset((page - 1) * size).Limit(size).Find(&out.Items).Error; err != nil {
		return nil, err
	}
	keys := []string{}
	for _, row := range out.Items {
		keys = append(keys, row.KeyHash)
	}
	if len(keys) > 0 {
		var latest []model.GscIndex
		if err := r.DB.WithContext(ctx).Where("project_id = ? AND property = ? AND key_hash IN ?", projectID, property, keys).Find(&latest).Error; err != nil {
			return nil, err
		}
		byKey := map[string]*model.GscIndex{}
		for i := range latest {
			latest[i].Raw = ""
			byKey[latest[i].KeyHash] = &latest[i]
		}
		for i := range out.Items {
			out.Items[i].Latest = byKey[out.Items[i].KeyHash]
		}
	}
	if err := r.enrichIndexInventory(ctx, projectID, property, out); err != nil {
		return nil, err
	}
	return out, nil
}

type InspectionHistory struct {
	Items    []model.IndexInspection `json:"items"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"page_size"`
}

func (r *Webstats) IndexHistory(ctx context.Context, projectID uint64, property, url string, page, size int) (*InspectionHistory, error) {
	out := &InspectionHistory{Items: []model.IndexInspection{}, Page: page, PageSize: size}
	db := r.DB.WithContext(ctx).Model(&model.IndexInspection{}).Where("project_id = ? AND key_hash = ? AND property = ?", projectID, model.RowKey(property, url), property)
	if err := db.Count(&out.Total).Error; err != nil {
		return nil, err
	}
	if err := db.Order("checked_at desc").Order("id desc").Offset((page - 1) * size).Limit(size).Find(&out.Items).Error; err != nil {
		return nil, err
	}
	for i := range out.Items {
		if out.Items[i].Result != nil {
			out.Items[i].Result.Raw = ""
		}
	}
	return out, nil
}

// RefreshIndexImpressions records observed dates from the independent page grain.
// It does not infer publication dates or a complete historical absence of traffic.
func (r *Webstats) RefreshIndexImpressions(ctx context.Context, pid uint64, property string, now int64) error {
	type observed struct{ URL, FirstDay, LastDay string }
	var facts []observed
	expr := r.indexBinary("page")
	if err := r.DB.WithContext(ctx).Model(&model.GscFact{}).Select(expr+" AS url, MIN(date(day)) AS first_day, MAX(date(day)) AS last_day").Where("project_id = ? AND "+r.indexBinary("property")+" = ? AND slice = ? AND search_type = ? AND impressions > 0", pid, property, "page", "web").Group(expr).Scan(&facts).Error; err != nil {
		return err
	}
	incoming := func(field string) string {
		if r.DB.Dialector.Name() == "mysql" {
			return "VALUES(" + field + ")"
		}
		return "excluded." + field
	}
	rows := []model.IndexURL{}
	for _, f := range facts {
		first, e1 := time.Parse("2006-01-02", f.FirstDay)
		last, e2 := time.Parse("2006-01-02", f.LastDay)
		if e1 != nil || e2 != nil {
			return fmt.Errorf("invalid stored impression date")
		}
		a, b := first.Unix(), last.Unix()
		rows = append(rows, model.IndexURL{ProjectID: pid, Property: property, KeyHash: model.RowKey(property, f.URL), URL: f.URL, FromSearch: true, FirstSeenAt: now, LastSeenAt: now, FirstImpressionAt: &a, LastImpressionAt: &b})
	}
	if len(rows) == 0 {
		return nil
	}
	a, b := incoming("first_impression_at"), incoming("last_impression_at")
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "project_id"}, {Name: "key_hash"}}, DoUpdates: clause.Assignments(map[string]any{"first_impression_at": gorm.Expr("CASE WHEN first_impression_at IS NULL OR first_impression_at > " + a + " THEN " + a + " ELSE first_impression_at END"), "last_impression_at": gorm.Expr("CASE WHEN last_impression_at IS NULL OR last_impression_at < " + b + " THEN " + b + " ELSE last_impression_at END"), "from_search": true})}).CreateInBatches(&rows, upsertBatchSize).Error
}
func (r *Webstats) enrichIndexInventory(ctx context.Context, pid uint64, property string, out *IndexInventory) error {
	var err error
	out.Sitemaps, err = r.SitemapStatus(ctx, pid, property)
	if err != nil {
		return err
	}
	out.Quota, err = r.GoogleQuota(ctx, property, "inspection")
	if err != nil {
		return err
	}
	if len(out.Items) == 0 {
		return nil
	}
	keys := []string{}
	urls := []string{}
	for _, row := range out.Items {
		keys = append(keys, row.KeyHash)
		urls = append(urls, row.URL)
	}
	type membership struct{ URLKey, URL string }
	var maps []membership
	if err := r.DB.WithContext(ctx).Table("sitemap_urls AS u").Select("u.url_key, s.url").Joins("JOIN sitemap_scans AS s ON s.id = u.sitemap_id AND s.project_id = u.project_id").Where("u.project_id = ? AND u.url_key IN ? AND u.present = ?", pid, keys, true).Scan(&maps).Error; err != nil {
		return err
	}
	byKey := map[string][]string{}
	for _, m := range maps {
		byKey[m.URLKey] = append(byKey[m.URLKey], m.URL)
	}
	var pages []model.Page
	if err := r.DB.WithContext(ctx).Select("url", "status_code", "final_url", "fetched_at", "fetch_error", "analysis").Where("project_id = ? AND url IN ?", pid, urls).Find(&pages).Error; err != nil {
		return err
	}
	byURL := map[string]*model.Page{}
	for i := range pages {
		p := &pages[i]
		limited := map[string]any{}
		for _, k := range []string{"canonical", "meta_robots", "x_robots_tag"} {
			limited[k] = p.Analysis[k]
		}
		p.Analysis = limited
		byURL[p.URL] = p
	}
	for i := range out.Items {
		out.Items[i].Sitemaps = byKey[out.Items[i].KeyHash]
		out.Items[i].Crawl = byURL[out.Items[i].URL]
	}
	return nil
}
