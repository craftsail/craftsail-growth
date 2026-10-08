// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type trafficQuotaKey struct{}
type trafficQuota struct {
	service    *Service
	properties map[string]string
}

// Context ownership keeps concurrent projects from sharing mutable Client hooks.
// These conservative request caps are local guards, not GA's token quotas.
func (s *Service) trafficQuotaContext(ctx context.Context, gsc, ga string) context.Context {
	gsc, _ = GSCPropertyKey(gsc)
	ga, _ = GAPropertyKey(ga)
	return context.WithValue(ctx, trafficQuotaKey{}, trafficQuota{s, map[string]string{"gsc": gsc, "ga": ga}})
}
func reserveTraffic(ctx context.Context, api string) error {
	q, ok := ctx.Value(trafficQuotaKey{}).(trafficQuota)
	if !ok || q.properties[api] == "" {
		return nil
	}
	now := q.service.now()
	loc, _ := time.LoadLocation("America/Los_Angeles")
	y, m, d := now.In(loc).Date()
	day := time.Date(y, m, d, 0, 0, 0, 0, loc)
	retry, err := q.service.rows.ReserveGoogleRequest(ctx, q.properties[api], "traffic/"+api, now.Unix(), day.Unix(), day.AddDate(0, 0, 1).Unix(), 20000, 120)
	if err != nil {
		return err
	}
	if retry > now.Unix() {
		return fmt.Errorf("%s HTTP 429 shared traffic quota; retry after %s", api, time.Unix(retry, 0).UTC().Format(time.RFC3339))
	}
	return nil
}
func backoffTraffic(ctx context.Context, api, retryAfter string) {
	q, ok := ctx.Value(trafficQuotaKey{}).(trafficQuota)
	if !ok || q.properties[api] == "" {
		return
	}
	now := q.service.now()
	until := now.Add(15 * time.Minute)
	if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds > 0 {
		if t := now.Add(time.Duration(seconds) * time.Second); t.After(until) {
			until = t
		}
	} else if t, err := http.ParseTime(retryAfter); err == nil && t.After(until) {
		until = t
	}
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	_ = q.service.rows.BackoffGoogle(saveCtx, q.properties[api], "traffic/"+api, until.Unix())
}
