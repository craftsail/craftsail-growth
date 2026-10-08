// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"context"
	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/plan"
	"testing"
)

func TestReleaseObservationPermissionsAndReadOnly(t *testing.T) {
	w := newWorld(t)
	w.h.plan = plan.New(w.db)
	p, _ := w.h.projects.Get(context.Background(), "alpha")
	task := model.Task{ProjectID: p.ID, Code: "S-001", Status: model.TaskDone, Acceptance: map[string]any{"type": "manual"}}
	if err := w.db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	r := testEngine(w.h)
	base := "/api/projects/alpha/tasks/S-001"
	body := `{"hypothesis":"Improve answers","metric":"clicks","urls":["https://a.com/a"],"released_at":1700000000}`
	if res := call(r, "POST", base+"/releases", body, w.viewer); res.Code != 403 {
		t.Fatal(res.Code)
	}
	if res := call(r, "POST", base+"/releases", body, w.outsider); res.Code != 404 {
		t.Fatal(res.Code)
	}
	if res := call(r, "POST", base+"/releases", body, w.editor); res.Code != 200 {
		t.Fatal(res.Code, res.Body.String())
	}
	for i := 0; i < 2; i++ {
		if res := call(r, "GET", base+"/observations", "", w.viewer); res.Code != 200 {
			t.Fatal(res.Code)
		}
	}
	var count int64
	w.db.Model(&model.ObservationResult{}).Count(&count)
	if count != 0 {
		t.Fatal("GET evaluated observations")
	}
	if res := call(r, "POST", base+"/observations/1/evaluate", `{"refresh":true}`, w.viewer); res.Code != 403 {
		t.Fatal(res.Code)
	}
	if res := call(r, "POST", base+"/observations/1/evaluate", `{"refresh":true}`, w.editor); res.Code != 200 {
		t.Fatal(res.Code, res.Body.String())
	}
	if res := call(r, "GET", base+"/observations", "", w.outsider); res.Code != 404 {
		t.Fatal(res.Code)
	}
}
