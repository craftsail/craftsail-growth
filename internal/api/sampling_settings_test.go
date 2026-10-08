// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/bootstrap"
	"github.com/craftsail/craftsail-growth/internal/service/sample"
	"testing"
)

func TestSamplingPreviewAndLibraryPermissions(t *testing.T) {
	w := newWorld(t)
	w.h.sample = sample.New(w.db, nil)
	w.h.bootstrap = bootstrap.New(w.db)
	r := testEngine(w.h)
	for _, path := range []string{"/api/projects/alpha/sample-preview?repeat=2", "/api/projects/alpha/question-libraries"} {
		if res := call(r, "GET", path, "", w.viewer); res.Code != 200 {
			t.Fatal(res.Code, res.Body.String())
		}
		if res := call(r, "GET", path, "", w.outsider); res.Code != 404 {
			t.Fatal(res.Code)
		}
	}
	var count int64
	w.db.Model(&model.QuestionLibrary{}).Count(&count)
	if count != 0 {
		t.Fatal("read created a library")
	}
	w.db.Model(&model.SampleRun{}).Count(&count)
	if count != 0 {
		t.Fatal("preview sampled")
	}
	if res := call(r, "GET", "/api/projects/alpha/sample-preview?repeat=500", "", w.viewer); res.Code != 400 {
		t.Fatal(res.Code)
	}
	if res := call(r, "PATCH", "/api/projects/alpha", `{"sampling_language":"xx"}`, testToken); res.Code != 400 {
		t.Fatal(res.Code, res.Body.String())
	}
	if res := call(r, "PATCH", "/api/projects/alpha", `{"sampling_language":"pt","target_region":"BR","site_language":"en"}`, w.viewer); res.Code != 403 {
		t.Fatal(res.Code)
	}
}
