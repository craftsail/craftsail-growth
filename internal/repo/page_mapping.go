// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"github.com/craftsail/craftsail-growth/internal/model"
	"time"
)

type LandingHost struct {
	Hostname string `json:"hostname"`
	Landing  string `json:"landing"`
}

func (r *Webstats) LandingHosts(ctx context.Context, pid uint64, property, landing string, from, to time.Time) ([]LandingHost, error) {
	var rows []LandingHost
	exact := "landing COLLATE BINARY"
	if r.DB.Dialector.Name() == "mysql" {
		exact = "CAST(landing AS BINARY)"
	}
	err := r.DB.WithContext(ctx).Model(&model.GaFact{}).Select("hostname, landing").Distinct().Where("project_id = ? AND property = ? AND report = ? AND date(day) >= ? AND date(day) <= ?", pid, property, "landing_context", from.Format("2006-01-02"), to.Format("2006-01-02")).Where(exact+" = ?", landing).Limit(201).Scan(&rows).Error
	return rows, err
}
func (r *Webstats) SearchPageURLs(ctx context.Context, pid uint64, property string, from, to time.Time) ([]string, error) {
	var rows []string
	expr := "page COLLATE BINARY"
	if r.DB.Dialector.Name() == "mysql" {
		expr = "CAST(page AS BINARY)"
	}
	err := r.DB.WithContext(ctx).Model(&model.GscFact{}).Where("project_id = ? AND property = ? AND slice = ? AND search_type = ? AND date(day) >= ? AND date(day) <= ?", pid, property, "page", "web", from.Format("2006-01-02"), to.Format("2006-01-02")).Distinct(expr).Pluck(expr, &rows).Error
	return rows, err
}
