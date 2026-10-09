// SPDX-License-Identifier: AGPL-3.0-or-later

package plan

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/metrics"
)

var ErrObservation = errors.New("invalid observation: choose a metric, hypothesis, past release time and targets; wait 0–90 days, window 28–90 days")
var ErrObservationMissing = errors.New("observation not found")

func (s *Service) Observations(ctx context.Context, slug, code string) ([]model.Observation, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	return s.observations.List(ctx, p.ID, code)
}
func (s *Service) Release(ctx context.Context, slug, code string, input model.Observation) (*model.Observation, error) {
	task, err := s.Get(ctx, slug, code)
	if err != nil {
		return nil, err
	}
	if input.WindowDays == 0 {
		input.WindowDays = 28
	}
	if strings.TrimSpace(input.Hypothesis) == "" || len(input.Hypothesis) > 4000 || len(input.Notes) > 8000 || len(input.Guardrails) > 4000 || len(input.Owner) > 128 || input.ReleasedAt <= 0 || input.ReleasedAt > s.Now().Unix() || input.WaitDays < 0 || input.WaitDays > 90 || input.WindowDays < 28 || input.WindowDays > 90 || input.EffortHours < 0 || input.EffortHours > 10000 || math.IsNaN(input.EffortHours) || math.IsInf(input.EffortHours, 0) {
		return nil, ErrObservation
	}
	input.URLs = unique(input.URLs)
	input.ControlURLs = unique(input.ControlURLs)
	input.QIDs = unique(input.QIDs)
	if len(input.URLs) > 100 || len(input.ControlURLs) > 100 || len(input.QIDs) > 100 {
		return nil, ErrObservation
	}
	targets := map[string]bool{}
	for _, u := range input.URLs {
		targets[u] = true
	}
	for _, u := range append(append([]string{}, input.URLs...), input.ControlURLs...) {
		v, e := url.Parse(u)
		if e != nil || v.Host == "" || (v.Scheme != "http" && v.Scheme != "https") || v.User != nil || len(u) > 8192 {
			return nil, ErrObservation
		}
	}
	for _, u := range input.ControlURLs {
		if targets[u] {
			return nil, ErrObservation
		}
	}
	input.ID = 0
	input.ProjectID = task.ProjectID
	input.TaskID = task.ID
	input.TaskCode = task.Code
	input.CreatedAt = s.Now().Unix()
	input.Results = nil
	input.Property = ""
	input.PromptRevision = ""
	input.Baseline = model.Measurement{}
	input.ControlBaseline = model.Measurement{}
	switch input.Metric {
	case "clicks", "impressions":
		if len(input.URLs) == 0 {
			return nil, ErrObservation
		}
	case "technical":
		check, _ := task.Acceptance["check"].(string)
		if !strings.HasPrefix(check, "issue.absent:") || len(input.URLs) == 0 {
			return nil, ErrObservation
		}
		input.Property = strings.TrimPrefix(check, "issue.absent:")
	case "visibility":
		if len(input.QIDs) == 0 || (input.Engine == "" || len(input.Engine) > 32) || (input.Access != "api" && input.Access != "web") {
			return nil, ErrObservation
		}
		revision, e := s.promptRevision(ctx, task.ProjectID, input.QIDs)
		if e != nil {
			return nil, e
		}
		input.PromptRevision = revision
	default:
		return nil, ErrObservation
	}
	day := time.Unix(input.ReleasedAt, 0).UTC()
	if input.Metric == "clicks" || input.Metric == "impressions" {
		zone, e := time.LoadLocation("America/Los_Angeles")
		if e != nil {
			return nil, e
		}
		day = day.In(zone)
	}
	releaseDay := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	input.BaselineThrough = releaseDay.AddDate(0, 0, -1).Format("2006-01-02")
	input.BaselineFrom = releaseDay.AddDate(0, 0, -input.WindowDays).Format("2006-01-02")
	input.FollowupFrom = releaseDay.AddDate(0, 0, 1+input.WaitDays).Format("2006-01-02")
	input.FollowupThrough = releaseDay.AddDate(0, 0, input.WaitDays+input.WindowDays).Format("2006-01-02")
	input.Baseline, err = s.measure(ctx, slug, &input, true, false)
	if err != nil {
		return nil, err
	}
	if len(input.ControlURLs) > 0 {
		input.ControlBaseline, err = s.measure(ctx, slug, &input, true, true)
		if err != nil {
			return nil, err
		}
	}
	if err := s.observations.Create(ctx, &input); err != nil {
		return nil, err
	}
	return &input, nil
}
func unique(rows []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, r := range rows {
		r = strings.TrimSpace(r)
		if r != "" && !seen[r] {
			seen[r] = true
			out = append(out, r)
		}
	}
	sort.Strings(out)
	return out
}
func (s *Service) promptRevision(ctx context.Context, pid uint64, qids []string) (string, error) {
	qs, err := (&repo.Questions{DB: s.tasks.DB}).List(ctx, pid)
	if err != nil {
		return "", err
	}
	wanted := map[string]bool{}
	for _, q := range qids {
		wanted[q] = true
	}
	selected := []model.Question{}
	for _, q := range qs {
		if wanted[q.QID] && q.Enabled {
			selected = append(selected, q)
		}
	}
	if len(selected) != len(wanted) {
		return "", ErrObservation
	}
	return model.QuestionsReviewRevision(selected), nil
}
func (s *Service) measure(ctx context.Context, slug string, o *model.Observation, baseline, control bool) (model.Measurement, error) {
	from, through := o.FollowupFrom, o.FollowupThrough
	if baseline {
		from, through = o.BaselineFrom, o.BaselineThrough
	}
	urls := o.URLs
	if control {
		urls = o.ControlURLs
	}
	if o.Metric == "technical" {
		return s.observations.Audit(ctx, o.ProjectID, urls, o.Property)
	}
	if o.Metric == "visibility" {
		rows, err := s.observations.Samples(ctx, o.ProjectID, o.Engine, o.Access, from, through, o.QIDs)
		if err != nil {
			return model.Measurement{}, err
		}
		return aiMeasurement(rows, o.QIDs), nil
	}
	m, property, err := s.web.ObservePages(ctx, slug, o.Property, from, through, o.Metric, urls)
	if o.Property == "" {
		o.Property = property
	}
	return m, err
}
func aiMeasurement(rows []model.Sample, qids []string) model.Measurement {
	out := model.Measurement{Valid: true, Strata: map[string][2]int{}}
	signatures := map[string]string{}
	for _, r := range rows {
		if !r.OK || r.BrandInQuestion {
			continue
		}
		modelName, _ := r.Raw["model"].(string)
		strategy, known := r.Raw["searched"]
		if r.SampleMode == "web" || r.SampleMode == "manual" {
			strategy, known = r.Raw["session_mode"]
			if value, ok := strategy.(string); !ok || value == "" {
				known = false
			}
		}
		if modelName == "" || !known || r.QuestionText == "" {
			out.Valid = false
			out.Reason = "sample_metadata"
			continue
		}
		sig, _ := json.Marshal([]any{r.Platform, r.SampleMode, modelName, strategy, r.QuestionText, r.Raw["prompt_revision"], r.Raw["sampling_language"], r.Raw["target_region"], r.Raw["strategy_version"], r.PromptRevision, r.SamplingLanguage, r.TargetRegion})
		if old := signatures[r.QID]; old != "" && old != string(sig) {
			out.Valid = false
			out.Reason = "sample_changed"
		}
		signatures[r.QID] = string(sig)
		counts := out.Strata[r.QID]
		counts[1]++
		if r.Mentioned {
			counts[0]++
			out.Value++
		}
		out.Count++
		out.Strata[r.QID] = counts
	}
	for _, q := range qids {
		if out.Strata[q][1] < 5 {
			out.Valid = false
			if out.Reason == "" {
				out.Reason = "small_sample"
			}
		}
	}
	out.Exposure = float64(out.Count)
	bytes, _ := json.Marshal(signatures)
	out.Signature = model.RowKey(string(bytes))
	return out
}
func (s *Service) Evaluate(ctx context.Context, slug, code string, id uint64, refresh bool, notes string) (*model.ObservationResult, error) {
	rows, err := s.Observations(ctx, slug, code)
	if err != nil {
		return nil, err
	}
	var o *model.Observation
	for i := range rows {
		if rows[i].ID == id {
			o = &rows[i]
			break
		}
	}
	if o == nil {
		return nil, ErrObservationMissing
	}
	if len(notes) > 8000 {
		return nil, ErrObservation
	}
	if len(o.Results) > 0 && !refresh {
		return &o.Results[0], nil
	}
	result := &model.ObservationResult{ProjectID: o.ProjectID, ObservationID: o.ID, Conclusion: "pending", Reason: "window", CreatedAt: s.Now().Unix(), Notes: notes}
	end, _ := time.Parse("2006-01-02", o.FollowupThrough)
	// Wait until the entire provider date has passed (Pacific for GSC), plus
	// its usual finalization delay. Coverage must independently be complete.
	matureAt := end.Add(24 * time.Hour)
	if o.Metric == "clicks" || o.Metric == "impressions" {
		matureAt = end.Add(80 * time.Hour)
	}
	if o.Metric != "technical" && s.Now().Before(matureAt) {
		return result, nil
	}
	if o.Metric == "visibility" {
		rev, e := s.promptRevision(ctx, o.ProjectID, o.QIDs)
		if e != nil && !errors.Is(e, ErrObservation) {
			return nil, e
		}
		if e != nil || rev != o.PromptRevision {
			result.Conclusion = "incomparable"
			result.Reason = "prompt_changed"
		}
	}
	result.Followup, err = s.measure(ctx, slug, o, false, false)
	if err != nil {
		return nil, err
	}
	if len(o.ControlURLs) > 0 {
		result.ControlFollowup, err = s.measure(ctx, slug, o, false, true)
		if err != nil {
			return nil, err
		}
	}
	if result.Conclusion == "pending" {
		result.Conclusion, result.Reason = conclude(*o, result.Followup)
	}
	all, err := s.observations.List(ctx, o.ProjectID, "")
	if err != nil {
		return nil, err
	}
	for _, other := range all {
		if other.TaskID == o.TaskID || other.ReleasedAt < func() int64 { a, _ := time.Parse("2006-01-02", o.BaselineFrom); return a.Unix() }() || other.ReleasedAt > end.Add(24*time.Hour).Unix() {
			continue
		}
		overlap := false
		for _, u := range append(append([]string{}, o.URLs...), o.ControlURLs...) {
			for _, v := range other.URLs {
				if u == v {
					overlap = true
				}
			}
		}
		for _, q := range o.QIDs {
			for _, v := range other.QIDs {
				if q == v {
					overlap = true
				}
			}
		}
		if overlap {
			result.Conclusion = "incomparable"
			result.Reason = "overlapping_changes"
			break
		}
	}
	if err := s.observations.Append(ctx, result); err != nil {
		return nil, err
	}
	return result, nil
}
func conclude(o model.Observation, after model.Measurement) (string, string) {
	if o.Metric == "technical" {
		if !after.Valid || after.AuditAt <= o.ReleasedAt {
			return "pending", "fresh_audit"
		}
		if after.Value == 0 {
			return "improved", "technical_clear"
		}
		return "insufficient", "technical_remaining"
	}
	if !o.Baseline.Valid {
		return "insufficient", "baseline"
	}
	if !after.Valid {
		if after.Reason == "property_changed" || after.Reason == "sample_changed" || after.Reason == "sample_metadata" {
			return "incomparable", after.Reason
		}
		if after.Reason == "coverage" {
			return "pending", after.Reason
		}
		return "insufficient", after.Reason
	}
	if o.Baseline.Signature != after.Signature {
		return "incomparable", "scope_changed"
	}
	if o.Metric == "visibility" {
		if o.Baseline.Count < 30 || after.Count < 30 {
			return "insufficient", "small_sample"
		}
		for q, a := range o.Baseline.Strata {
			b := after.Strata[q]
			if a[1]*after.Count != b[1]*o.Baseline.Count {
				return "incomparable", "sample_mix"
			}
		}
		switch metrics.Change(int(o.Baseline.Value), o.Baseline.Count, int(after.Value), after.Count) {
		case metrics.ChangeUp:
			return "improved", "observed_change"
		case metrics.ChangeDown:
			return "declined", "observed_change"
		}
		return "insufficient", "within_noise"
	}
	before := o.Baseline.Value
	if o.Baseline.Exposure < 1000 || after.Exposure < 1000 || before == 0 {
		return "insufficient", "small_sample"
	}
	delta := after.Value - before
	absolute := 50.0
	if o.Metric == "impressions" {
		absolute = 500
	}
	if math.Abs(delta) < absolute || math.Abs(delta)/before < .25 {
		return "insufficient", "below_threshold"
	}
	if delta > 0 {
		return "improved", "observed_change"
	}
	return "declined", "observed_change"
}
