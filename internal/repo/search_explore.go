// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"gorm.io/gorm"
)

// SearchFilter selects one explicit grain. Exact filters are for drill-down;
// text search is a literal, case-insensitive substring applied before paging.
type SearchFilter struct {
	SearchType, Brand                                                    string
	BrandTerms                                                           []string
	ProjectID                                                            uint64
	Property, Slice, Group, Text, Country, Device, ExactQuery, ExactPage string
	From, Through, PreviousFrom                                          time.Time
	Sort, Direction                                                      string
	Offset, Limit                                                        int
}

type SearchMetric struct {
	CurrentRows         int64   `json:"current_rows"`
	PreviousRows        int64   `json:"previous_rows"`
	Name                string  `json:"name"`
	Clicks              float64 `json:"clicks"`
	Impressions         float64 `json:"impressions"`
	CTR                 float64 `json:"ctr"`
	Position            float64 `json:"position"`
	PreviousClicks      float64 `json:"previous_clicks"`
	PreviousImpressions float64 `json:"previous_impressions"`
	PreviousCTR         float64 `json:"previous_ctr"`
	PreviousPosition    float64 `json:"previous_position"`
	ClicksChange        float64 `json:"clicks_change"`
}

func (r *Webstats) searchQuery(ctx context.Context, f SearchFilter) (*gorm.DB, error) {
	if f.Group != "query" && f.Group != "page" && f.Group != "day" && f.Group != "country" && f.Group != "device" {
		return nil, fmt.Errorf("invalid search grouping")
	}
	// Binary grouping keeps case-sensitive URLs distinct on MySQL as on SQLite.
	key := "`" + f.Group + "` COLLATE BINARY"
	if r.DB.Dialector.Name() == "mysql" {
		key = "CAST(`" + f.Group + "` AS BINARY)"
	}
	if f.Group == "day" {
		key = "date(day)"
	}
	if f.SearchType == "" {
		f.SearchType = "web"
	}
	q := r.DB.WithContext(ctx).Model(&model.GscFact{}).Where("project_id = ? AND property = ? AND slice = ? AND search_type = ? AND date(day) >= ? AND date(day) <= ?", f.ProjectID, f.Property, f.Slice, f.SearchType, f.PreviousFrom.Format("2006-01-02"), f.Through.Format("2006-01-02"))
	if f.Brand != "" {
		var terms []string
		var args []any
		for _, term := range f.BrandTerms {
			if term = strings.TrimSpace(term); term != "" {
				terms = append(terms, "LOWER(`query`) LIKE ? ESCAPE '!' ")
				escaped := strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(strings.ToLower(term))
				args = append(args, "%"+escaped+"%")
			}
		}
		condition := "1 = 0"
		if len(terms) > 0 {
			condition = "(" + strings.Join(terms, " OR ") + ")"
		}
		if f.Brand == "nonbrand" {
			condition = "NOT (" + condition + ")"
		}
		q = q.Where(condition, args...)
	}
	if f.Country != "" {
		q = q.Where("country = ?", f.Country)
	}
	if f.Device != "" {
		q = q.Where("device = ?", f.Device)
	}
	exact := func(column, value string) {
		if value == "" {
			return
		}
		expr := column + " COLLATE BINARY"
		if r.DB.Dialector.Name() == "mysql" {
			expr = "CAST(" + column + " AS BINARY)"
		}
		q = q.Where(expr+" = ?", value)
	}
	exact("`query`", f.ExactQuery)
	exact("page", f.ExactPage)
	if f.Text != "" {
		escaped := strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(strings.ToLower(f.Text))
		q = q.Where("LOWER(`"+f.Group+"`) LIKE ? ESCAPE '!'", "%"+escaped+"%")
	}
	if f.Group != "day" {
		q = q.Where("`"+f.Group+"` <> ?", "")
	}
	start := f.From.Format("2006-01-02") // Parsed date only; never user SQL.
	sum := func(field string, previous bool) string {
		op := ">="
		if previous {
			op = "<"
		}
		return fmt.Sprintf("SUM(CASE WHEN date(day) %s '%s' THEN %s ELSE 0 END)", op, start, field)
	}
	clicks, impressions := sum("clicks", false), sum("impressions", false)
	pc, pi := sum("clicks", true), sum("impressions", true)
	ratio := func(n, d string) string { return "CASE WHEN " + d + " > 0 THEN 1.0 * " + n + " / " + d + " ELSE 0 END" }
	selectSQL := sum("1", false) + " AS current_rows, " + sum("1", true) + " AS previous_rows, " + key + " AS name, " + clicks + " AS clicks, " + impressions + " AS impressions, " + ratio(clicks, impressions) + " AS ctr, " + ratio(sum("position * impressions", false), impressions) + " AS position, " + pc + " AS previous_clicks, " + pi + " AS previous_impressions, " + ratio(pc, pi) + " AS previous_ctr, " + ratio(sum("position * impressions", true), pi) + " AS previous_position, (" + clicks + " - " + pc + ") AS clicks_change"
	return q.Select(selectSQL).Group(key), nil
}

func searchOrder(f SearchFilter) (string, error) {
	column := f.Sort
	if column == "" {
		column = "clicks"
	}
	switch column {
	case "clicks", "impressions", "ctr", "position", "clicks_change", "name":
	default:
		return "", fmt.Errorf("invalid search sort")
	}
	dir := f.Direction
	if dir == "" {
		dir = "desc"
		if column == "position" || column == "name" {
			dir = "asc"
		}
	}
	if dir != "asc" && dir != "desc" {
		return "", fmt.Errorf("invalid search direction")
	}
	prefix := ""
	if column == "position" || column == "ctr" {
		prefix = "CASE WHEN impressions = 0 THEN 1 ELSE 0 END asc, "
	}
	return prefix + column + " " + dir + ", name asc", nil
}

func (r *Webstats) SearchMetrics(ctx context.Context, f SearchFilter) ([]SearchMetric, int64, error) {
	order, err := searchOrder(f)
	if err != nil {
		return nil, 0, err
	}
	if f.Limit < 1 || f.Limit > 200 || f.Offset < 0 {
		return nil, 0, fmt.Errorf("invalid search pagination")
	}
	var rows []SearchMetric
	var count int64
	err = r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rr := &Webstats{DB: tx}
		q, err := rr.searchQuery(ctx, f)
		if err != nil {
			return err
		}
		if err := tx.Table("(?) AS grouped", q).Count(&count).Error; err != nil {
			return err
		}
		q, err = rr.searchQuery(ctx, f)
		if err != nil {
			return err
		}
		return tx.Table("(?) AS search_result", q).Order(order).Offset(f.Offset).Limit(f.Limit).Scan(&rows).Error
	})
	return rows, count, err
}

// WalkSearchMetrics streams one grouped result set without a hidden row cap.
func (r *Webstats) WalkSearchMetrics(ctx context.Context, f SearchFilter, visit func(SearchMetric) error) error {
	order, err := searchOrder(f)
	if err != nil {
		return err
	}
	q, err := r.searchQuery(ctx, f)
	if err != nil {
		return err
	}
	rows, err := r.DB.WithContext(ctx).Table("(?) AS search_result", q).Order(order).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var row SearchMetric
		if err := r.DB.ScanRows(rows, &row); err != nil {
			return err
		}
		if err := visit(row); err != nil {
			return err
		}
	}
	return rows.Err()
}
