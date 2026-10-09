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

type GAFilter struct {
	Country, Device                         string
	Events                                  []string
	ProjectID                               uint64
	Property, Report, Text, Sort, Direction string
	From, Through, PreviousFrom             time.Time
	Offset, Limit                           int
}
type GAMetric struct {
	EventName          string   `json:"event_name"`
	EventCount         float64  `json:"event_count"`
	PreviousEventCount float64  `json:"previous_event_count"`
	EventChange        float64  `json:"event_change"`
	Channel            string   `json:"channel"`
	Source             string   `json:"source"`
	Medium             string   `json:"medium"`
	Landing            string   `json:"landing"`
	CurrentRows        int64    `json:"current_rows"`
	PreviousRows       int64    `json:"previous_rows"`
	Sessions           float64  `json:"sessions"`
	Engaged            float64  `json:"engaged"`
	KeyEvents          float64  `json:"key_events"`
	Duration           *float64 `json:"duration"`
	EngagementRate     *float64 `json:"engagement_rate"`
	DurationPerSession *float64 `json:"duration_per_session"`
	PreviousSessions   float64  `json:"previous_sessions"`
	SessionsChange     float64  `json:"sessions_change"`
}

func (r *Webstats) gaQuery(ctx context.Context, f GAFilter) (*gorm.DB, []string, error) {
	fields := []string{"channel", "source", "medium"}
	if strings.HasPrefix(f.Report, "landing") {
		fields = []string{"landing"}
	} else if !strings.HasPrefix(f.Report, "channel") {
		return nil, nil, fmt.Errorf("invalid GA report")
	}
	if strings.HasSuffix(f.Report, "_event") {
		fields = append(fields, "event_name")
	}
	q := r.DB.WithContext(ctx).Model(&model.GaFact{}).Where("project_id = ? AND property = ? AND report = ? AND date(day) >= ? AND date(day) <= ?", f.ProjectID, f.Property, f.Report, f.PreviousFrom.Format("2006-01-02"), f.Through.Format("2006-01-02"))
	if f.Country != "" {
		q = q.Where("country = ?", f.Country)
	}
	if f.Device != "" {
		q = q.Where("device = ?", f.Device)
	}
	if len(f.Events) > 0 {
		// Event names are exact identifiers, including case, on both databases.
		event := "event_name COLLATE BINARY"
		if r.DB.Dialector.Name() == "mysql" {
			event = "CAST(event_name AS BINARY)"
		}
		q = q.Where(event+" IN ?", f.Events)
	}
	if f.Text != "" {
		needle := "%" + strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(strings.ToLower(f.Text)) + "%"
		var clauses []string
		var args []any
		for _, field := range fields {
			clauses = append(clauses, "LOWER(`"+field+"`) LIKE ? ESCAPE '!'")
			args = append(args, needle)
		}
		q = q.Where("("+strings.Join(clauses, " OR ")+")", args...)
	}
	var selects, groups []string
	for _, field := range fields {
		expr := "`" + field + "` COLLATE BINARY"
		if r.DB.Dialector.Name() == "mysql" {
			expr = "CAST(`" + field + "` AS BINARY)"
		}
		selects = append(selects, expr+" AS "+field)
		groups = append(groups, expr)
	}
	start := f.From.Format("2006-01-02")
	sum := func(field string, previous bool) string {
		op := ">="
		if previous {
			op = "<"
		}
		return fmt.Sprintf("SUM(CASE WHEN date(day) %s '%s' THEN %s ELSE 0 END)", op, start, field)
	}
	sessions, previous, engaged := sum("sessions", false), sum("sessions", true), sum("engaged", false)
	duration := "CASE WHEN " + sum("CASE WHEN engagement_duration IS NULL THEN 1 ELSE 0 END", false) + " > 0 THEN NULL ELSE " + sum("engagement_duration", false) + " END"
	ratio := func(n, d string) string {
		return "CASE WHEN " + d + " > 0 THEN 1.0 * (" + n + ") / " + d + " ELSE NULL END"
	}
	selects = append(selects, sum("event_count", false)+" AS event_count", sum("event_count", true)+" AS previous_event_count", "("+sum("event_count", false)+" - "+sum("event_count", true)+") AS event_change", sum("1", false)+" AS current_rows", sum("1", true)+" AS previous_rows", sessions+" AS sessions", previous+" AS previous_sessions", "("+sessions+" - "+previous+") AS sessions_change", engaged+" AS engaged", sum("key_events", false)+" AS key_events", duration+" AS duration", ratio(engaged, sessions)+" AS engagement_rate", ratio(duration, sessions)+" AS duration_per_session")
	return q.Select(strings.Join(selects, ", ")).Group(strings.Join(groups, ", ")), fields, nil
}
func gaOrder(f GAFilter, fields []string) (string, error) {
	switch f.Sort {
	case "event_count", "event_change", "sessions", "engaged", "key_events", "engagement_rate", "duration", "duration_per_session", "sessions_change":
	default:
		return "", fmt.Errorf("invalid GA sort")
	}
	if f.Direction != "asc" && f.Direction != "desc" {
		return "", fmt.Errorf("invalid GA direction")
	}
	return "CASE WHEN " + f.Sort + " IS NULL THEN 1 ELSE 0 END asc, " + f.Sort + " " + f.Direction + ", " + strings.Join(fields, " asc, ") + " asc", nil
}
func (r *Webstats) GAMetrics(ctx context.Context, f GAFilter) ([]GAMetric, int64, error) {
	if f.Limit < 1 || f.Limit > 200 || f.Offset < 0 {
		return nil, 0, fmt.Errorf("invalid GA pagination")
	}
	var rows []GAMetric
	var count int64
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rr := &Webstats{DB: tx}
		q, fields, err := rr.gaQuery(ctx, f)
		if err != nil {
			return err
		}
		order, err := gaOrder(f, fields)
		if err != nil {
			return err
		}
		if err := tx.Table("(?) AS grouped", q).Count(&count).Error; err != nil {
			return err
		}
		q, _, err = rr.gaQuery(ctx, f)
		if err != nil {
			return err
		}
		return tx.Table("(?) AS ga_result", q).Order(order).Offset(f.Offset).Limit(f.Limit).Scan(&rows).Error
	})
	return rows, count, err
}
func (r *Webstats) WalkGAMetrics(ctx context.Context, f GAFilter, visit func(GAMetric) error) error {
	q, fields, err := r.gaQuery(ctx, f)
	if err != nil {
		return err
	}
	order, err := gaOrder(f, fields)
	if err != nil {
		return err
	}
	rows, err := r.DB.WithContext(ctx).Table("(?) AS ga_result", q).Order(order).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var row GAMetric
		if err := r.DB.ScanRows(rows, &row); err != nil {
			return err
		}
		if err := visit(row); err != nil {
			return err
		}
	}
	return rows.Err()
}
