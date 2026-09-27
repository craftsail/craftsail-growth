// SPDX-License-Identifier: AGPL-3.0-or-later

package model

import "testing"

func TestMentionTargetFallsBackToDefault(t *testing.T) {
	if got := (Targets{}).MentionTarget(); got != DefaultTargets().MentionRate {
		t.Fatalf("got %v", got)
	}
	if got := (Targets{MentionRate: 0.2}).MentionTarget(); got != 0.2 {
		t.Fatalf("got %v", got)
	}
}
