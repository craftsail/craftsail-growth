# Page Consolidation Implementation Plan (Phase 3)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every page or tab answers a step in the playbooks. Search goes from 8 tabs to 5; query fan-out becomes a tab of Citations. No data or backend changes.

**Architecture:** Navigation lives in `web/src/app/nav.ts` (sidebar items, tabs, `LEGACY_REDIRECTS`), routes in `web/src/app.tsx`. Countries and devices become a "Split by" card on the Performance tab that reuses `SearchExplorer`. Channels and landing pages become one "Arrivals" tab that reuses `GAExplorer` with a view switch kept in the URL (`?view=landing`). Old addresses redirect.

**Tech Stack:** React 18 + TypeScript + react-router 6, i18n en/zh/pt.

**Spec:** `docs/superpowers/specs/2026-10-10-playbook-guided-growth-design.md` §四.3. The "CTR reference and page mapping into row detail" item is already true in the code (`CTRReferenceCard` renders in the query detail; `LandingMapping` opens on a row click), so this plan does not touch them.

**Commits:** none; the user asked for no commits. Checkpoints run `git status`.

**Result:**

| Before | After |
|---|---|
| Sidebar: Visibility, Share of voice, Citations, Fan-out, Answers | Visibility, Share of voice, Citations (tabs: Citations, Fan-out), Answers |
| Search tabs: Performance, Indexing, Keywords, Pages, Countries, Devices, Channels, Landings | Performance (with "Split by: Country / Device"), Indexing, Keywords, Pages, Arrivals (view: Channels / Landing pages) |
| `search/countries`, `search/devices` | redirect to `search?dim=country` / `search?dim=device` |
| `search/channels`, `search/landings` | redirect to `search/arrivals` / `search/arrivals?view=landing` |
| `ai/fan-out` | unchanged address, now a tab under Citations |

---

### Task 1: Navigation, routes, redirects, text

**Files:** `web/src/app/nav.ts`, `web/src/app.tsx`, `web/src/i18n/locales/{en,zh,pt}.ts`

- [ ] **Step 1: `nav.ts`**

In the `measure` section, replace the `citations` and `fanout` items with:

```ts
      {
        id: "citations", label: "nav.citations", icon: "quote", to: "ai/citations", tabs: [
          { label: "nav.citations", to: "ai/citations" },
          { label: "nav.fanout", to: "ai/fan-out" },
        ],
      },
```

In the `search` item, replace the `tabs` array with:

```ts
        id: "search", label: "nav.search", icon: "search", to: "search", tabs: [
          { label: "nav.tabs.performance", to: "search" },
          { label: "nav.tabs.indexing", to: "search/indexing" },
          { label: "nav.tabs.keywords", to: "search/keywords" },
          { label: "nav.tabs.pages", to: "search/pages" },
          { label: "nav.tabs.arrivals", to: "search/arrivals" },
        ],
```

Set `LEGACY_REDIRECTS`:

```ts
export const LEGACY_REDIRECTS: Record<string, string> = {
  "search/countries": "search?dim=country",
  "search/devices": "search?dim=device",
  "search/channels": "search/arrivals",
  "search/landings": "search/arrivals?view=landing",
};
```

Remove `"split"` from `NavIcon` only if nothing else uses it (check `shell.tsx`'s icon map; remove the map entry too if you remove the type member).

`itemFor` has a special case `if (leaf.startsWith("ai/prompts/")) return NAV[1].items[0];` — still correct (Visibility is still the first item of section 1).

- [ ] **Step 2: `app.tsx` routes**

Delete the four routes `search/countries`, `search/devices`, `search/channels`, `search/landings`. Add:

```tsx
        <Route path="search/arrivals" element={<Arrivals />} />
```

and `import { Arrivals } from "./features/search/arrivals";`. Remove imports that become unused (`SearchExplorer`, `GAExplorer`) — `tsc` with `noUnusedLocals` will tell you.

Check how the `LEGACY_REDIRECTS` routes render: `<Navigate to={`../${to}`} replace />`. Confirm in the browser (Task 4) that `../search?dim=country` lands on `/p/<slug>/search?dim=country`; if react-router resolves it differently, split `to` into pathname and search: `<Navigate to={{ pathname: `../${path}`, search }} replace />`.

- [ ] **Step 3: Text (all three locales)**

Add to the `nav.tabs` block:

| key | en | zh | pt |
|---|---|---|---|
| `nav.tabs.arrivals` | `Arrivals` | `到站` | `Chegadas` |

Add a new top-level block `searchViews` (after `searchSegments`):

```ts
  searchViews: {
    splitBy: "Split by",          // zh "按维度切开"   pt "Dividir por"
    splitHint: "Use a split to find where a change happened, not as a daily view.", // zh "用来找变化发生在哪里，不用每天看。" pt "Use para achar onde uma mudança aconteceu, não como visão diária."
    country: "Country",           // zh "国家"         pt "País"
    device: "Device",             // zh "设备"         pt "Dispositivo"
    channels: "Channels",         // zh "来源渠道"     pt "Canais"
    landings: "Landing pages",    // zh "落地页"       pt "Páginas de entrada"
    arrivalsHint: "Visits from Google Analytics: where people came from and which page they landed on.", // zh "来自 Google Analytics 的访问：人从哪里来、落在哪个页面。" pt "Visitas do Google Analytics: de onde as pessoas vieram e em qual página chegaram."
  },
```

(Write each locale's own strings; the comments give zh and pt.) Then search for `nav.tabs.channels`, `nav.tabs.landings`, `searchSegments.countries`, `searchSegments.devices` outside the locale files; delete those keys from all three locales only if nothing uses them any more.

- [ ] **Step 4: Verify** — `cd web && ./node_modules/.bin/tsc --noEmit` (errors only about the missing `./features/search/arrivals` module, created in Task 2).

---

### Task 2: Arrivals tab

**Files:** Create `web/src/features/search/arrivals.tsx`; modify `web/src/features/search/ga-explore.tsx`

- [ ] **Step 1: Keep the view across filter changes in `GAExplorer`**

In `apply` and in the Reset button, keep `view` like `lang` is kept:

```tsx
    for (const k of ["lang", "view"]) if (params.has(k)) next.set(k, params.get(k)!);
```

(replace the existing single `lang` line in `apply`; for Reset, build the same object: `const keep = new URLSearchParams(); for (const k of ["lang","view"]) if (params.has(k)) keep.set(k, params.get(k)!); setParams(keep);`). `resolved()` / `page()` start from the current params, so they already keep it — verify.

- [ ] **Step 2: Create `arrivals.tsx`**

```tsx
// SPDX-License-Identifier: AGPL-3.0-or-later

import { useSearchParams } from "react-router-dom";
import { useI18n } from "../../i18n";
import { GAExplorer } from "./ga-explore";

// Arrivals is one tab for Google Analytics visits: by channel or by landing
// page. The view lives in the URL so links and reloads keep it.
export function Arrivals() {
  const { t } = useI18n();
  const [params, setParams] = useSearchParams();
  const view = params.get("view") === "landing" ? "landing" : "channel";
  const choose = (v: "channel" | "landing") => {
    const next = new URLSearchParams();
    if (params.has("lang")) next.set("lang", params.get("lang")!);
    if (v === "landing") next.set("view", "landing");
    setParams(next);
  };
  return (
    <div className="flex flex-col gap-4">
      <p className="page-description">{t("searchViews.arrivalsHint")}</p>
      <div className="flex gap-2" role="group" aria-label={t("nav.tabs.arrivals")}>
        {(["channel", "landing"] as const).map((v) => (
          <button key={v} type="button" aria-pressed={view === v} onClick={() => choose(v)}
            className={"btn " + (view === v ? "btn-primary" : "btn-secondary")}>
            {t(v === "channel" ? "searchViews.channels" : "searchViews.landings")}
          </button>
        ))}
      </div>
      <GAExplorer key={view} report={view} />
    </div>
  );
}
```

Switching view clears filters on purpose: channel and landing reports take different sort keys. Check that `btn-primary` / `btn-secondary` match how other toggles in the app look (e.g. the filter bar in `features/measure/filters.tsx`); if the app uses a different chip/segmented style for toggles, use that instead. No new colors.

- [ ] **Step 3: Verify** — `tsc --noEmit` clean.

---

### Task 3: "Split by" on Performance

**Files:** Modify `web/src/features/search/webstats.tsx`, `web/src/features/search/explore.tsx`

- [ ] **Step 1: Keep `dim` in `SearchExplorer`**

In `apply`, after the `lang` line: `if (params.has("dim")) next.set("dim", params.get("dim")!);`. In Reset: keep `lang` and `dim` the same way (`value` is still kept as today). `page()` starts from current params — verify.

- [ ] **Step 2: Add the card to `Webstats`**

Add a component in `webstats.tsx`:

```tsx
// SplitBy shows country and device lists on demand. They answer "where did
// a change happen", a diagnostic step, so they sit under the overview
// numbers instead of having their own tabs.
function SplitBy() {
  const { t } = useI18n();
  const [params, setParams] = useSearchParams();
  const dim = params.get("dim") === "device" ? "device" : params.get("dim") === "country" ? "country" : "";
  const choose = (d: "" | "country" | "device") => {
    const next = new URLSearchParams();
    if (params.has("lang")) next.set("lang", params.get("lang")!);
    if (d) next.set("dim", d);
    setParams(next);
  };
  return (
    <section className="mt-5 card p-5" id="split">
      <div className="stat-label">{t("searchViews.splitBy")}</div>
      <p className="hint">{t("searchViews.splitHint")}</p>
      <div className="mt-3 flex gap-2" role="group" aria-label={t("searchViews.splitBy")}>
        {(["country", "device"] as const).map((d) => (
          <button key={d} type="button" aria-pressed={dim === d} onClick={() => choose(dim === d ? "" : d)}
            className={"btn " + (dim === d ? "btn-primary" : "btn-secondary")}>
            {t(`searchViews.${d}`)}
          </button>
        ))}
      </div>
      {dim && <div className="mt-4"><SearchExplorer key={dim} kind={dim} /></div>}
    </section>
  );
}
```

Import `useSearchParams` and `SearchExplorer` (`./explore`). Render `<SplitBy />` as the last element inside `Webstats`'s main returned content (the branch that renders the data, not the loading/empty branch). If `SearchExplorer` renders its own page-level heading or description that looks wrong inside a card, pass nothing new — just note it in the report; do not restyle `SearchExplorer`.

When `dim` is present on load (from the redirect), scroll the card into view once: `useEffect(() => { if (dim) document.getElementById("split")?.scrollIntoView(); }, [])` inside `SplitBy` (eslint-free; the repo has no lint step).

- [ ] **Step 3: Verify** — `tsc --noEmit` clean, `npm run build` ok.

---

### Task 4: Playbook links, help, checks, browser

**Files:** `web/src/features/playbook/catalog.ts`, possibly `web/src/features/help/*`, Playwright script in `/tmp/pw`

- [ ] **Step 1: Catalog**

Change step `seoPeopleWeek` page to `{ path: "search/arrivals", label: "nav.tabs.arrivals" }`. Grep `catalog.ts`, `features/help/` and the locale files for any other reference to the removed addresses or labels and update them.

- [ ] **Step 2: Static checks**

Run: `cd web && ./node_modules/.bin/tsc --noEmit && npm run check:cjk && npm run check:playbook && npm run build` and `cd .. && go test ./... -count=1` (no Go change, guards the embed).

- [ ] **Step 3: Browser (en, zh, pt)** — rebuild (`make build`), start the demo as in earlier phases on port 8799, then check:

1. Sidebar has no Fan-out item; Citations shows two tabs; `ai/fan-out` opens with the Fan-out tab active and the page title "Citations".
2. Search shows exactly 5 tabs in the order above.
3. `/search/countries` → `/search?dim=country` with the country list visible; `/search/devices` likewise; `/search/channels` → `/search/arrivals` (Channels pressed); `/search/landings` → `/search/arrivals?view=landing` (Landing pages pressed).
4. On Arrivals, apply a filter: the view stays; switch view: filters reset.
5. On Performance, toggling Country / Device shows and hides the list; applying a filter in the list keeps `dim`.
6. Help › SEO: new site → the GA4 step's "Where" link reads "Search › Arrivals" and opens it. `npm run check:playbook` already passed.
7. The info icon on `search/arrivals` opens SEO: new site; on `ai/fan-out` opens AI: recommends you.
8. No page errors; re-run `/tmp/pw/cg-help.mjs` and `/tmp/pw/cg-status.mjs` — both still pass.

- [ ] **Step 4: Report** — what passed, screenshots path, nothing committed.
