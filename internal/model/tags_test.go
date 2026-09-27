// SPDX-License-Identifier: AGPL-3.0-or-later

package model

import "testing"

func TestSystemTagUsesBrandAliasAndDomain(t *testing.T) {
	aliases := []string{"Acme CLI", "示例命令行"}
	if got := ComputeSystemTags("What are the best tools?", "acmecli", aliases, "https://acmecli.example"); got[0] != TagUnbranded {
		t.Fatalf("got %v", got)
	}
	if got := ComputeSystemTags("acmecli 怎么样", "acmecli", aliases, "https://acmecli.example"); got[0] != TagBranded {
		t.Fatalf("name got %v", got)
	}
	if got := ComputeSystemTags("示例命令行 靠谱吗", "acmecli", aliases, "https://acmecli.example"); got[0] != TagBranded {
		t.Fatalf("alias got %v", got)
	}
	if got := ComputeSystemTags("see acmecli.example docs", "Other", nil, "https://www.acmecli.example/path"); got[0] != TagBranded {
		t.Fatalf("domain got %v", got)
	}
}

func TestUserTagOverridesSystemBrand(t *testing.T) {
	if !EffectiveBranded([]string{TagUnbranded}, []string{"branded", "price"}) {
		t.Fatal("user branded should override")
	}
	if EffectiveBranded([]string{TagBranded}, []string{"unbranded"}) {
		t.Fatal("user unbranded should override")
	}
	if !EffectiveBranded([]string{TagBranded}, []string{"branded", "unbranded"}) {
		t.Fatal("both user tags should fall back to system")
	}
}

func TestMatchTagsIsOr(t *testing.T) {
	system := []string{TagUnbranded}
	user := []string{"price"}
	if !MatchTags(system, user, nil) {
		t.Fatal("empty filter includes the prompt")
	}
	if MatchTags(system, user, []string{TagBranded}) {
		t.Fatal("branded filter should skip an unbranded prompt")
	}
	if !MatchTags(system, user, []string{"price"}) {
		t.Fatal("user tag should match")
	}
	if !MatchTags(system, user, []string{TagBranded, "price"}) {
		t.Fatal("any selected tag should match")
	}
}
