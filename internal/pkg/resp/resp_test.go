// SPDX-License-Identifier: AGPL-3.0-or-later

package resp

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestOKEnvelope(t *testing.T) {
	raw, err := json.Marshal(OK(map[string]int{"n": 1}))
	if err != nil {
		t.Fatal(err)
	}
	var got Envelope
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Code != 0 || got.Msg != "ok" {
		t.Fatalf("envelope = %+v", got)
	}
	data, ok := got.Data.(map[string]any)
	if !ok || data["n"].(float64) != 1 {
		t.Fatalf("data = %#v", got.Data)
	}
}

func TestFailEnvelope(t *testing.T) {
	env, status := Fail(4001, http.StatusBadRequest, "非法项目标识")
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d", status)
	}
	if env.Code != 4001 || env.Msg != "非法项目标识" || env.Data != nil {
		t.Fatalf("fail envelope = %+v", env)
	}
}
