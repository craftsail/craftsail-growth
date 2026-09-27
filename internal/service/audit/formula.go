// SPDX-License-Identifier: AGPL-3.0-or-later

package audit

// FormulaMarkdown is the content readiness score definition, matching
// ScorePage in score.go. The thresholds come from correlations in the
// citation dataset (evidence level: observational); they describe pages that
// were cited, not targets that guarantee citation.
const FormulaMarkdown = `## Content readiness score

Score = crawlability + length + structure + extractable blocks + authority + relevance, out of 100.
Grades: >=80 A, >=65 B, >=45 C, otherwise D. The site average is the mean over reachable pages.
Words are counted in static HTML only; JavaScript is not executed.
Length, section and list rules only raise issues on content pages (not the home page or login, cart and contact pages).

### Crawlability (15)

- HTTP 200: +7; other 2xx/3xx: +3; otherwise 0
- No meta robots or X-Robots-Tag noindex: +3
- Has canonical: +2
- At least 120 words of static text: +3. Fewer words only lose these 3 points. An empty shell is judged separately: almost no text, only "loading" / "enable JavaScript", or an empty #root / #app.

### Length (15)

Factor by word count, times 15:

| Words | Factor |
|---:|---:|
| >=1500 | 1.00 |
| >=1000 | 0.85 |
| >=600 | 0.60 |
| >=300 | 0.35 |
| >=120 | 0.15 |
| fewer | 0 |

### Structure (20)

- Exactly one H1: +4; more than one: +2; none: 0
- H2 count factor x6: >=8 -> 1.00; >=6 -> 0.85; >=4 -> 0.60; >=2 -> 0.30
- Paragraph count factor x5: >=40 -> 1.00; >=25 -> 0.80; >=15 -> 0.55; >=8 -> 0.30
- List density li/(li+p) factor x5: >=0.35 -> 1.00; >=0.20 -> 0.75; >=0.10 -> 0.45; >=0.03 -> 0.20

### Extractable blocks (25)

Points when present: definition +7; numbers (at least 3 numbers with units) +7; comparison (comparison words or a table) +6; steps +5.
FAQ is detected but not scored: Q&A formatting showed 5.74% lower mean influence than non-Q&A pages (zhang2026).

### Authority (15)

- Visible date, or datePublished / dateModified in JSON-LD: +4
- Visible author: +2
- External link count factor x4: >=6 -> 1.00; >=3 -> 0.70; >=1 -> 0.40
- Authoritative JSON-LD type count factor x5: >=3 -> 1.00; >=2 -> 0.75; >=1 -> 0.45 (Organization, Product, FAQPage, Article, HowTo and similar)

### Relevance (10)

Share of prompt-library keywords covered by the title, H1 and H2, factor x10: >=0.40 -> 1.00; >=0.25 -> 0.80; >=0.12 -> 0.55; >=0.04 -> 0.30.
`
