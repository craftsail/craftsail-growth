// SPDX-License-Identifier: AGPL-3.0-or-later

package audit

type Reference struct {
	Key  string `json:"key"`
	Cite string `json:"cite"`
	URL  string `json:"url"`
	Kind string `json:"kind"` // paper | preprint | standard | vendor | dataset | project
}

// References are cited by key from Registry.
// Link-check them before each release.
var References = map[string]Reference{
	"aggarwal2024":     {Key: "aggarwal2024", Kind: "paper", Cite: "Aggarwal et al. GEO: Generative Engine Optimization. KDD 2024.", URL: "https://arxiv.org/abs/2311.09735"},
	"chen2025":         {Key: "chen2025", Kind: "preprint", Cite: "Chen, Wang, Chen, Koudas. Generative Engine Optimization: How to Dominate AI Search. 2025.", URL: "https://arxiv.org/abs/2509.08919"},
	"zhang2026":        {Key: "zhang2026", Kind: "preprint", Cite: "Zhang, He, Yao. From Citation Selection to Citation Absorption. 2026.", URL: "https://arxiv.org/abs/2604.25707"},
	"zhen2026":         {Key: "zhen2026", Kind: "preprint", Cite: "Zhen, Liu, Zhang, Niu. What Do Chinese-Language Generative Search Engines Cite and Surface? 2026.", URL: "https://arxiv.org/abs/2607.15771"},
	"martinez2026":     {Key: "martinez2026", Kind: "preprint", Cite: "Martinez. Optimizing Visibility in Generative Engines: A Critical Survey (2023-2026). 2026.", URL: "https://arxiv.org/abs/2607.14035"},
	"geo-citation-lab": {Key: "geo-citation-lab", Kind: "dataset", Cite: "GEO Citation Lab dataset (overseas and CN-GEO).", URL: "https://github.com/yaojingang/geo-citation-lab"},
	"rfc9309":          {Key: "rfc9309", Kind: "standard", Cite: "RFC 9309: Robots Exclusion Protocol. IETF, 2022.", URL: "https://www.rfc-editor.org/rfc/rfc9309"},
	"llmstxt":          {Key: "llmstxt", Kind: "project", Cite: "Howard. The /llms.txt file. 2024.", URL: "https://llmstxt.org"},
	"g-ai-features":    {Key: "g-ai-features", Kind: "vendor", Cite: "Google Search Central. AI features and your website.", URL: "https://developers.google.com/search/docs/appearance/ai-features"},
	"g-helpful":        {Key: "g-helpful", Kind: "vendor", Cite: "Google Search Central. Creating helpful, reliable, people-first content.", URL: "https://developers.google.com/search/docs/fundamentals/creating-helpful-content"},
	"g-structured":     {Key: "g-structured", Kind: "vendor", Cite: "Google Search Central. Introduction to structured data markup.", URL: "https://developers.google.com/search/docs/appearance/structured-data/intro-structured-data"},
	"g-title":          {Key: "g-title", Kind: "vendor", Cite: "Google Search Central. Influencing title links in search results.", URL: "https://developers.google.com/search/docs/appearance/title-link"},
	"g-snippet":        {Key: "g-snippet", Kind: "vendor", Cite: "Google Search Central. Control your snippets in search results.", URL: "https://developers.google.com/search/docs/appearance/snippet"},
	"g-canonical":      {Key: "g-canonical", Kind: "vendor", Cite: "Google Search Central. Consolidate duplicate URLs.", URL: "https://developers.google.com/search/docs/crawling-indexing/consolidate-duplicate-urls"},
	"g-robots-meta":    {Key: "g-robots-meta", Kind: "vendor", Cite: "Google Search Central. Robots meta tag and X-Robots-Tag.", URL: "https://developers.google.com/search/docs/crawling-indexing/robots-meta-tag"},
	"g-sitemaps":       {Key: "g-sitemaps", Kind: "vendor", Cite: "Google Search Central. Learn about sitemaps.", URL: "https://developers.google.com/search/docs/crawling-indexing/sitemaps/overview"},
	"g-js":             {Key: "g-js", Kind: "vendor", Cite: "Google Search Central. JavaScript SEO basics.", URL: "https://developers.google.com/search/docs/crawling-indexing/javascript/javascript-seo-basics"},
	"g-images":         {Key: "g-images", Kind: "vendor", Cite: "Google Search Central. Google Images SEO best practices.", URL: "https://developers.google.com/search/docs/appearance/google-images"},
	"g-links":          {Key: "g-links", Kind: "vendor", Cite: "Google Search Central. Link best practices.", URL: "https://developers.google.com/search/docs/crawling-indexing/links-crawlable"},
	"g-redirects":      {Key: "g-redirects", Kind: "vendor", Cite: "Google Search Central. Redirects and Google Search.", URL: "https://developers.google.com/search/docs/crawling-indexing/301-redirects"},
	"g-hreflang":       {Key: "g-hreflang", Kind: "vendor", Cite: "Google Search Central. Tell Google about localized versions of your page.", URL: "https://developers.google.com/search/docs/specialty/international/localized-versions"},
	"g-http":           {Key: "g-http", Kind: "vendor", Cite: "Google Search Central. How HTTP status codes and network errors affect Search.", URL: "https://developers.google.com/search/docs/crawling-indexing/http-network-errors"},
}
