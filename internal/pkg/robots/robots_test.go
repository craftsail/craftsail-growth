// SPDX-License-Identifier: AGPL-3.0-or-later

package robots

import "testing"

func TestSharedUserAgentGroup(t *testing.T) {
	txt := "User-agent: GPTBot\nUser-agent: ClaudeBot\nDisallow: /secret\n"
	g := Parse(txt)
	ok, _ := Decision(g, "GPTBot", "/secret")
	if ok {
		t.Fatal("GPTBot should share Disallow with ClaudeBot")
	}
	ok, _ = Decision(g, "ClaudeBot", "/secret")
	if ok {
		t.Fatal("ClaudeBot should be blocked on /secret")
	}
}

func TestSpecificOverridesWildcard(t *testing.T) {
	txt := "User-agent: *\nDisallow: /\n\nUser-agent: GPTBot\nAllow: /\n"
	g := Parse(txt)
	ok, _ := Decision(g, "GPTBot", "/")
	if !ok {
		t.Fatal("GPTBot specific Allow should beat wildcard Disallow")
	}
	ok, _ = Decision(g, "PerplexityBot", "/")
	if ok {
		t.Fatal("PerplexityBot should be blocked by wildcard")
	}
}

func TestLongestMatchAllowWinsTie(t *testing.T) {
	txt := "User-agent: *\nDisallow: /app\nAllow: /app\n"
	g := Parse(txt)
	ok, rule := Decision(g, "GPTBot", "/app")
	if !ok {
		t.Fatalf("same-length Allow should win, rule=%s", rule)
	}
}

func TestEmptyDisallowIsNotARule(t *testing.T) {
	txt := "User-agent: *\nDisallow:\n"
	g := Parse(txt)
	ok, _ := Decision(g, "GPTBot", "/")
	if !ok {
		t.Fatal("empty Disallow means allow all")
	}
}

func TestWildcardPathAndEndAnchor(t *testing.T) {
	txt := "User-agent: *\nDisallow: /*.pdf$\nAllow: /docs/\n"
	g := Parse(txt)
	ok, _ := Decision(g, "GPTBot", "/file.pdf")
	if ok {
		t.Fatal("*.pdf$ should block")
	}
	ok, _ = Decision(g, "GPTBot", "/docs/a")
	if !ok {
		t.Fatal("/docs/ should be allowed")
	}
}
