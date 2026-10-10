# Weekly Report by Stage Implementation Plan (Phase 4)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The weekly report opens with the book's weekly table for the project's SEO stage: the new-site table from 《新站的 SEO 监测》§7 or the established table from 《SEO 的监测》§8, with this week, last week and a comparison column. It says which stage the project is in, using the same judgment as the help center, and lists what shipped this week.

**Architecture:**
- `webstats.WeeklySearch` (new, exported) computes per-week numbers for three complete Monday–Sunday weeks: this week, last week, and four weeks ago. It uses the unexported coverage helpers. A cell without full coverage is `nil` and renders "—", never 0.
- `report` gains a pure renderer `writeWeeklyTable` with two layouts.
- The stage comes from `playbook.Service.Status`. To allow `report → playbook`, `playbook` stops importing `report` and reads the latest report through `repo.Reports`.
- Rows the app does not store (Cloudflare, crawl stats, LCP, Search Console's own index report) are listed once as "read these in …".

**Tech Stack:** Go, SQLite + MySQL 8, i18n catalogs generated into `internal/service/report/catalog_generated.json` by `web/scripts/report-resources.mjs`.

**Commits:** none (user asked). Checkpoints run `git status`.

**Week anchoring (same as `visibleShare`):** `through` = Search Console finalized-through date (`searchWindow`). "This week" ends on the last Sunday on or before `through`. GSC days are Pacific dates and GA4 days are property-time-zone dates; the report states this once under the table.

---

### Task 1: Break the import cycle

**Files:** `internal/service/playbook/service.go`, its test if needed.

- [ ] **Step 1:** In `playbook.Service`, replace the `reports *report.Service` field with `reports *repo.Reports` built as `&repo.Reports{DB: db}`, and in `facts` replace `s.reports.Latest(ctx, slug)` with `s.reports.Latest(ctx, p.ID)`. Check the real signature of `repo.(*Reports).Latest` in `internal/repo/asset.go` and adapt. Remove the `report` import.
- [ ] **Step 2:** `go build ./... && go test ./internal/service/playbook ./internal/api -count=1` → PASS. Run `go list -deps ./internal/service/playbook | grep service/report` → no output.

---

### Task 2: `webstats.WeeklySearch`

**Files:**
- Create: `internal/service/webstats/weekly.go`, `internal/service/webstats/weekly_test.go`
- Modify: `internal/repo/` (one or two small read functions, see Step 3)

- [ ] **Step 1: Types**

```go
// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

// Week holds one complete Monday–Sunday week. A nil value means the week
// was not fully covered by imported data; it is shown as "—", never 0.
type Week struct {
	From                 string   `json:"from"`
	Through              string   `json:"through"`
	Clicks               *float64 `json:"clicks"`
	Impressions          *float64 `json:"impressions"`
	NonBrandClicks       *float64 `json:"non_brand_clicks"`       // visible queries only
	NonBrandImpressions  *float64 `json:"non_brand_impressions"`  // visible queries only
	VisibleClicks        *float64 `json:"visible_clicks"`         // all visible queries, for the non-brand share
	PagesWithImpressions *float64 `json:"pages_with_impressions"`
	NewPages             *float64 `json:"new_pages"`              // publish dates set on the Indexing tab
	OrganicSessions      *float64 `json:"organic_sessions"`
	OrganicEngaged       *float64 `json:"organic_engaged"`
	OrganicKeyEvents     *float64 `json:"organic_key_events"`
	AISessions           *float64 `json:"ai_sessions"`
}

// WeeklySearch is the weekly table's data: this week, last week and four
// weeks ago, plus index facts that exist only as a current snapshot.
type WeeklySearch struct {
	Weeks             [3]Week `json:"weeks"`
	Indexed           int64   `json:"indexed"`
	Inspected         int64   `json:"inspected"`
	PublishedMature   int64   `json:"published_mature"`
	IndexedWithinWeek int64   `json:"indexed_within_week"`
}
```

- [ ] **Step 2: Write the failing test** (`weekly_test.go`), modelled on `TestVisibleShareWeeks` in `playbook_test.go` (same project setup and `s.syncReport` fixtures):
  - Seed 35 days of GSC `daily` rows (10 clicks, 100 impressions per day) and `query` rows: per day one brand query (`Query: p.Name`, 3 clicks, 30 impressions) and one non-brand query (`"best notes app"`, 2 clicks, 20 impressions).
  - Call `s.WeeklySearch(ctx, p, end)` where `end` is a Sunday.
  - Expect `Weeks[0]` (week ending `end`): Clicks 70, Impressions 700, NonBrandClicks 14, NonBrandImpressions 140, VisibleClicks 35; `Weeks[2]` is the week ending `end − 28 days` with the same numbers.
  - Expect GA cells `nil` (no GA data seeded) and `NewPages` `nil` when no URL has a publish date.
  - A second call with `end + 7 days` returns `Weeks[0].Clicks == nil` (not covered).

Run: `go test ./internal/service/webstats -run TestWeeklySearch -count=1` → FAIL (undefined).

- [ ] **Step 3: Implement**

`func (s *Service) WeeklySearch(ctx context.Context, p *model.Project, through time.Time) (*WeeklySearch, error)` — exported for the report; pass `through` from `searchWindow` (add a small exported wrapper `func (s *Service) WeeklyThrough(ctx, p) (time.Time, error)` returning `searchWindow`'s `through`, or make `WeeklySearch` call `searchWindow` itself when `through.IsZero()` — pick one and document it).

For each of the three weeks (`end := weekEnd − 7*{0,1,4}` days, `start := end − 6`), with `property` from `searchWindow` and `weekEnd` computed as in `visibleShare`:

| Field | Source | Gate |
|---|---|---|
| Clicks, Impressions | `ListGscDaily` + `sumOfficial(rows, start, end)` | daily coverage `covered` and `rows == 7` |
| NonBrand*, VisibleClicks | `ListGscSlice(…, "query", start, end)`; sum rows where `SearchType` is `""` or `"web"`; non-brand = `nonbrand(row.Query, p)` (`ctr.go`) | query coverage `covered` |
| PagesWithImpressions | `s.rows.PagesWithImpressions(ctx, p.ID, property, start, end)` | page coverage `covered` |
| NewPages | new repo `CountPublished(ctx, pid, indexProperty, fromUnix, toUnix) (int64, error)` = `COUNT(*) FROM index_urls WHERE project_id=? AND property=? AND published_at BETWEEN ? AND ?`; nil when the project has no URL with a publish date at all (add `HasPublished` or reuse the count over all time) | — |
| Organic* | GA4 `"channel"` report facts for the week where `Channel == "Organic Search"`; sum Sessions, Engaged (nil-safe), KeyEvents (nil-safe). Add a repo reader mirroring `ListGaSessionFacts` for `report = "channel"` | `reportCoverage(…, "ga4", gaProperty, "channel", "", start, end)` `covered` and its `Quality` not `Sampled`, `Thresholded`, `OtherRow` or `Restricted` |
| AISessions | same channel facts, rows where `AISource(row.Source, row.Medium)` | same gate |

`gaProperty` = `officialProperty(ctx, p, "ga4")` only when `p.GA4Property` is set (as in `PlaybookSearch`). Index snapshot: `IndexInventory(ctx, p.Slug, "", "", 1, 1)` → `Indexed`, `Inspected`, `PublishedMature`, `IndexedWithinWeek`.

Check before writing: `model.GaFact` field names (`Channel`, `Source`, `Medium`, `Sessions`, `Engaged`, `KeyEvents`, and how the report kind is stored), and that the GA channel report is named `"channel"` in `ga_facts.go`. Report every adaptation.

- [ ] **Step 4:** `go test ./internal/service/webstats -count=1` → PASS. Add one more test case seeding GA `channel` facts (one `Organic Search` row and one `chatgpt.com / referral` row per day, with coverage) and assert `OrganicSessions` and `AISessions` for `Weeks[0]`.

- [ ] **Step 5: Checkpoint** — `gofmt -l internal && go vet ./internal/... && git status --short`

---

### Task 3: Text

**Files:** `web/src/i18n/locales/{en,zh,pt}.ts` (`weeklyReport` block), then regenerate `internal/service/report/catalog_generated.json`.

- [ ] **Step 1: Add keys inside `weeklyReport`** (en / zh / pt):

| key | en | zh | pt |
|---|---|---|---|
| `stagesLine` | `Stage: SEO {seo} · AI {ai} · product {product}. Playbooks: {link}` | `阶段：SEO {seo} · AI {ai} · 产品 {product}。剧本：{link}` | `Estágio: SEO {seo} · IA {ai} · produto {product}. Roteiros: {link}` |
| `stageUnset` | `not chosen` | `未选择` | `não escolhido` |
| `seoNew` | `new site` | `新站阶段` | `site novo` |
| `seoGrow` | `growth` | `起量阶段` | `crescimento` |
| `noSite` | `no website` | `无网站` | `sem site` |
| `aiReach` | `reachable` | `抓得到` | `acessível` |
| `aiKnown` | `knows you` | `认得你` | `conhece você` |
| `aiRecommend` | `recommends you` | `推荐你` | `recomenda você` |
| `tableNew` | `Weekly table: new site` | `周报表：新站` | `Tabela semanal: site novo` |
| `tableGrow` | `Weekly table: growth` | `周报表：起量` | `Tabela semanal: crescimento` |
| `tableSourceNew` | `Rows from the new-site weekly template in craftsail-book, “SEO monitoring for a new site” §7.` | `行来自 craftsail-book《新站的 SEO 监测》七的新站周报模板。` | `Linhas do modelo semanal para site novo do craftsail-book, “Monitoramento de SEO para site novo” §7.` |
| `tableSourceGrow` | `Rows from the weekly template in craftsail-book, “Monitoring SEO” §8.` | `行来自 craftsail-book《SEO 的监测》八的周报模板。` | `Linhas do modelo semanal do craftsail-book, “Monitoramento de SEO” §8.` |
| `colMetric` | `Metric` | `指标` | `Métrica` |
| `colThis` | `This week` | `本周` | `Esta semana` |
| `colLast` | `Last week` | `上周` | `Semana passada` |
| `colFour` | `4 weeks ago` | `4 周前` | `4 semanas atrás` |
| `colChange` | `Change` | `变化` | `Variação` |
| `colAlert` | `Alert line` | `告警线` | `Linha de alerta` |
| `rowIndexed` | `Indexed URLs (inspected)` | `已收录 URL（已检查）` | `URLs indexadas (verificadas)` |
| `rowIndexedValue` | `{indexed} of {inspected}` | `{inspected} 个中 {indexed} 个` | `{indexed} de {inspected}` |
| `rowSevenDay` | `Indexed within 7 days of publishing` | `发布后 7 天内收录` | `Indexadas até 7 dias após publicar` |
| `rowNewPages` | `New pages (publish dates)` | `新页数（按发布日期）` | `Páginas novas (datas de publicação)` |
| `rowPagesImpr` | `Pages with impressions` | `有展示的网页数` | `Páginas com impressões` |
| `rowImpressions` | `Impressions` | `展示` | `Impressões` |
| `rowClicks` | `Clicks` | `点击` | `Cliques` |
| `rowOrganic` | `Organic search sessions (GA4)` | `自然搜索会话（GA4）` | `Sessões de busca orgânica (GA4)` |
| `rowKeyEvents` | `Organic key events (GA4)` | `自然搜索关键事件（GA4）` | `Eventos-chave orgânicos (GA4)` |
| `rowNbClicks` | `Non-brand clicks (visible queries)` | `非品牌点击（可见查询）` | `Cliques sem marca (consultas visíveis)` |
| `rowNbImpr` | `Non-brand impressions (visible queries)` | `非品牌展示（可见查询）` | `Impressões sem marca (consultas visíveis)` |
| `rowNbCtr` | `Non-brand CTR` | `非品牌 CTR` | `CTR sem marca` |
| `rowNbShare` | `Non-brand share of visible clicks` | `非品牌占可见点击` | `Parcela sem marca dos cliques visíveis` |
| `rowClickToSession` | `Organic sessions per Search Console click` | `每个 Search Console 点击对应的自然搜索会话` | `Sessões orgânicas por clique do Search Console` |
| `rowEngagement` | `Organic engagement rate` | `自然搜索互动率` | `Taxa de engajamento orgânico` |
| `rowEventsPerSession` | `Organic key events per session` | `自然搜索每会话关键事件` | `Eventos-chave orgânicos por sessão` |
| `rowAISessions` | `Sessions from AI referrers` | `AI 来源会话` | `Sessões vindas de IA` |
| `alertWeekDrop` | `week over week below −20%` | `周环比低于 −20%` | `queda semanal abaixo de −20%` |
| `alertHit` | `below the alert line` | `低于告警线` | `abaixo da linha de alerta` |
| `snapshot` | `Index rows are the current snapshot; earlier weeks are not stored.` | `收录相关的行是当前快照，往周数据没有保存。` | `As linhas de indexação são o retrato atual; semanas anteriores não ficam guardadas.` |
| `dayNote` | `Search Console weeks use Pacific dates; GA4 weeks use the property's time zone. “—” means the week is not fully imported.` | `Search Console 按太平洋时间分天，GA4 按资源时区分天。“—”表示这一周的数据没有导入完整。` | `As semanas do Search Console usam datas do Pacífico; as do GA4, o fuso da propriedade. “—” significa que a semana não foi totalmente importada.` |
| `notTracked` | `Not stored here; read them where they live: crawl stats and the page indexing report in Search Console, Googlebot errors and LCP in Cloudflare, AI impressions in Search Console's generative AI report.` | `这些这里不保存，请到原处看：Search Console 的抓取统计和网页索引报告，Cloudflare 的 Googlebot 错误和 LCP，Search Console 的生成式 AI 报告里的 AI 展示。` | `Não ficam guardados aqui; veja onde estão: estatísticas de rastreamento e relatório de indexação no Search Console, erros do Googlebot e LCP no Cloudflare, impressões de IA no relatório de IA generativa do Search Console.` |
| `shipped` | `Shipped this week` | `本周上线的改动` | `Publicado nesta semana` |
| `smallNumbers` | `Numbers are small at this stage: read absolute counts, not percentages.` | `这一阶段数字小：看绝对数，不看百分比。` | `Os números são pequenos neste estágio: leia contagens absolutas, não porcentagens.` |

Remove the keys `stage`, `new_site`, `established`, `indexing`, `search` from `weeklyReport` only if Task 4 removes their last use (grep first).

- [ ] **Step 2: Regenerate** — `cd web && npm run generate:report && npm run check:report` → no error. Add `- run: npm run check:report` to `.github/workflows/ci.yml` after `npm run check:playbook` so catalog drift fails CI, not only release.

---

### Task 4: Render the table in the weekly report

**Files:** Create `internal/service/report/weekly_table.go`, `weekly_table_test.go`; modify `internal/service/report/localized.go` (`weeklyMarkdown`), `report.go` (Service gets `playbook *playbook.Service`, built in `New`).

- [ ] **Step 1: Write the failing renderer test** (pure, like `search_section_test.go`):

```go
func TestWeeklyTableLayouts(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	w := &webstats.WeeklySearch{Indexed: 12, Inspected: 20, PublishedMature: 4, IndexedWithinWeek: 3}
	w.Weeks[0] = webstats.Week{Clicks: f(70), Impressions: f(700), PagesWithImpressions: f(9), NonBrandClicks: f(14), NonBrandImpressions: f(140), VisibleClicks: f(35), OrganicSessions: f(50), OrganicEngaged: f(25), OrganicKeyEvents: f(5)}
	w.Weeks[1] = webstats.Week{Clicks: f(60), Impressions: f(600), NonBrandClicks: f(20), NonBrandImpressions: f(150), VisibleClicks: f(30), OrganicSessions: f(70)}
	var b strings.Builder
	writeWeeklyTable(&b, w, "seoNew", "en")
	out := b.String()
	for _, want := range []string{"Weekly table: new site", "| Indexed URLs (inspected) | 12 of 20 |", "| Impressions | 700 | 600 | — |", "3 / 4", "Pacific"} {
		if !strings.Contains(out, want) {
			t.Fatalf("new-site table missing %q:\n%s", want, out)
		}
	}
	b.Reset()
	writeWeeklyTable(&b, w, "seoGrow", "en")
	out = b.String()
	for _, want := range []string{"Weekly table: growth", "| Non-brand clicks (visible queries) | 14 | 20 | −30% |", "below the alert line", "| Non-brand CTR | 10.0% |", "| Organic search sessions (GA4) | 50 | 70 | −29% |"} {
		if !strings.Contains(out, want) {
			t.Fatalf("growth table missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "{") {
		t.Fatalf("unfilled placeholder:\n%s", out)
	}
}
```

Adjust the expected strings to the exact formatting you implement (number formatting, minus sign), but keep each assertion's meaning: new-site has the three-week layout and snapshot rows; growth has change % and flags a drop past −20%.

- [ ] **Step 2: Implement `writeWeeklyTable(b *strings.Builder, w *webstats.WeeklySearch, seoStage, lang string)`**

- `t := func(k string, vars ...string) string { return reportText(lang, "weeklyReport."+k, vars...) }`
- Cell formatting: `nil → "—"`; counts `%.0f`; ratios: guard zero/nil denominators → "—".
- **`seoNew` layout** (columns Metric | This week | Last week | 4 weeks ago):
  rows `rowIndexed` (this-week cell `rowIndexedValue` from the snapshot, others "—"), `rowSevenDay` (`IndexedWithinWeek / PublishedMature` as "x / y", snapshot only; "—" when `PublishedMature == 0`), `rowNewPages`, `rowPagesImpr`, `rowImpressions`, `rowClicks`, `rowOrganic`, `rowKeyEvents`. Then lines `snapshot`, `smallNumbers`, `tableSourceNew`.
- **`seoGrow` layout** (columns Metric | This week | Last week | Change | Alert line):
  rows `rowNbClicks` (alert `alertWeekDrop`), `rowNbImpr`, `rowNbCtr` (NonBrandClicks/NonBrandImpressions as %), `rowNbShare` (NonBrandClicks/VisibleClicks as %), `rowOrganic` (alert `alertWeekDrop`), `rowClickToSession` (OrganicSessions/Clicks, 2 decimals), `rowEngagement` (OrganicEngaged/OrganicSessions %), `rowEventsPerSession` (OrganicKeyEvents/OrganicSessions, 2 decimals), `rowAISessions`. Change = `(this−last)/last` as a percent with sign, "—" when either is nil or last is 0. When a row with an alert line changes below −20%, append ` · ` + `alertHit` in the Alert column. Then line `tableSourceGrow`.
- Both layouts end with lines `dayNote` and `notTracked`.
- `seoStage == ""` (no site): write nothing.

- [ ] **Step 3: Use it in `weeklyMarkdown`**

- Build `Service.playbook = playbook.New(db)` in `report.New`.
- Right after the title/demo banner, write the stages line: call `s.playbook.Status(ctx, p.Slug)`; on error skip the line. `seo` = `t(status.SeoStage)` or `t("noSite")` when empty; `ai` = `t(status.AiStage)`; `product` = the `playbook.stage.<s>.name` text is not in the report catalog, so use `"S0"/"S1"/"S2"` or `t("stageUnset")`; `link` = `/p/<slug>/help#start` (escape like the `next` section's links).
- Replace the whole `heading("stage")` block (the `stage` line, the `indexing` line, the `search` line) with: compute `through` (via the wrapper chosen in Task 2), `weekly, err := web.WeeklySearch(ctx, p, through)`; on error write `line(t("unavailable"))`; else `writeWeeklyTable(&b, weekly, status.SeoStage, lang)`.
- Add a `shipped` section right after the table: observations from the already-loaded list whose `ReleasedAt` falls in this week (Monday 00:00 to now, server local time), latest per `TaskID`, as `- CODE · hypothesis`; `t("none")` when empty. Reuse the `observations` slice that the `observations` section loads (move its loading earlier) instead of querying twice.
- Keep every other section unchanged.

- [ ] **Step 4: Run** — `go test ./internal/service/report -count=1` (including `localized_test.go`, which checks section keys and unresolved placeholders in en/zh/pt; update its expected section list if it asserts on the removed `stage` heading) → PASS. Then `go test ./... -count=1`, `go vet ./...`, `gofmt -l .`.

- [ ] **Step 5: Checkpoint** — `git status --short`

---

### Task 5: Verify in the app

- [ ] `make build`, start the demo (fresh DB in a temp dir) on port 8799, log in as `demo`.
- [ ] Generate the report in each language for `quillpad-new-site` and `quillpad-established` (Reports page, or `POST /api/projects/<slug>/report` with the language the Reports page sends — read `web/src/features/reports/reports.tsx`), then read the stored report (`GET /api/projects/<slug>/report`).
- [ ] Check: new-site report shows "Weekly table: new site" with three week columns; established shows "Weekly table: growth" with Change and Alert columns; the stages line matches what Help › Start here shows for that project; no `{…}` and no raw `weeklyReport.` keys; "—" (not 0) for weeks without data; zh and pt render their own text.
- [ ] Re-run `/tmp/pw/cg-help.mjs`, `cg-status.mjs`, `cg-pages.mjs` (fix only test-script expectations if labels moved).
- [ ] Report results; nothing committed.
