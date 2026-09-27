// SPDX-License-Identifier: AGPL-3.0-or-later

package audit

import (
	"net/url"
	"strings"

	"github.com/craftsail/craftsail-growth/internal/service/crawl"
)

const (
	SevCritical = "critical"
	SevWarning  = "warning"
	SevInfo     = "info"

	LayerAccess     = "access"
	LayerDiscover   = "discover"
	LayerUnderstand = "understand"
	LayerCite       = "cite"

	SurfaceSEO  = "seo"
	SurfaceGEO  = "geo"
	SurfaceBoth = "both"

	ScopeSite    = "site"
	ScopePage    = "page"
	ScopeContent = "content" // only pages IsContentPage reports as content

	EvStandard      = "standard"
	EvVendor        = "vendor"
	EvExperiment    = "experiment"
	EvObservational = "observational"
	EvHeuristic     = "heuristic"
)

// LayerOrder is the fix order: a failure upstream hides everything downstream.
var LayerOrder = []string{LayerAccess, LayerDiscover, LayerUnderstand, LayerCite}

// Issue is one registered audit rule. Every code the audit emits must be
// here.
type Issue struct {
	Code     string   `json:"code"`
	Severity string   `json:"severity"`
	Layer    string   `json:"layer"`
	Surface  string   `json:"surface"`
	Scope    string   `json:"scope"`
	Evidence string   `json:"evidence"`
	Refs     []string `json:"refs"`
	Title    string   `json:"title"`
	Why      string   `json:"why"`
	Fix      string   `json:"fix"`
}

func Lookup(code string) Issue {
	if is, ok := Registry[code]; ok {
		return is
	}
	return Issue{Code: code, Severity: SevInfo, Layer: LayerCite, Surface: SurfaceBoth, Scope: ScopePage,
		Evidence: EvHeuristic, Title: code, Why: "Unregistered code.", Fix: "-"}
}

var Registry = map[string]Issue{}

func register(items ...Issue) {
	for _, is := range items {
		Registry[is.Code] = is
	}
}

func init() {
	// ---- access ----
	register(
		Issue{Code: "ROBOTS_BLOCKS_AI", Severity: SevCritical, Layer: LayerAccess, Surface: SurfaceBoth, Scope: ScopeSite, Evidence: EvStandard, Refs: []string{"rfc9309"},
			Title: "robots.txt blocks AI crawlers",
			Why:   "Under RFC 9309 a matching Disallow group stops that crawler from fetching the listed paths.",
			Fix:   "Add an explicit User-agent group that allows GPTBot, OAI-SearchBot, ClaudeBot, PerplexityBot, Google-Extended and the regional crawlers you target."},
		Issue{Code: "AI_UA_BLOCKED", Severity: SevCritical, Layer: LayerAccess, Surface: SurfaceGEO, Scope: ScopeSite, Evidence: EvVendor, Refs: []string{"g-http"},
			Title: "WAF or CDN rejects AI crawler user agents",
			Why:   "The home page loads for a browser but returns 403/406 for real AI crawler user agents, even though robots.txt allows them.",
			Fix:   "Allowlist these user agents in the bot protection rules, then crawl again."},
		Issue{Code: "PAGE_UNREACHABLE", Severity: SevCritical, Layer: LayerAccess, Surface: SurfaceBoth, Scope: ScopePage, Evidence: EvVendor, Refs: []string{"g-http"},
			Title: "Page could not be fetched",
			Why:   "Network errors and timeouts mean no crawler receives the content.",
			Fix:   "Check DNS, TLS and server logs for this URL."},
		Issue{Code: "SERVER_ERROR", Severity: SevCritical, Layer: LayerAccess, Surface: SurfaceBoth, Scope: ScopePage, Evidence: EvVendor, Refs: []string{"g-http"},
			Title: "Server error (5xx)",
			Why:   "Google documents that repeated 5xx responses slow crawling and can drop URLs from the index.",
			Fix:   "Fix the server error, or return 404/410 if the page is gone."},
		Issue{Code: "CLIENT_ERROR", Severity: SevWarning, Layer: LayerAccess, Surface: SurfaceBoth, Scope: ScopePage, Evidence: EvVendor, Refs: []string{"g-http"},
			Title: "Page returns 4xx",
			Why:   "4xx URLs are not indexed; if they are linked or listed in the sitemap they waste crawl requests.",
			Fix:   "Restore the page, or remove it from links and the sitemap and redirect to the closest live page."},
		Issue{Code: "NON_200_STATUS", Severity: SevWarning, Layer: LayerAccess, Surface: SurfaceBoth, Scope: ScopePage, Evidence: EvVendor, Refs: []string{"g-http"},
			Title: "Page returns a non-200 success or redirect code",
			Why:   "202 and 3xx responses are handled differently from 200 by crawlers.",
			Fix:   "Serve the canonical URL directly with 200."},
		Issue{Code: "NOINDEX", Severity: SevCritical, Layer: LayerAccess, Surface: SurfaceBoth, Scope: ScopePage, Evidence: EvVendor, Refs: []string{"g-robots-meta"},
			Title: "Page has meta robots noindex",
			Why:   "noindex tells search engines not to show the page, which also keeps it out of AI features built on the index.",
			Fix:   "Remove noindex from pages that should be found."},
		Issue{Code: "XROBOTS_NOINDEX", Severity: SevCritical, Layer: LayerAccess, Surface: SurfaceBoth, Scope: ScopePage, Evidence: EvVendor, Refs: []string{"g-robots-meta"},
			Title: "X-Robots-Tag header has noindex",
			Why:   "The header has the same effect as the meta tag but is invisible in page source; it is often injected by a CDN.",
			Fix:   "Remove the header rule at the server or CDN."},
		Issue{Code: "SPA_SHELL", Severity: SevCritical, Layer: LayerAccess, Surface: SurfaceBoth, Scope: ScopePage, Evidence: EvVendor, Refs: []string{"g-js"},
			Title: "Static HTML has no content",
			Why:   "Google renders JavaScript in a deferred queue, and many AI crawlers read static HTML only, so a client-only shell looks empty.",
			Fix:   "Use server-side rendering or prerendering for content pages."},
		Issue{Code: "SLOW_RESPONSE", Severity: SevInfo, Layer: LayerAccess, Surface: SurfaceSEO, Scope: ScopePage, Evidence: EvHeuristic,
			Title: "Slow server response",
			Why:   "The HTML took more than 3 seconds; crawlers fetch fewer pages from slow hosts. The 3 s threshold is this project's rule.",
			Fix:   "Cache HTML at the edge or profile the slow route."},
		Issue{Code: "LOW_CONTENT_PAGE", Severity: SevInfo, Layer: LayerCite, Surface: SurfaceBoth, Scope: ScopePage, Evidence: EvHeuristic,
			Title: "Functional page with little text",
			Why:   "Login, cart and contact pages are expected to be short; this is listed only so they are not mistaken for empty shells.",
			Fix:   "No action needed unless the page should rank."},
	)
	// ---- discover ----
	register(
		Issue{Code: "NO_SITEMAP", Severity: SevWarning, Layer: LayerDiscover, Surface: SurfaceBoth, Scope: ScopeSite, Evidence: EvVendor, Refs: []string{"g-sitemaps"},
			Title: "No sitemap.xml",
			Why:   "A sitemap tells crawlers which URLs matter; large or poorly linked sites benefit most.",
			Fix:   "Publish /sitemap.xml listing canonical URLs only."},
		Issue{Code: "SITEMAP_NOT_DECLARED", Severity: SevInfo, Layer: LayerDiscover, Surface: SurfaceBoth, Scope: ScopeSite, Evidence: EvStandard, Refs: []string{"g-sitemaps"},
			Title: "robots.txt does not declare the sitemap",
			Why:   "A Sitemap: line lets any crawler find the sitemap without being told.",
			Fix:   "Add `Sitemap: https://example.com/sitemap.xml` to robots.txt."},
		Issue{Code: "SITEMAP_LOW_VALUE_URLS", Severity: SevInfo, Layer: LayerDiscover, Surface: SurfaceBoth, Scope: ScopeSite, Evidence: EvVendor, Refs: []string{"g-sitemaps"},
			Title: "Sitemap lists parameter, search or pagination URLs",
			Why:   "Sitemaps should list the canonical URLs you want indexed.",
			Fix:   "Remove those URLs from the sitemap."},
		Issue{Code: "NO_LLMS_TXT", Severity: SevInfo, Layer: LayerDiscover, Surface: SurfaceGEO, Scope: ScopeSite, Evidence: EvHeuristic, Refs: []string{"llmstxt"},
			Title: "No /llms.txt",
			Why:   "llms.txt is a 2024 proposal, not a standard, and no engine documents that it reads it. Low cost, uncertain benefit.",
			Fix:   "Optional: publish /llms.txt generated from the brand facts."},
		Issue{Code: "LLMS_TXT_BROKEN_LINKS", Severity: SevWarning, Layer: LayerDiscover, Surface: SurfaceGEO, Scope: ScopeSite, Evidence: EvHeuristic, Refs: []string{"llmstxt"},
			Title: "llms.txt links to broken or blocked pages",
			Why:   "A map that points to 404s or disallowed paths is worse than none.",
			Fix:   "Fix or remove the broken entries."},
		Issue{Code: "NO_CANONICAL", Severity: SevInfo, Layer: LayerDiscover, Surface: SurfaceBoth, Scope: ScopePage, Evidence: EvVendor, Refs: []string{"g-canonical"},
			Title: "No canonical link",
			Why:   "Without it, search engines pick a canonical themselves when duplicates exist.",
			Fix:   "Add a self-referencing rel=canonical."},
		Issue{Code: "CANONICAL_MISMATCH", Severity: SevWarning, Layer: LayerDiscover, Surface: SurfaceBoth, Scope: ScopePage, Evidence: EvVendor, Refs: []string{"g-canonical"},
			Title: "Canonical points to another URL",
			Why:   "Signals are consolidated to the target, so this page will not be the one shown.",
			Fix:   "Confirm the consolidation is intended; otherwise point canonical to this URL."},
		Issue{Code: "DUPLICATE_TITLE", Severity: SevWarning, Layer: LayerDiscover, Surface: SurfaceBoth, Scope: ScopeSite, Evidence: EvVendor, Refs: []string{"g-title", "g-canonical"},
			Title: "Pages share the same title",
			Why:   "Identical titles make pages hard to tell apart and often indicate duplicate URLs.",
			Fix:   "Write a distinct title per page, or consolidate duplicates with canonical or 301."},
		Issue{Code: "DUPLICATE_BODY", Severity: SevWarning, Layer: LayerDiscover, Surface: SurfaceBoth, Scope: ScopeSite, Evidence: EvVendor, Refs: []string{"g-canonical"},
			Title: "Pages share near-identical body text",
			Why:   "Search engines index one version of duplicates; the one you want may lose.",
			Fix:   "Keep one canonical version and 301 or canonical the rest."},
		Issue{Code: "DUPLICATE_META_DESCRIPTION", Severity: SevInfo, Layer: LayerDiscover, Surface: SurfaceSEO, Scope: ScopeSite, Evidence: EvVendor, Refs: []string{"g-snippet"},
			Title: "Pages share the same meta description",
			Why:   "Google recommends a unique description per page.",
			Fix:   "Write a page-specific description or remove the duplicate."},
		Issue{Code: "HREFLANG_LOW", Severity: SevWarning, Layer: LayerDiscover, Surface: SurfaceBoth, Scope: ScopeSite, Evidence: EvVendor, Refs: []string{"g-hreflang"},
			Title: "Multilingual site with little hreflang",
			Why:   "hreflang tells engines which language version to serve; without it versions compete.",
			Fix:   "Add reciprocal hreflang links between language versions."},
		Issue{Code: "BROKEN_INTERNAL_LINK", Severity: SevWarning, Layer: LayerDiscover, Surface: SurfaceBoth, Scope: ScopePage, Evidence: EvVendor, Refs: []string{"g-links"},
			Title: "Links to a broken internal URL",
			Why:   "Crawlers follow links to discover pages; broken links waste requests and hide the intended target.",
			Fix:   "Point the link at the live URL or remove it."},
		Issue{Code: "ORPHAN_PAGE", Severity: SevInfo, Layer: LayerDiscover, Surface: SurfaceBoth, Scope: ScopePage, Evidence: EvVendor, Refs: []string{"g-links"},
			Title: "No crawled page links here",
			Why:   "Pages reachable only from the sitemap are harder to discover and get little internal context.",
			Fix:   "Link to it from a relevant page."},
		Issue{Code: "REDIRECT_CHAIN", Severity: SevWarning, Layer: LayerDiscover, Surface: SurfaceSEO, Scope: ScopePage, Evidence: EvVendor, Refs: []string{"g-redirects"},
			Title: "Two or more redirects before the page",
			Why:   "Each hop adds latency, and crawlers stop following long chains.",
			Fix:   "Redirect straight to the final URL and update internal links."},
	)
	// ---- understand ----
	register(
		Issue{Code: "MISSING_TITLE", Severity: SevCritical, Layer: LayerUnderstand, Surface: SurfaceBoth, Scope: ScopePage, Evidence: EvVendor, Refs: []string{"g-title"},
			Title: "Missing <title>",
			Why:   "Google uses the title element as a main source for title links.",
			Fix:   "Add a unique, descriptive title."},
		Issue{Code: "TITLE_TOO_LONG", Severity: SevInfo, Layer: LayerUnderstand, Surface: SurfaceSEO, Scope: ScopePage, Evidence: EvHeuristic, Refs: []string{"g-title"},
			Title: "Title is long",
			Why:   "Google has no length limit but truncates by display width. Threshold here: about 60 Latin or 32 CJK characters.",
			Fix:   "Put the key phrase first."},
		Issue{Code: "MISSING_META_DESCRIPTION", Severity: SevInfo, Layer: LayerUnderstand, Surface: SurfaceSEO, Scope: ScopePage, Evidence: EvVendor, Refs: []string{"g-snippet"},
			Title: "Missing meta description",
			Why:   "Google may use the description for the snippet; without it the snippet is taken from page text.",
			Fix:   "Add a one or two sentence summary."},
		Issue{Code: "MISSING_H1", Severity: SevWarning, Layer: LayerUnderstand, Surface: SurfaceBoth, Scope: ScopeContent, Evidence: EvHeuristic, Refs: []string{"g-title"},
			Title: "Content page has no H1",
			Why:   "The main heading is one of the signals Google uses for title links, and it states the topic for readers.",
			Fix:   "Add one H1 that states the page topic."},
		Issue{Code: "MULTIPLE_H1", Severity: SevInfo, Layer: LayerUnderstand, Surface: SurfaceSEO, Scope: ScopeContent, Evidence: EvHeuristic,
			Title: "More than one H1",
			Why:   "Usually a template mistake such as a logo marked as H1. Google does not penalize it; listed for clarity.",
			Fix:   "Keep one H1 for the main heading."},
		Issue{Code: "IMAGES_MISSING_ALT", Severity: SevInfo, Layer: LayerUnderstand, Surface: SurfaceSEO, Scope: ScopePage, Evidence: EvVendor, Refs: []string{"g-images"},
			Title: "Images without alt text",
			Why:   "Google uses alt text to understand images, and accessibility requires it.",
			Fix:   "Describe informative images in alt; use alt=\"\" for decorative ones."},
		Issue{Code: "NO_JSONLD", Severity: SevWarning, Layer: LayerUnderstand, Surface: SurfaceBoth, Scope: ScopePage, Evidence: EvVendor, Refs: []string{"g-structured", "g-ai-features"},
			Title: "No structured data",
			Why:   "Structured data helps engines classify the page. Google states it is not required for AI features.",
			Fix:   "Add Organization on the home page and Article or Product where they fit."},
		Issue{Code: "SCHEMA_CONTENT_MISMATCH", Severity: SevWarning, Layer: LayerUnderstand, Surface: SurfaceBoth, Scope: ScopePage, Evidence: EvVendor, Refs: []string{"g-structured"},
			Title: "Structured data does not match visible content",
			Why:   "Google's guidelines require markup to describe content that users can see.",
			Fix:   "Remove the markup or add the matching visible content."},
		Issue{Code: "NO_AUTHOR_ENTITY", Severity: SevInfo, Layer: LayerUnderstand, Surface: SurfaceBoth, Scope: ScopeContent, Evidence: EvVendor, Refs: []string{"g-helpful"},
			Title: "Article markup has no author",
			Why:   "Google's helpful-content guidance asks who created the content; article markup can state it.",
			Fix:   "Add author to Article markup and show a byline."},
		Issue{Code: "LANG_IMBALANCE", Severity: SevInfo, Layer: LayerUnderstand, Surface: SurfaceGEO, Scope: ScopeSite, Evidence: EvHeuristic,
			Title: "Languages are very uneven",
			Why:   "The site publishes in Chinese and English, but one language has far fewer content pages than the other.",
			Fix:   "Close the gap on the thinner side first, starting with pages tied to your prompts."},
	)
	// ---- cite (content pages only unless noted) ----
	register(
		Issue{Code: "SHORT_CONTENT", Severity: SevInfo, Layer: LayerCite, Surface: SurfaceGEO, Scope: ScopeContent, Evidence: EvObservational, Refs: []string{"zhang2026", "geo-citation-lab"},
			Title: "Content page is short",
			Why:   "In the citation dataset top-quartile pages averaged 1,943 words and bottom-quartile 170. This is a correlation, not a target.",
			Fix:   "Add substance the reader needs (numbers, comparisons, steps), not filler."},
		Issue{Code: "FEW_H2", Severity: SevInfo, Layer: LayerCite, Surface: SurfaceGEO, Scope: ScopeContent, Evidence: EvObservational, Refs: []string{"geo-citation-lab"},
			Title: "Few sections",
			Why:   "Top-quartile cited pages averaged 10.6 headings. Correlation only.",
			Fix:   "Split long content into sections that each answer one question."},
		Issue{Code: "LOW_LIST_DENSITY", Severity: SevInfo, Layer: LayerCite, Surface: SurfaceGEO, Scope: ScopeContent, Evidence: EvObservational, Refs: []string{"geo-citation-lab"},
			Title: "Almost no lists",
			Why:   "Top-quartile cited pages had list density 0.43 versus 0.05 in the bottom quartile. Correlation only.",
			Fix:   "Turn enumerations into ul/ol lists."},
		Issue{Code: "NO_DEFINITION", Severity: SevWarning, Layer: LayerCite, Surface: SurfaceGEO, Scope: ScopeContent, Evidence: EvObservational, Refs: []string{"zhang2026"},
			Title: "No definition sentence",
			Why:   "Pages with a definition had 57.3% higher mean influence in the absorption dataset.",
			Fix:   "Open with one sentence that says what the subject is."},
		Issue{Code: "NO_NUMBERS", Severity: SevWarning, Layer: LayerCite, Surface: SurfaceGEO, Scope: ScopeContent, Evidence: EvExperiment, Refs: []string{"aggarwal2024", "zhang2026"},
			Title: "Few concrete numbers",
			Why:   "Adding statistics was among the strongest rewrites in the GEO-bench experiment; pages with numbers had 61.6% higher influence in the absorption dataset.",
			Fix:   "Add sourced numbers with units and dates."},
		Issue{Code: "NO_COMPARISON", Severity: SevInfo, Layer: LayerCite, Surface: SurfaceGEO, Scope: ScopeContent, Evidence: EvObservational, Refs: []string{"zhang2026"},
			Title: "No comparison",
			Why:   "Comparison pages had 55.3% higher mean influence in the absorption dataset.",
			Fix:   "Add a table comparing options on the same criteria."},
		Issue{Code: "NO_HOWTO", Severity: SevInfo, Layer: LayerCite, Surface: SurfaceGEO, Scope: ScopeContent, Evidence: EvObservational, Refs: []string{"zhang2026"},
			Title: "No steps",
			Why:   "How-to content had 41.2% higher mean influence in the absorption dataset.",
			Fix:   "Add numbered steps where the reader has to do something."},
		Issue{Code: "NO_QUOTABLE_PASSAGE", Severity: SevWarning, Layer: LayerCite, Surface: SurfaceGEO, Scope: ScopeContent, Evidence: EvHeuristic,
			Title: "No self-contained passage",
			Why:   "Retrieval picks passages, not pages. Rule: a section of at least 60 words with a number, definition or step. The rule is this project's.",
			Fix:   "Rewrite two or three key sections so the first sentence answers the heading."},
		Issue{Code: "NO_DATE", Severity: SevInfo, Layer: LayerCite, Surface: SurfaceBoth, Scope: ScopeContent, Evidence: EvObservational, Refs: []string{"zhen2026", "g-helpful"},
			Title: "No visible date",
			Why:   "Cited content for time-sensitive queries had a half-life of about 39 days in the Chinese-engine study; a date lets readers and engines judge freshness.",
			Fix:   "Show published and updated dates, and add them to Article markup."},
		Issue{Code: "FEW_EXTERNAL_LINKS", Severity: SevInfo, Layer: LayerCite, Surface: SurfaceGEO, Scope: ScopeContent, Evidence: EvExperiment, Refs: []string{"aggarwal2024"},
			Title: "Cites few sources",
			Why:   "Citing sources was among the strongest rewrites in the GEO-bench experiment.",
			Fix:   "Link the sources behind your claims."},
		Issue{Code: "LOW_RELEVANCE", Severity: SevWarning, Layer: LayerCite, Surface: SurfaceBoth, Scope: ScopeContent, Evidence: EvObservational, Refs: []string{"zhang2026"},
			Title: "Headings miss the target prompt wording",
			Why:   "Relevance to the query was the strongest single predictor of influence (r = 0.432).",
			Fix:   "Use the words buyers use in the title, H1 and H2."},
	)
}

// IsContentPage reports whether content-shape rules apply. Home, functional
// and near-empty pages are not content pages.
func IsContentPage(page crawl.PageDoc) bool {
	u, err := url.Parse(page.URL)
	if err != nil {
		return false
	}
	if strings.TrimRight(u.Path, "/") == "" || funcPage.MatchString(u.Path) {
		return false
	}
	for _, t := range page.JSONLDTypes {
		switch t {
		case "Article", "TechArticle", "NewsArticle", "BlogPosting", "HowTo", "FAQPage":
			return true
		}
	}
	return page.WordCount >= 300
}
