# Playbook Help Center Implementation Plan (Phase 0 + 1)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the help center from a feature manual into stage-based playbooks taken from craftsail-book, so a user can read "which stage am I in, what do I do, when have I passed" without guessing.

**Architecture:** One step catalog (`web/src/features/playbook/catalog.ts`) holds every playbook step, pass signal, threshold and book reference as data. The help center renders it through `web/src/features/help/playbook.tsx`; all prose lives in the three i18n catalogs. Phase 2 (separate plan) adds per-project status to the same playbooks; the overview only links to them. Before the product, the book gets the missing chapter on monitoring AI recommendations.

**Tech Stack:** React 18 + TypeScript + Vite, Tailwind v4, i18n typed against `en.ts` (missing keys fail `tsc`). No frontend test runner exists; verification is `tsc --noEmit`, `npm run check:cjk`, a new `npm run check:playbook`, `npm run build`, and a browser pass in three languages.

**Spec:** `docs/superpowers/specs/2026-10-10-playbook-guided-growth-design.md`

**Commits:** The user asked not to commit. Every task ends with a `git status` checkpoint instead of a commit. Do not commit in either repository.

**Out of scope here (later plans):** playbook status API, status in the help-center playbooks, overview link, `search_stage` promotion rule, page consolidation (search 8→5 tabs, fan-out under citations), stage-specific weekly reports.

---

## File map

| File | Change | Responsibility |
|---|---|---|
| `../craftsail-book/seo/AI推荐的监测.md` | Create | Book chapter the AI playbook cites |
| `../craftsail-book/README.md` | Modify | List the new chapter |
| `web/src/features/playbook/catalog.ts` | Create | Steps, signals, thresholds, book paths; compile-time key checks |
| `web/src/features/help/playbook.tsx` | Create | Renders playbook topics from the catalog |
| `web/src/features/help/kit.tsx` | Modify | New topic ids, groups |
| `web/src/features/help/usage.tsx` | Modify | Drop feature topics now covered by playbooks; troubleshooting by symptom |
| `web/src/features/help/content/{en,zh,pt}.tsx` | Modify | New group names; merge playbook bodies |
| `web/src/features/help/help.tsx` | Modify | Old `#topic` links land on the replacing playbook |
| `web/src/app/tips.ts` | Modify | Button tips point at playbook topics |
| `web/src/app/nav.ts` | Modify | Page info icons point at playbook topics |
| `web/src/i18n/locales/{en,zh,pt}.ts` | Modify | `playbook` block, new `helpTopics`, new issues, remove dead keys |
| `web/scripts/check-playbook.mjs` | Create | Fails when a step links to a page that is not in `NAV` |
| `web/package.json`, `.github/workflows/ci.yml` | Modify | Run the check |

---

### Task 0: Write the book chapter 《AI 推荐的监测》

The AI playbook steps cite `seo/AI推荐的监测.md` §3, §4, §5, §6. That chapter must exist, with exactly these numbered sections, before the product cites it. Write it in the book's style: Chinese, short sentences, tables, formulas in code blocks, an evidence level for each claim, references at the end. Follow the structure of `seo/新站的SEO监测.md`.

**Files:**
- Create: `/Users/askuy/code/my/github/craftsail/craftsail-book/seo/AI推荐的监测.md`
- Modify: `/Users/askuy/code/my/github/craftsail/craftsail-book/README.md` (the 现在可以读 table)

- [ ] **Step 1: Read the sources the chapter builds on**

Read in craftsail-book: `seo/新站的SEO监测.md` (structure to copy), `seo/SEO落地.md` §8 (robots, llms.txt) and §10, `growth/目录站与awesome列表.md` §6 (AI 推荐和目录的关系), `growth/流量公式与阶段权重.md` §4 (AI 推荐 row). Read in craftsail-growth: `web/src/features/help/kit.tsx` `FX` (visibility, recognition, sov, ownCite, wilson, change, stability, layers), these are the formulas the product computes and the chapter must state identically.

- [ ] **Step 2: Verify external sources with WebSearch/WebFetch**

Fetch and cite at least: the GEO paper (Aggarwal et al., "GEO: Generative Engine Optimization", arXiv 2311.09735); OpenAI's GPTBot / OAI-SearchBot documentation; Anthropic's ClaudeBot documentation; Perplexity's PerplexityBot documentation; Google's documentation on `Google-Extended` and AI features in Search; the llms.txt proposal (llmstxt.org). Record the exact URLs and what each one supports. Do not cite anything you did not open.

- [ ] **Step 3: Write the chapter with these sections**

```markdown
# AI 推荐的监测

## 一、这个渠道在问什么
   用户问 AI“有什么好用的 xxx”，AI 回答里提到谁，用户就从里面挑。三个问题按顺序：
   抓得到 → 认得你 → 推荐你（以及依据是什么）。和 SEO 的关系：AI 的依据大多来自网页和站外提及。
   在流量公式里的位置：AI 推荐 S0 ★★ / S1 ★★★ / S2 ★★★★，依赖站外提及。

## 二、三个阶段
   表格：阶段 | 问题 | 做什么 | 什么时候算过关
   AI-1 抓得到 | AI-2 认得你 | AI-3 推荐你。

## 三、怎么量
   1. 回答每次都不一样：同一问题多次采样，比率是估计值
   2. 可见性、认知度、首位/前三、声量、引用自有域名（公式与 craftsail-growth FX 一字不差）
   3. 区间：Wilson 95%，n < 30 算小样本；举例 5/20 → 11.2%–46.9%
   4. 变没变：Newcombe 差值区间，整个区间在 0 一侧才算变
   5. 不能比的东西：跨引擎比引用数、API 和网页版混在一起、改了问题前后比
   6. 引用稳定度（Bray–Curtis），只用来判断来源是否还开放

## 四、抓得到
   1. robots.txt 和各家 AI 爬虫（GPTBot、OAI-SearchBot、ClaudeBot、PerplexityBot、Google-Extended）分别管什么
   2. CDN / WAF 默认拦截
   3. 不执行 JavaScript 时正文在不在
   4. llms.txt：是什么、谁在读、证据等级
   过关：体检访问层、发现层 pass

## 五、认得你
   1. 品牌问题和非品牌问题分开统计，为什么
   2. 写一份一致的品牌事实：一句话定义、别名、目标用户、价格、带出处的数字
   3. 读回答：说错的事实追到它依据的网页
   过关：认知度 n ≥ 30 且 Wilson 下界 ≥ 50%（先验，待校准）

## 六、推荐你
   1. 问题怎么写：买家原话，按品类和场景；改问题就是改测量对象
   2. 竞品怎么选：买家真的会比的
   3. 引用缺口：竞品被引、你没被引的问题；反复出现的目录、社区、测评站
   4. 去这些来源上出现：每周一个，记成实验（链接《目录站与 awesome 列表》《怎么找社区》）

## 七、节奏
   一次 / 每天（自动采样）/ 每周 30 分钟 / 每月；周报模板表格

## 八、什么时候算过了这一关
   每个阶段的过关信号表；哪些能自动判定

## 九、常见误判
   单日比率、跨引擎比引用数、把认知度当可见性、改了问题还比前后、小样本下的“下降”

## 参考资料
   按“AI 爬虫”“生成式引擎优化”“统计”分组，只列 Step 2 实际打开过的链接
```

Every number in the chapter must either come from a cited source (mark the evidence level: 标准 / 官方文档 / 实验 / 观察 / 经验) or be labelled 先验 with the reason. Section numbers 三, 四, 五, 六 are fixed because the product cites them.

- [ ] **Step 4: Add the chapter to the book README**

In `README.md`, in the `| 章节 | 讲什么 |` table, insert after the `新站的 SEO 监测` row:

```markdown
| [AI 推荐的监测](seo/AI推荐的监测.md) | 用户问 AI 时有没有你：抓得到、认得你、推荐你三个阶段；可见性、认知度、声量、引用怎么量，为什么要多次采样和给区间；robots 和各家 AI 爬虫、llms.txt；引用缺口和站外提及；节奏、过关信号和常见误判 |
```

- [ ] **Step 5: Ask the author to review the chapter**

Stop here and show the user the chapter. Tasks 1–8 do not depend on its wording, only on the file path and section numbers 三/四/五/六, so they can proceed in parallel, but do not tell the user the work is done until the chapter is approved.

- [ ] **Step 6: Checkpoint**

Run: `cd /Users/askuy/code/my/github/craftsail/craftsail-book && git status --short`
Expected: `seo/AI推荐的监测.md` untracked, `README.md` modified. Do not commit.

---

### Task 1: Step catalog

**Files:**
- Create: `web/src/features/playbook/catalog.ts`

The catalog will not compile until Task 2 adds the i18n keys (the compile-time checks reference them). That is intended: Task 2 follows directly.

- [ ] **Step 1: Create the catalog**

```ts
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Key } from "../../i18n";
import type { TopicId } from "../help/kit";

// The playbook is craftsail-book's method as data. Every step cites the
// chapter it comes from; a method that is not in the book gets no step.
// The help center renders this catalog; guidance stays out of the overview.

export const BOOK_BASE = "https://github.com/craftsail/craftsail-book/blob/main/";

// Chapter paths, URL-encoded so this file has no CJK text (check:cjk).
export const BOOKS = {
  model: "growth/%E6%8A%8A%E8%BF%90%E8%90%A5%E5%BD%93%E6%88%90%E4%B8%80%E9%81%93%E5%BB%BA%E6%A8%A1%E9%A2%98.md",
  formula: "growth/%E6%B5%81%E9%87%8F%E5%85%AC%E5%BC%8F%E4%B8%8E%E9%98%B6%E6%AE%B5%E6%9D%83%E9%87%8D.md",
  pmf: "growth/PMF%E9%AA%8C%E8%AF%81%E4%B8%8E%E7%AC%AC%E4%B8%80%E6%89%B9%E7%94%A8%E6%88%B7.md",
  communities: "growth/%E6%80%8E%E4%B9%88%E6%89%BE%E7%A4%BE%E5%8C%BA.md",
  directories: "growth/%E7%9B%AE%E5%BD%95%E7%AB%99%E4%B8%8Eawesome%E5%88%97%E8%A1%A8.md",
  outreach: "growth/%E5%86%B7%E5%A4%96%E8%81%94.md",
  toolGrowth: "growth/%E5%B7%A5%E5%85%B7%E4%BA%A7%E5%93%81%E5%87%BA%E6%B5%B7%E8%BF%90%E8%90%A5.md",
  toolData: "growth/%E5%B7%A5%E5%85%B7%E4%BA%A7%E5%93%81%E7%9C%8B%E6%95%B0%E6%8D%AE.md",
  seoSetup: "seo/SEO%E8%90%BD%E5%9C%B0.md",
  seoMonitor: "seo/SEO%E7%9A%84%E7%9B%91%E6%B5%8B.md",
  seoNew: "seo/%E6%96%B0%E7%AB%99%E7%9A%84SEO%E7%9B%91%E6%B5%8B.md",
  ai: "seo/AI%E6%8E%A8%E8%8D%90%E7%9A%84%E7%9B%91%E6%B5%8B.md",
} as const;
export type BookId = keyof typeof BOOKS;
// section is the chapter's section number, "8.1" for section 8 item 1; omit for the whole chapter.
export type BookRef = { book: BookId; section?: string };

export const PLAYBOOK_TOPICS = ["start", "seoNew", "seoGrow", "aiReach", "aiKnown", "aiRecommend", "weekly"] as const satisfies readonly TopicId[];
export type PlaybookTopic = (typeof PLAYBOOK_TOPICS)[number];
export const CADENCES = ["once", "daily", "weekly", "monthly"] as const;
export type Cadence = (typeof CADENCES)[number];

// Pass thresholds are our starting assumptions, published in the help
// center. Phase 2 judges the signals with the same numbers.
export const THRESHOLDS = {
  newPageDays: 3, sitemapRate: 80, sitemapSwing: 5, weeks: 4, imprCoverage: 50,
  visibleShare: 50, recognitionN: 30, recognitionLower: 50, indexDrop: 5,
} as const;

type Step = { id: string; topic: PlaybookTopic; cadence: Cadence; page: { path: string; label: Key }; books: readonly BookRef[] };

// Order inside a topic and cadence is the reading order.
export const STEPS = [
  { id: "project", topic: "start", cadence: "once", page: { path: "settings/projects", label: "nav.projects" }, books: [{ book: "model" }] },
  { id: "google", topic: "start", cadence: "once", page: { path: "settings/google", label: "nav.google" }, books: [{ book: "seoMonitor", section: "2" }] },
  { id: "engines", topic: "start", cadence: "once", page: { path: "settings/providers", label: "nav.providers" }, books: [{ book: "model" }] },

  { id: "seoLayers", topic: "seoNew", cadence: "once", page: { path: "audit", label: "nav.audit" }, books: [{ book: "seoSetup", section: "15" }] },
  { id: "seoSitemap", topic: "seoNew", cadence: "once", page: { path: "audit/issues", label: "nav.tabs.issues" }, books: [{ book: "seoSetup", section: "8.2" }, { book: "seoNew", section: "3.1" }] },
  { id: "seoInspect", topic: "seoNew", cadence: "daily", page: { path: "search/indexing", label: "nav.tabs.indexing" }, books: [{ book: "seoNew", section: "3.4" }] },
  { id: "seoIndexWeek", topic: "seoNew", cadence: "weekly", page: { path: "search/indexing", label: "nav.tabs.indexing" }, books: [{ book: "seoNew", section: "3" }] },
  { id: "seoImprWeek", topic: "seoNew", cadence: "weekly", page: { path: "search", label: "nav.tabs.performance" }, books: [{ book: "seoNew", section: "4" }] },
  { id: "seoPeopleWeek", topic: "seoNew", cadence: "weekly", page: { path: "search/channels", label: "nav.tabs.channels" }, books: [{ book: "seoNew", section: "5" }] },
  { id: "seoZeroMonth", topic: "seoNew", cadence: "monthly", page: { path: "search/pages", label: "nav.tabs.pages" }, books: [{ book: "seoNew", section: "7" }] },
  { id: "seoTitlesMonth", topic: "seoNew", cadence: "monthly", page: { path: "search/pages", label: "nav.tabs.pages" }, books: [{ book: "seoNew", section: "7" }] },

  { id: "growAlerts", topic: "seoGrow", cadence: "daily", page: { path: "search", label: "nav.tabs.performance" }, books: [{ book: "seoMonitor", section: "6.13" }] },
  { id: "growBrand", topic: "seoGrow", cadence: "weekly", page: { path: "search/keywords", label: "nav.tabs.keywords" }, books: [{ book: "seoMonitor", section: "3.2" }] },
  { id: "growStriking", topic: "seoGrow", cadence: "weekly", page: { path: "opportunities", label: "nav.actionPlan" }, books: [{ book: "seoMonitor", section: "6.4" }] },
  { id: "growDecline", topic: "seoGrow", cadence: "weekly", page: { path: "search/pages", label: "nav.tabs.pages" }, books: [{ book: "seoMonitor", section: "6.14" }] },
  { id: "growCtrMonth", topic: "seoGrow", cadence: "monthly", page: { path: "search/keywords", label: "nav.tabs.keywords" }, books: [{ book: "seoMonitor", section: "6.2" }, { book: "seoMonitor", section: "6.3" }] },

  { id: "aiBots", topic: "aiReach", cadence: "once", page: { path: "audit/issues", label: "nav.tabs.issues" }, books: [{ book: "seoSetup", section: "8.1" }, { book: "ai", section: "4" }] },
  { id: "aiLlms", topic: "aiReach", cadence: "once", page: { path: "audit", label: "nav.audit" }, books: [{ book: "seoSetup", section: "8.3" }, { book: "ai", section: "4" }] },
  { id: "aiNoJs", topic: "aiReach", cadence: "once", page: { path: "audit/pages", label: "nav.tabs.pages" }, books: [{ book: "seoSetup", section: "10.3" }] },

  { id: "knownFacts", topic: "aiKnown", cadence: "once", page: { path: "settings/brand", label: "nav.brand" }, books: [{ book: "toolGrowth", section: "3.1" }, { book: "ai", section: "5" }] },
  { id: "knownRecognition", topic: "aiKnown", cadence: "weekly", page: { path: "ai/visibility", label: "nav.visibility" }, books: [{ book: "ai", section: "3" }] },
  { id: "knownRead", topic: "aiKnown", cadence: "weekly", page: { path: "ai/answers", label: "nav.answers" }, books: [{ book: "ai", section: "5" }] },

  { id: "recQuestions", topic: "aiRecommend", cadence: "once", page: { path: "settings/questions", label: "nav.questions" }, books: [{ book: "toolGrowth", section: "3" }, { book: "pmf", section: "3.2" }] },
  { id: "recCompetitors", topic: "aiRecommend", cadence: "once", page: { path: "settings/competitors", label: "nav.competitors" }, books: [{ book: "toolGrowth", section: "3.2" }] },
  { id: "recVisibility", topic: "aiRecommend", cadence: "weekly", page: { path: "ai/visibility", label: "nav.visibility" }, books: [{ book: "ai", section: "3" }] },
  { id: "recCitations", topic: "aiRecommend", cadence: "weekly", page: { path: "ai/citations", label: "nav.citations" }, books: [{ book: "ai", section: "6" }, { book: "directories", section: "6" }] },
  { id: "recGetListed", topic: "aiRecommend", cadence: "weekly", page: { path: "opportunities", label: "nav.actionPlan" }, books: [{ book: "directories", section: "8" }, { book: "communities", section: "3" }] },

  { id: "weekRead", topic: "weekly", cadence: "weekly", page: { path: "overview", label: "nav.overview" }, books: [{ book: "seoNew", section: "7" }, { book: "seoMonitor", section: "7" }] },
  { id: "weekPick", topic: "weekly", cadence: "weekly", page: { path: "opportunities", label: "nav.actionPlan" }, books: [{ book: "toolGrowth", section: "11.1" }] },
  { id: "weekShip", topic: "weekly", cadence: "weekly", page: { path: "opportunities", label: "nav.actionPlan" }, books: [{ book: "toolGrowth", section: "11.2" }] },
  { id: "weekReview", topic: "weekly", cadence: "weekly", page: { path: "opportunities", label: "nav.actionPlan" }, books: [{ book: "toolData", section: "9.3" }] },
  { id: "weekReport", topic: "weekly", cadence: "weekly", page: { path: "reports", label: "nav.reports" }, books: [{ book: "seoNew", section: "7" }, { book: "seoMonitor", section: "8" }] },
] as const satisfies readonly Step[];
export type StepId = (typeof STEPS)[number]["id"];

type Signal = { id: string; topic: PlaybookTopic; auto: boolean };

// Pass signals per stage. auto: phase 2 can judge it from stored data;
// otherwise the user confirms it by hand.
export const SIGNALS = [
  { id: "newFast", topic: "seoNew", auto: true },
  { id: "sitemapStable", topic: "seoNew", auto: true },
  { id: "discoveredDown", topic: "seoNew", auto: false },
  { id: "crawlUp", topic: "seoNew", auto: false },
  { id: "imprCoverage", topic: "seoNew", auto: true },
  { id: "visibleShare", topic: "seoNew", auto: true },
  { id: "brandFilter", topic: "seoNew", auto: false },
  { id: "gaThreshold", topic: "seoNew", auto: true },
  { id: "accessPass", topic: "aiReach", auto: true },
  { id: "discoverPass", topic: "aiReach", auto: true },
  { id: "recognized", topic: "aiKnown", auto: true },
] as const satisfies readonly Signal[];
export type SignalId = (typeof SIGNALS)[number]["id"];

// Topics that show a "don't read these yet" warning.
export const AVOID_TOPICS = ["seoNew", "seoGrow", "aiRecommend"] as const satisfies readonly PlaybookTopic[];

// S0 readers are sent to the chapters on finding first users.
export const S0_BOOKS: readonly BookRef[] = [{ book: "pmf" }, { book: "communities" }, { book: "directories" }, { book: "outreach" }];

// Compile-time checks: every id above has its text in the catalogs.
type Assert<T extends true> = T;
export type StepKeysExist = Assert<`playbook.steps.${StepId}.${"do" | "why" | "done"}` extends Key ? true : false>;
export type SignalKeysExist = Assert<`playbook.signals.${SignalId}.${"name" | "rule"}` extends Key ? true : false>;
export type BookKeysExist = Assert<`playbook.books.${BookId}` extends Key ? true : false>;
export type TopicKeysExist = Assert<`playbook.asks.${PlaybookTopic}` extends Key ? true : false>;
```

- [ ] **Step 2: Checkpoint**

Run: `git status --short web/src/features/playbook`
Expected: `?? web/src/features/playbook/`. (`tsc` fails until Task 2.)

---

### Task 2: Playbook text in three languages

Add one `playbook` block to each locale, right after the `helpCenter` block (en.ts line ~1571, zh.ts/pt.ts ~1573). zh and pt must have exactly the same keys as en; `tsc` enforces it.

**Files:**
- Modify: `web/src/i18n/locales/en.ts`, `zh.ts`, `pt.ts`

- [ ] **Step 1: Add the English block to `en.ts` after `helpCenter: { ... },`**

```ts
  playbook: {
    ask: "What this stage asks",
    avoid: "Don't read these yet",
    why: "Why",
    done: "Done when",
    open: "Where",
    pass: "When you have passed this stage",
    signal: "Signal",
    rule: "How it is judged",
    judged: "Check",
    auto: "Automatic",
    manual: "By hand",
    passNote: "These thresholds are our starting assumptions, not Google or engine rules. We will tune them with real projects and write the result back into the book.",
    bookNote: "Each step cites the craftsail-book chapter it comes from (in Chinese).",
    cadence: { once: "Once", daily: "Every day (automatic)", weekly: "Every week, 30 minutes", monthly: "Every month" },
    asks: {
      start: "Before any numbers: which stage is your product in? SEO and AI recommendations are slow channels. How much time they deserve depends on the stage, so decide that first, then do the one-time setup.",
      seoNew: "Does Google accept this site? Four questions, in order: are new pages found and indexed → do indexed pages get impressions → are impressions growing → does anyone click, and what do they do next. The first two take most of the effort at this stage; if indexing is stuck, rewriting titles will not help.",
      seoGrow: "Google treats the site as a normal site. Now the questions are which queries and pages can still grow, and what is slipping.",
      aiReach: "Can AI crawlers fetch your content at all? Until they can, nothing else on the AI side can improve.",
      aiKnown: "When someone asks an engine about you by name, does it know you and describe you correctly?",
      aiRecommend: "When a buyer asks for a tool like yours without naming it, are you in the answer, and which sources is the answer built from? Engines repeat what other sites say about you, so this stage is mostly work off your own site.",
      weekly: "One experiment a week. Read this week's numbers for your stage, pick one item, ship it, record it, and review the ones whose window has ended.",
    },
    avoids: {
      seoNew: "Average position, CTR, week-over-week percentages and GA4 rates. Samples are too small: a 20% drop can be three clicks, and average position falls whenever you start ranking for a new long-tail query. Read absolute numbers and individual pages instead.",
      seoGrow: "Site-wide average position and CTR averaged over queries. Compute CTR from totals, and compare a query with your own CTR at the same position, not with a generic table.",
      aiRecommend: "A single day's rate, and citation counts compared across engines. Answers change from run to run; read the interval, and only call a change up or down when the interval says so.",
    },
    stage: {
      title: "Which stage is your product in?",
      stage: "Stage",
      sign: "You are here if",
      weights: "Weight of SEO · AI",
      advice: "What to do in this tool",
      s0: { name: "S0 Cold start", sign: "Fewer than about 100 active users; retention has not flattened", weights: "SEO ★ · AI ★★", advice: "Do the one-time setup below and let monitoring run. Spend most of your week finding users by hand." },
      s1: { name: "S1 1→10", sign: "Looking for the first channel that brings people every week", weights: "SEO ★★★★ · AI ★★★", advice: "Follow the SEO and AI playbooks every week. SEO lags by months, so start before you need it." },
      s2: { name: "S2 10→100", sign: "The main channel is saturating; a second one must carry 30% or more", weights: "SEO ★★★★★ · AI ★★★★", advice: "Work the SEO growth stage and keep AI recommendations growing through off-site mentions." },
      s0Tip: "In S0, the weights in the book give SEO one star: it will not bring your first users in time. Communities, directories and direct outreach will. Read:",
      source: "Stages and weights come from",
    },
    books: {
      model: "Operations as a modelling problem",
      formula: "Traffic formula and stage weights",
      pmf: "PMF validation and first users",
      communities: "Finding communities",
      directories: "Directories and awesome lists",
      outreach: "Cold outreach",
      toolGrowth: "Growing a tool product overseas",
      toolData: "Reading data for a tool product",
      seoSetup: "SEO in practice",
      seoMonitor: "Monitoring SEO",
      seoNew: "SEO monitoring for a new site",
      ai: "Monitoring AI recommendations",
    },
    steps: {
      project: { do: "Create a project for your site.", why: "A project keeps one brand's site, questions, evidence and reports together; the first site check needs no model key.", done: "The first site check has finished." },
      google: { do: "Connect Google Search Console and GA4, and pick this project's properties.", why: "Indexing, impressions and clicks only exist in Google's data; set it up on day one so history starts early.", done: "A Search Console property is selected for this project." },
      engines: { do: "Add at least one AI engine key.", why: "Visibility is measured by asking engines your buyers' questions every day.", done: "A connection test passes for at least one engine." },
      seoLayers: { do: "Fix the site audit's Access, Discover and Understand layers, in that order.", why: "If crawlers cannot fetch, find or read a page, nothing later in the funnel can happen.", done: "All three layers pass." },
      seoSitemap: { do: "Put every page you want indexed in sitemap.xml, split by page type, and reference it from robots.txt.", why: "Separate sitemaps give you an index rate per page type, which is the main number at this stage.", done: "The audit shows no sitemap findings." },
      seoInspect: { do: "Let the indexing check go through every sitemap URL. Act only on alerts: a page dropped out of the index, Googlebot errors, or a sitemap's index rate falling more than {indexDrop} points in a day.", why: "Google's index report is days late and sampled; checking each URL tells you on the day.", done: "An indexing check succeeded in the last 7 days." },
      seoIndexWeek: { do: "Read the index rate per sitemap, how long this week's new pages took to be indexed, the 7-day index rate, and which pages are new under “Crawled - currently not indexed”.", why: "Indexing is where a new site gets stuck; a slow or falling index rate is a quality signal, not a technical one.", done: "You know this week's median indexing time and which section the unindexed pages belong to." },
      seoImprWeek: { do: "Read weekly impressions and the number of pages with impressions. Open the pages that got their first impressions this week and note their queries as leads.", why: "Impressions come weeks before clicks. Pages with impressions show where Google is starting to use the site.", done: "You have this week's impressions, pages with impressions, and a list of newly visible pages." },
      seoPeopleWeek: { do: "In GA4, read organic search sessions and key events as absolute numbers, then follow 3–5 individual visitors through the site.", why: "With small numbers, rates swing wildly; individual visits tell you what people came for.", done: "You have this week's organic sessions and key events, and notes on a few visits." },
      seoZeroMonth: { do: "List indexed pages with no impressions in 28 days and decide for each: improve it, merge it into another page, or noindex it.", why: "Indexed pages that never appear dilute how Google judges the whole site.", done: "Every zero-impression page has a decision." },
      seoTitlesMonth: { do: "Revisit pages whose title or description you changed a month ago: did impressions and position move?", why: "Title changes take days to weeks to show; checking a month later separates effect from noise.", done: "Each change from last month is marked kept or reverted." },
      growAlerts: { do: "Let alerts watch week-over-week drops; they fire only when the sample is large enough to mean something.", why: "Daily charts invite reactions to noise; a threshold with a sample floor does the watching for you.", done: "No open alert is older than a week." },
      growBrand: { do: "Split brand and non-brand queries, and judge SEO work by non-brand clicks.", why: "Brand clicks grow with everything else you do; only non-brand clicks show what search itself brings.", done: "You have this week's non-brand clicks and how they compare with four weeks ago." },
      growStriking: { do: "Work the queries at positions 8–20 that already have impressions; the action plan ranks them by opportunity clicks.", why: "Moving from page two to page one changes clicks far more than any change at position 30.", done: "One page-two query is in this week's experiment." },
      growDecline: { do: "Check pages whose clicks fell two weeks in a row. Compare the same days of the week with the same filters before acting.", why: "One bad week is usually noise; two in a row on comparable windows is worth a look.", done: "Each declining page has a cause or a note to wait." },
      growCtrMonth: { do: "Compare each query's CTR with your own CTR at the same position, and rewrite the titles and descriptions with the largest gaps.", why: "Your own curve reflects your result types and brand; a generic CTR table does not.", done: "The three largest gaps have a new title or a reason to leave them." },
      aiBots: { do: "Allow the AI crawlers you want in robots.txt (for example GPTBot, ClaudeBot, PerplexityBot) and check your CDN does not block them.", why: "Many CDNs and templates block AI user agents by default.", done: "The audit's Access layer passes." },
      aiLlms: { do: "Publish an llms.txt that lists your most important pages with one line each.", why: "It gives engines a short map of what to read first.", done: "The audit's Discover layer passes." },
      aiNoJs: { do: "Make sure each page's main text is in the HTML before JavaScript runs.", why: "Most AI crawlers do not run JavaScript; an empty shell looks like an empty page.", done: "No audited page is empty without JavaScript." },
      knownFacts: { do: "Write your one-sentence definition, aliases, target users, prices and key numbers with their sources.", why: "Engines answer branded questions from what they can find; give them one clear, consistent version.", done: "Definition, aliases and target users are filled in." },
      knownRecognition: { do: "Read recognition on branded questions with its interval.", why: "It shows whether engines know you at all, before asking whether they recommend you.", done: "Recognition has at least {recognitionN} answers this week." },
      knownRead: { do: "Read the answers to branded questions. Where an engine gets a fact wrong, find the page it relied on and fix that fact on your site or ask that site to correct it.", why: "Wrong descriptions spread: other answers and pages copy them.", done: "Every wrong fact you found has a fix in progress." },
      recQuestions: { do: "Write questions the way buyers ask them, by category and use case, and confirm the version before sampling.", why: "Changing a question changes what is measured; get the wording from real buyers first.", done: "The current question version is confirmed." },
      recCompetitors: { do: "List the competitors buyers actually compare you with.", why: "Share of voice is only meaningful against the products buyers weigh you against.", done: "At least three competitors are listed." },
      recVisibility: { do: "Read visibility and share of voice with their intervals, by engine and by question.", why: "One number per week hides which engine and which question moved.", done: "You know which questions moved this week, and whether the interval says they really did." },
      recCitations: { do: "Find the questions where competitors' sources are cited and yours are not, and list the directories, communities and review sites that keep coming up.", why: "Engines build answers from these sources; being on them is how you get into the answer.", done: "You have a short list of sources to get onto." },
      recGetListed: { do: "Get onto one source from that list each week: a directory listing, a useful community answer or a review. Record it as an experiment.", why: "Off-site mentions move AI recommendations; one a week adds up and stays measurable.", done: "This week's source is listed as an accepted action." },
      weekRead: { do: "Spend 30 minutes on the weekly steps of your current SEO and AI stages.", why: "A fixed weekly slot beats checking every day: small sites change slowly and daily numbers are mostly noise.", done: "You did the weekly steps of both playbooks." },
      weekPick: { do: "Pick one item from the action plan, accept it, and write the hypothesis, the metric and the observation window.", why: "Writing the hypothesis first is what makes the result readable later.", done: "One accepted action has a hypothesis, a metric and a window." },
      weekShip: { do: "When the change is live, record the real release date.", why: "The observation window starts at release, not when you accepted the item.", done: "The action has a release date." },
      weekReview: { do: "Evaluate the experiments whose window has ended: keep, extend or drop.", why: "Before-and-after comparisons need a finished window; reading them early is how false wins happen.", done: "No experiment with an ended window is left unevaluated." },
      weekReport: { do: "Generate the weekly report and add one line on what you changed this week.", why: "A written week is what lets you estimate, months later, which work paid off.", done: "This week has a report." },
    },
    signals: {
      newFast: { name: "New pages are indexed within days and start getting impressions", rule: "Median indexing time of new pages in the last 28 days is at most {newPageDays} days, and they have impressions" },
      sitemapStable: { name: "Index rate of each sitemap stays high and steady", rule: "At least {sitemapRate}% for {weeks} weeks in a row, moving no more than {sitemapSwing} points between weeks" },
      discoveredDown: { name: "“Discovered - currently not indexed” keeps shrinking", rule: "Read it in Search Console's page indexing report; the API does not provide it" },
      crawlUp: { name: "Googlebot crawls clearly more per day than at launch", rule: "Read it in Search Console's crawl stats; the API does not provide it" },
      imprCoverage: { name: "Most indexed pages get impressions", rule: "Pages with impressions in 28 days are at least {imprCoverage}% of indexed pages" },
      visibleShare: { name: "The gap between total clicks and query-table clicks is closing", rule: "Clicks on visible queries rise {weeks} weeks in a row or reach {visibleShare}% of total clicks" },
      brandFilter: { name: "Search Console offers a brand query filter", rule: "Check the performance report's filters" },
      gaThreshold: { name: "GA4 no longer applies thresholds often", rule: "No GA4 report in the last {weeks} weeks was thresholded" },
      accessPass: { name: "The audit's Access layer passes", rule: "Latest audit has no critical or warning finding in Access" },
      discoverPass: { name: "The audit's Discover layer passes", rule: "Latest audit has no critical or warning finding in Discover" },
      recognized: { name: "Engines know you", rule: "Recognition on branded questions has at least {recognitionN} answers and the lower end of its 95% interval is at least {recognitionLower}%" },
    },
  },
```

- [ ] **Step 2: Add the Chinese block to `zh.ts` after `helpCenter: { ... },`**

```ts
  playbook: {
    ask: "这个阶段在问什么",
    avoid: "这个阶段先别看",
    why: "为什么",
    done: "做完的标志",
    open: "在哪做",
    pass: "什么时候算过了这一关",
    signal: "信号",
    rule: "怎么判断",
    judged: "判定",
    auto: "自动",
    manual: "人工确认",
    passNote: "这些阈值是我们的先验判断，不是 Google 或 AI 的规则。之后会用真实项目校准，再写回书里。",
    bookNote: "每一步都标了出自 craftsail-book 的哪一章。",
    cadence: { once: "一次", daily: "每天（自动）", weekly: "每周 30 分钟", monthly: "每月" },
    asks: {
      start: "先别看数：你的产品在哪个阶段？SEO 和 AI 推荐都是慢渠道，值得花多少时间取决于阶段。先判断阶段，再做一次性设置。",
      seoNew: "Google 认不认这个站？拆成四个问题，按顺序：新页能不能被发现和收录 → 收录的页有没有展示 → 展示在不在涨 → 有没有人点、点完做了什么。这一阶段八成精力在前两个；收录卡住，改标题也没用。",
      seoGrow: "Google 已经把这个站当成正常的站。现在的问题是：哪些词和页面还能再涨，哪些在往下掉。",
      aiReach: "AI 的爬虫能不能拿到你的内容？拿不到，AI 这一侧做什么都没用。",
      aiKnown: "有人直接点名问 AI 你的产品时，它认不认识你，说得对不对？",
      aiRecommend: "买家不点名、只问“有什么好用的 xxx 工具”时，回答里有没有你，回答的依据是哪些网页？AI 复述的是别的网站怎么说你，所以这一阶段的活大多在你自己的站外。",
      weekly: "每周一个实验：按你所在的阶段看本周的数，挑一件事，上线并记录，再复查观察期已到的。",
    },
    avoids: {
      seoNew: "平均排名、CTR、周环比百分比、GA4 的各种率。样本太小：周环比 −20% 可能只是少了 3 个点击；开始排上新的长尾词时，平均排名反而会变差。看绝对数，一页一页地看。",
      seoGrow: "全站平均排名，以及按查询词平均出来的 CTR。CTR 要用合计算；拿一个词和你自己同排名的 CTR 比，不要套通用表。",
      aiRecommend: "单日的比率，以及跨引擎比较引用数。回答每次都不一样；看区间，只有区间说变了才算变。",
    },
    stage: {
      title: "你的产品在哪个阶段？",
      stage: "阶段",
      sign: "判断标志",
      weights: "SEO · AI 的权重",
      advice: "在这个工具里做什么",
      s0: { name: "S0 冷启动", sign: "活跃用户不到 100 个左右，留存曲线还没走平", weights: "SEO ★ · AI ★★", advice: "做完下面的一次性设置，让监测自动跑。一周的大部分时间去手动找用户。" },
      s1: { name: "S1 1→10", sign: "在找第一条每周稳定来人的渠道", weights: "SEO ★★★★ · AI ★★★", advice: "每周按 SEO 和 AI 剧本做。SEO 滞后几个月，要在需要它之前就开始。" },
      s2: { name: "S2 10→100", sign: "主渠道开始饱和，第二条渠道要撑起 30% 以上", weights: "SEO ★★★★★ · AI ★★★★", advice: "按 SEO 起量阶段做，并通过站外提及继续推高 AI 推荐。" },
      s0Tip: "S0 阶段，书里的权重表给 SEO 只有一颗星：它来不及带来第一批用户，社区、目录站和主动联系才会。去读：",
      source: "阶段和权重来自",
    },
    books: {
      model: "把运营当成一道建模题",
      formula: "流量公式与阶段权重",
      pmf: "PMF 验证与第一批用户",
      communities: "怎么找社区",
      directories: "目录站与 awesome 列表",
      outreach: "冷外联",
      toolGrowth: "工具产品出海运营",
      toolData: "工具产品看数据",
      seoSetup: "SEO 落地",
      seoMonitor: "SEO 的监测",
      seoNew: "新站的 SEO 监测",
      ai: "AI 推荐的监测",
    },
    steps: {
      project: { do: "为你的网站创建项目。", why: "一个项目保存一个品牌的网站、问题、证据和报告；首次站点检查不需要模型密钥。", done: "首次站点检查已完成。" },
      google: { do: "连接 Google Search Console 和 GA4，为本项目选好资源。", why: "收录、展示和点击只在 Google 的数据里；第一天就接上，历史才会尽早开始积累。", done: "本项目已选定 Search Console 资源。" },
      engines: { do: "至少添加一个 AI 引擎的密钥。", why: "可见性是每天拿买家的问题去问引擎测出来的。", done: "至少一个引擎的连接测试通过。" },
      seoLayers: { do: "按顺序修好站点体检的访问、发现、理解三层。", why: "爬虫抓不到、找不到、读不懂，漏斗后面的事都不会发生。", done: "三层都通过。" },
      seoSitemap: { do: "把每个要收录的页面放进 sitemap.xml，按页面类型拆开，并在 robots.txt 里声明。", why: "拆开的 sitemap 才能按页面类型算索引率，这是这一阶段最主要的数。", done: "体检里没有 sitemap 相关问题。" },
      seoInspect: { do: "让收录检查每天把 sitemap 里的 URL 都查一遍。只在提醒时处理：有页面掉出索引、Googlebot 报错、某个 sitemap 的索引率一天掉了 {indexDrop} 个点以上。", why: "Google 的索引报告晚几天、还是抽样的；逐个 URL 检查，当天就知道。", done: "最近 7 天内有一次成功的收录检查。" },
      seoIndexWeek: { do: "看各 sitemap 的索引率、本周新页的收录时长、7 天收录率，以及「已抓取 - 尚未编入索引」里新增了哪些页。", why: "新站最容易卡在收录；索引率慢或在掉，多半是质量信号，不是技术问题。", done: "你知道本周收录时长的中位数，也知道未收录的页属于哪一块。" },
      seoImprWeek: { do: "看周展示和有展示的网页数。打开本周第一次出现展示的页面，把它们的查询词记下来当线索。", why: "展示比点击早几周出现。有展示的页面说明 Google 开始在哪里用这个站。", done: "你有本周的展示数、有展示的网页数和新出现展示的页面清单。" },
      seoPeopleWeek: { do: "在 GA4 里看自然搜索会话和关键事件的绝对数，再挑 3–5 个访客，看他们在站内的路径。", why: "数小的时候，率会大起大落；一个人一个人地看，才知道他们来找什么。", done: "你有本周的自然搜索会话和关键事件数，以及几位访客的记录。" },
      seoZeroMonth: { do: "列出已收录但 28 天零展示的页面，逐个决定：改、合并到别的页，还是 noindex。", why: "收录了却从不出现的页，会拉低 Google 对全站的判断。", done: "每个零展示页都有了决定。" },
      seoTitlesMonth: { do: "回看一个月前改过标题或描述的页面：展示和排名有没有变化。", why: "标题改动要几天到几周才生效；隔一个月再看，才分得清效果和噪声。", done: "上个月的每处改动都标了保留或撤回。" },
      growAlerts: { do: "让告警盯周环比下跌；样本够大才会触发。", why: "天天看图会对噪声做出反应；带样本门槛的告警线替你盯着。", done: "没有超过一周未处理的告警。" },
      growBrand: { do: "把品牌词和非品牌词拆开，用非品牌点击来评价 SEO。", why: "品牌词点击会随你做的所有事一起涨；只有非品牌点击才是搜索本身带来的。", done: "你有本周的非品牌点击，以及和 4 周前的对比。" },
      growStriking: { do: "处理排名 8–20、已经有展示的词；行动清单按机会点击给它们排了序。", why: "从第二页进到第一页，点击的变化远大于第 30 名附近的任何变化。", done: "本周的实验里有一个第二页的词。" },
      growDecline: { do: "检查点击连续两周下降的页面。先用相同的星期几、相同的筛选条件对比，再动手。", why: "一周不好通常是噪声；口径一致的连续两周才值得看。", done: "每个下滑页都有原因，或注明继续观察。" },
      growCtrMonth: { do: "拿每个词的 CTR 和你自己同排名的 CTR 比，改差距最大的几页的标题和描述。", why: "你自己的曲线反映了你的结果类型和品牌，通用 CTR 表反映不了。", done: "差距最大的三个都有了新标题，或写明不改的理由。" },
      aiBots: { do: "在 robots.txt 里放行你想要的 AI 爬虫（如 GPTBot、ClaudeBot、PerplexityBot），并确认 CDN 没有拦它们。", why: "很多 CDN 和模板默认就拦 AI 的 user agent。", done: "体检的访问层通过。" },
      aiLlms: { do: "发布 llms.txt，列出最重要的页面，每页一句话。", why: "它给 AI 一张简短的地图，告诉它先读什么。", done: "体检的发现层通过。" },
      aiNoJs: { do: "确认每个页面的正文在 JavaScript 运行前就在 HTML 里。", why: "大部分 AI 爬虫不执行 JavaScript；空壳页在它们眼里就是空页。", done: "体检的页面里没有一个在不执行 JavaScript 时是空的。" },
      knownFacts: { do: "写下一句话定义、别名、目标用户、价格和带出处的关键数字。", why: "AI 回答点名问题，靠的是它能找到的说法；给它一个清楚、一致的版本。", done: "定义、别名和目标用户都已填写。" },
      knownRecognition: { do: "看品牌问题上的认知度和它的区间。", why: "先看 AI 认不认识你，再问它推不推荐你。", done: "本周认知度至少有 {recognitionN} 个回答。" },
      knownRead: { do: "读品牌问题的回答。AI 说错的事实，找到它依据的网页，在你的站上改，或者请那个网站更正。", why: "错误的描述会扩散：别的回答和网页会照抄。", done: "找到的每个错误事实都在修正中。" },
      recQuestions: { do: "按买家的原话写问题，按品类和使用场景分，采样前确认版本。", why: "改问题就是改测量对象；先从真实买家那里拿到措辞。", done: "当前题库版本已确认。" },
      recCompetitors: { do: "列出买家真的会拿来和你比的竞品。", why: "只有和买家真正比较的产品放在一起，声量才有意义。", done: "至少列了三个竞品。" },
      recVisibility: { do: "按引擎、按问题看可见性和声量，连同区间一起看。", why: "每周一个总数，看不出是哪个引擎、哪个问题在变。", done: "你知道本周哪些问题变了，以及区间是否说明它真的变了。" },
      recCitations: { do: "找出引用了竞品来源、却没引用你的问题，列出反复出现的目录站、社区和测评站。", why: "AI 用这些来源拼出回答；出现在它们上面，才进得了回答。", done: "你有一份要去争取的来源短名单。" },
      recGetListed: { do: "每周从名单里拿下一个来源：一个目录收录、一条有用的社区回答或一篇测评。记成一个实验。", why: "站外提及会推动 AI 推荐；每周一个，积少成多，也能量出效果。", done: "本周的来源已作为一条已接受的行动记下。" },
      weekRead: { do: "花 30 分钟，按你当前 SEO 和 AI 阶段的每周步骤过一遍。", why: "固定每周一次，比天天看强：小站变化慢，日数据大多是噪声。", done: "两份剧本的每周步骤都做完了。" },
      weekPick: { do: "从行动清单里挑一条，接受它，写下假设、指标和观察窗口。", why: "先写假设，之后才读得懂结果。", done: "有一条已接受的行动写好了假设、指标和窗口。" },
      weekShip: { do: "改动上线后，记下真实的上线日期。", why: "观察窗口从上线开始算，不是从接受那天。", done: "这条行动有了上线日期。" },
      weekReview: { do: "评估观察期已结束的实验：保留、延长还是放弃。", why: "前后对比要等窗口走完；提前看，最容易看出假的胜利。", done: "没有窗口已结束却未评估的实验。" },
      weekReport: { do: "生成周报，写一句本周改了什么。", why: "每周写下来，几个月后才估得出哪些事有回报。", done: "本周有一份报告。" },
    },
    signals: {
      newFast: { name: "新页几天内收录，并开始有展示", rule: "近 28 天新页收录时长中位数不超过 {newPageDays} 天，且有展示" },
      sitemapStable: { name: "各 sitemap 的索引率稳定在高位", rule: "连续 {weeks} 周不低于 {sitemapRate}%，周间波动不超过 {sitemapSwing} 个点" },
      discoveredDown: { name: "「已发现 - 尚未编入索引」持续减少", rule: "在 Search Console 的网页索引报告里看；API 不提供" },
      crawlUp: { name: "每天的抓取请求明显高于刚上线时", rule: "在 Search Console 的抓取统计里看；API 不提供" },
      imprCoverage: { name: "有展示的网页占已收录的大多数", rule: "28 天内有展示的网页不少于已收录网页的 {imprCoverage}%" },
      visibleShare: { name: "总点击和查询表点击之和的差距在缩小", rule: "可见查询的点击占比连续 {weeks} 周上升，或达到总点击的 {visibleShare}%" },
      brandFilter: { name: "GSC 里出现了品牌词过滤", rule: "在效果报告的筛选条件里看" },
      gaThreshold: { name: "GA4 不再频繁出现阈值提示", rule: "近 {weeks} 周没有 GA4 报表被阈值处理" },
      accessPass: { name: "体检的访问层通过", rule: "最近一次体检，访问层没有严重或警告问题" },
      discoverPass: { name: "体检的发现层通过", rule: "最近一次体检，发现层没有严重或警告问题" },
      recognized: { name: "AI 认识你", rule: "品牌问题上的认知度至少有 {recognitionN} 个回答，95% 区间下界不低于 {recognitionLower}%" },
    },
  },
```

- [ ] **Step 3: Add the Portuguese block to `pt.ts` after `helpCenter: { ... },`**

```ts
  playbook: {
    ask: "O que este estágio pergunta",
    avoid: "Não leia isto ainda",
    why: "Por quê",
    done: "Pronto quando",
    open: "Onde",
    pass: "Quando você passou deste estágio",
    signal: "Sinal",
    rule: "Como é avaliado",
    judged: "Verificação",
    auto: "Automática",
    manual: "Manual",
    passNote: "Estes limites são nossas hipóteses iniciais, não regras do Google ou dos mecanismos. Vamos ajustá-los com projetos reais e registrar o resultado no livro.",
    bookNote: "Cada passo cita o capítulo do craftsail-book de onde vem (em chinês).",
    cadence: { once: "Uma vez", daily: "Todo dia (automático)", weekly: "Toda semana, 30 minutos", monthly: "Todo mês" },
    asks: {
      start: "Antes dos números: em que estágio está o seu produto? SEO e recomendações de IA são canais lentos. O tempo que merecem depende do estágio; decida isso primeiro e depois faça a configuração inicial.",
      seoNew: "O Google aceita este site? Quatro perguntas, em ordem: as páginas novas são encontradas e indexadas → as páginas indexadas têm impressões → as impressões estão crescendo → alguém clica e o que faz depois. As duas primeiras levam quase todo o esforço nesta fase; se a indexação travou, reescrever títulos não ajuda.",
      seoGrow: "O Google já trata o site como um site normal. Agora as perguntas são quais consultas e páginas ainda podem crescer e o que está caindo.",
      aiReach: "Os rastreadores de IA conseguem buscar o seu conteúdo? Enquanto não conseguirem, nada mais do lado da IA melhora.",
      aiKnown: "Quando alguém pergunta a um mecanismo sobre você pelo nome, ele conhece o produto e o descreve corretamente?",
      aiRecommend: "Quando um comprador pede uma ferramenta como a sua sem citar o nome, você aparece na resposta, e em quais fontes ela se baseia? Os mecanismos repetem o que outros sites dizem sobre você, então este estágio é sobretudo trabalho fora do seu site.",
      weekly: "Um experimento por semana. Leia os números da semana do seu estágio, escolha um item, publique, registre e revise os que já completaram a janela.",
    },
    avoids: {
      seoNew: "Posição média, CTR, variações semanais em porcentagem e taxas do GA4. As amostras são pequenas demais: uma queda de 20% pode ser três cliques, e a posição média piora sempre que você começa a aparecer para uma nova consulta de cauda longa. Leia números absolutos e páginas individuais.",
      seoGrow: "Posição média do site e CTR calculado como média entre consultas. Calcule o CTR a partir dos totais e compare uma consulta com o seu próprio CTR na mesma posição, não com uma tabela genérica.",
      aiRecommend: "A taxa de um único dia e contagens de citações comparadas entre mecanismos. As respostas mudam a cada execução; leia o intervalo e só considere uma alta ou queda quando o intervalo indicar.",
    },
    stage: {
      title: "Em que estágio está o seu produto?",
      stage: "Estágio",
      sign: "Você está aqui se",
      weights: "Peso de SEO · IA",
      advice: "O que fazer nesta ferramenta",
      s0: { name: "S0 Partida a frio", sign: "Menos de cerca de 100 usuários ativos; a retenção ainda não se estabilizou", weights: "SEO ★ · IA ★★", advice: "Faça a configuração inicial abaixo e deixe o monitoramento rodar. Passe a maior parte da semana encontrando usuários manualmente." },
      s1: { name: "S1 1→10", sign: "Procurando o primeiro canal que traz pessoas toda semana", weights: "SEO ★★★★ · IA ★★★", advice: "Siga os roteiros de SEO e IA toda semana. O SEO tem atraso de meses; comece antes de precisar dele." },
      s2: { name: "S2 10→100", sign: "O canal principal está saturando; um segundo precisa sustentar 30% ou mais", weights: "SEO ★★★★★ · IA ★★★★", advice: "Trabalhe o estágio de crescimento de SEO e continue aumentando as recomendações de IA com menções fora do site." },
      s0Tip: "No S0, a tabela de pesos do livro dá uma estrela ao SEO: ele não trará os primeiros usuários a tempo. Comunidades, diretórios e contato direto trarão. Leia:",
      source: "Estágios e pesos vêm de",
    },
    books: {
      model: "Operação como um problema de modelagem",
      formula: "Fórmula de tráfego e pesos por estágio",
      pmf: "Validação de PMF e primeiros usuários",
      communities: "Como encontrar comunidades",
      directories: "Diretórios e listas awesome",
      outreach: "Contato frio",
      toolGrowth: "Crescimento internacional de ferramentas",
      toolData: "Dados para produtos de ferramenta",
      seoSetup: "SEO na prática",
      seoMonitor: "Monitoramento de SEO",
      seoNew: "Monitoramento de SEO para site novo",
      ai: "Monitoramento de recomendações de IA",
    },
    steps: {
      project: { do: "Crie um projeto para o seu site.", why: "Um projeto reúne o site, as perguntas, as evidências e os relatórios de uma marca; a primeira verificação do site não precisa de chave de modelo.", done: "A primeira verificação do site terminou." },
      google: { do: "Conecte o Google Search Console e o GA4 e escolha as propriedades deste projeto.", why: "Indexação, impressões e cliques só existem nos dados do Google; configure no primeiro dia para o histórico começar cedo.", done: "Uma propriedade do Search Console está selecionada para este projeto." },
      engines: { do: "Adicione a chave de pelo menos um mecanismo de IA.", why: "A visibilidade é medida perguntando aos mecanismos, todos os dias, as perguntas dos seus compradores.", done: "O teste de conexão passa em pelo menos um mecanismo." },
      seoLayers: { do: "Corrija as camadas Acesso, Descoberta e Compreensão da auditoria, nessa ordem.", why: "Se os rastreadores não conseguem buscar, encontrar ou ler uma página, nada depois no funil acontece.", done: "As três camadas passam." },
      seoSitemap: { do: "Coloque todas as páginas que devem ser indexadas no sitemap.xml, separado por tipo de página, e referencie-o no robots.txt.", why: "Sitemaps separados dão uma taxa de indexação por tipo de página, o número principal neste estágio.", done: "A auditoria não mostra problemas de sitemap." },
      seoInspect: { do: "Deixe a verificação de indexação passar por todas as URLs do sitemap. Aja só com alertas: uma página saiu do índice, erros do Googlebot ou a taxa de indexação de um sitemap caiu mais de {indexDrop} pontos em um dia.", why: "O relatório de indexação do Google atrasa dias e é amostrado; verificar cada URL avisa no mesmo dia.", done: "Uma verificação de indexação teve sucesso nos últimos 7 dias." },
      seoIndexWeek: { do: "Leia a taxa de indexação por sitemap, quanto tempo as páginas novas da semana levaram para ser indexadas, a taxa de indexação em 7 dias e quais páginas são novas em “Rastreada, mas não indexada no momento”.", why: "A indexação é onde um site novo trava; uma taxa lenta ou em queda é sinal de qualidade, não técnico.", done: "Você sabe a mediana do tempo de indexação da semana e a qual seção pertencem as páginas não indexadas." },
      seoImprWeek: { do: "Leia as impressões da semana e o número de páginas com impressões. Abra as páginas que tiveram as primeiras impressões nesta semana e anote as consultas como pistas.", why: "As impressões chegam semanas antes dos cliques. Páginas com impressões mostram onde o Google começa a usar o site.", done: "Você tem as impressões da semana, as páginas com impressões e a lista de páginas que passaram a aparecer." },
      seoPeopleWeek: { do: "No GA4, leia sessões de busca orgânica e eventos-chave como números absolutos e siga 3–5 visitantes pelo site.", why: "Com números pequenos, as taxas oscilam muito; visitas individuais mostram o que as pessoas vieram buscar.", done: "Você tem as sessões orgânicas e os eventos-chave da semana e anotações sobre algumas visitas." },
      seoZeroMonth: { do: "Liste as páginas indexadas sem impressões em 28 dias e decida para cada uma: melhorar, mesclar em outra página ou aplicar noindex.", why: "Páginas indexadas que nunca aparecem diluem como o Google avalia o site inteiro.", done: "Cada página sem impressões tem uma decisão." },
      seoTitlesMonth: { do: "Revise as páginas cujo título ou descrição você mudou há um mês: impressões e posição mudaram?", why: "Mudanças de título levam de dias a semanas para aparecer; verificar um mês depois separa efeito de ruído.", done: "Cada mudança do mês passado está marcada como mantida ou revertida." },
      growAlerts: { do: "Deixe os alertas acompanharem quedas semanais; eles só disparam quando a amostra é grande o bastante para significar algo.", why: "Gráficos diários levam a reagir a ruído; um limite com amostra mínima vigia por você.", done: "Nenhum alerta aberto tem mais de uma semana." },
      growBrand: { do: "Separe consultas de marca e sem marca e avalie o trabalho de SEO pelos cliques sem marca.", why: "Cliques de marca crescem com tudo o que você faz; só os cliques sem marca mostram o que a busca traz.", done: "Você tem os cliques sem marca da semana e a comparação com quatro semanas antes." },
      growStriking: { do: "Trabalhe as consultas nas posições 8 a 20 que já têm impressões; o plano de ação as ordena por cliques de oportunidade.", why: "Passar da segunda para a primeira página muda os cliques muito mais do que qualquer mudança na posição 30.", done: "Uma consulta da segunda página está no experimento da semana." },
      growDecline: { do: "Verifique páginas cujos cliques caíram duas semanas seguidas. Compare os mesmos dias da semana com os mesmos filtros antes de agir.", why: "Uma semana ruim costuma ser ruído; duas seguidas em janelas comparáveis merecem atenção.", done: "Cada página em queda tem uma causa ou uma nota para esperar." },
      growCtrMonth: { do: "Compare o CTR de cada consulta com o seu próprio CTR na mesma posição e reescreva títulos e descrições com as maiores diferenças.", why: "A sua curva reflete seus tipos de resultado e sua marca; uma tabela genérica de CTR não.", done: "As três maiores diferenças têm um novo título ou um motivo para ficar como estão." },
      aiBots: { do: "Permita no robots.txt os rastreadores de IA que você quer (por exemplo GPTBot, ClaudeBot, PerplexityBot) e confirme que a CDN não os bloqueia.", why: "Muitas CDNs e modelos bloqueiam agentes de IA por padrão.", done: "A camada Acesso da auditoria passa." },
      aiLlms: { do: "Publique um llms.txt com as páginas mais importantes, uma linha para cada.", why: "Ele dá aos mecanismos um mapa curto do que ler primeiro.", done: "A camada Descoberta da auditoria passa." },
      aiNoJs: { do: "Garanta que o texto principal de cada página esteja no HTML antes de o JavaScript rodar.", why: "A maioria dos rastreadores de IA não executa JavaScript; uma casca vazia parece uma página vazia.", done: "Nenhuma página auditada fica vazia sem JavaScript." },
      knownFacts: { do: "Escreva a definição em uma frase, apelidos, usuários-alvo, preços e números-chave com as fontes.", why: "Os mecanismos respondem perguntas com a marca a partir do que encontram; dê a eles uma versão clara e consistente.", done: "Definição, apelidos e usuários-alvo estão preenchidos." },
      knownRecognition: { do: "Leia o reconhecimento nas perguntas com a marca e o intervalo.", why: "Mostra se os mecanismos conhecem você antes de perguntar se recomendam.", done: "O reconhecimento tem pelo menos {recognitionN} respostas nesta semana." },
      knownRead: { do: "Leia as respostas às perguntas com a marca. Quando um mecanismo erra um fato, encontre a página em que se baseou e corrija no seu site ou peça a esse site para corrigir.", why: "Descrições erradas se espalham: outras respostas e páginas as copiam.", done: "Cada fato errado encontrado tem uma correção em andamento." },
      recQuestions: { do: "Escreva as perguntas como os compradores perguntam, por categoria e caso de uso, e confirme a versão antes da amostragem.", why: "Mudar uma pergunta muda o que é medido; pegue o texto de compradores reais primeiro.", done: "A versão atual das perguntas está confirmada." },
      recCompetitors: { do: "Liste os concorrentes com que os compradores realmente comparam você.", why: "A participação de voz só faz sentido contra os produtos que os compradores consideram.", done: "Pelo menos três concorrentes estão listados." },
      recVisibility: { do: "Leia visibilidade e participação de voz com os intervalos, por mecanismo e por pergunta.", why: "Um número por semana esconde qual mecanismo e qual pergunta mudaram.", done: "Você sabe quais perguntas mudaram na semana e se o intervalo confirma a mudança." },
      recCitations: { do: "Encontre as perguntas em que as fontes dos concorrentes são citadas e as suas não, e liste os diretórios, comunidades e sites de avaliação que se repetem.", why: "Os mecanismos montam as respostas a partir dessas fontes; estar nelas é como você entra na resposta.", done: "Você tem uma lista curta de fontes para conquistar." },
      recGetListed: { do: "Conquiste uma fonte da lista por semana: um diretório, uma resposta útil numa comunidade ou uma avaliação. Registre como experimento.", why: "Menções fora do site movem as recomendações de IA; uma por semana se acumula e continua mensurável.", done: "A fonte da semana está registrada como ação aceita." },
      weekRead: { do: "Dedique 30 minutos aos passos semanais dos seus estágios atuais de SEO e IA.", why: "Um horário fixo por semana é melhor do que olhar todo dia: sites pequenos mudam devagar e números diários são quase só ruído.", done: "Você fez os passos semanais dos dois roteiros." },
      weekPick: { do: "Escolha um item do plano de ação, aceite e escreva a hipótese, a métrica e a janela de observação.", why: "Escrever a hipótese antes é o que torna o resultado legível depois.", done: "Uma ação aceita tem hipótese, métrica e janela." },
      weekShip: { do: "Quando a mudança estiver no ar, registre a data real de lançamento.", why: "A janela de observação começa no lançamento, não quando você aceitou o item.", done: "A ação tem uma data de lançamento." },
      weekReview: { do: "Avalie os experimentos cuja janela terminou: manter, estender ou descartar.", why: "Comparações de antes e depois precisam da janela completa; ler cedo é como surgem vitórias falsas.", done: "Nenhum experimento com janela encerrada ficou sem avaliação." },
      weekReport: { do: "Gere o relatório semanal e acrescente uma linha sobre o que mudou nesta semana.", why: "Uma semana registrada é o que permite estimar, meses depois, qual trabalho valeu a pena.", done: "Esta semana tem um relatório." },
    },
    signals: {
      newFast: { name: "Páginas novas são indexadas em poucos dias e começam a ter impressões", rule: "A mediana do tempo de indexação das páginas novas nos últimos 28 dias é de no máximo {newPageDays} dias, e elas têm impressões" },
      sitemapStable: { name: "A taxa de indexação de cada sitemap fica alta e estável", rule: "Pelo menos {sitemapRate}% por {weeks} semanas seguidas, variando no máximo {sitemapSwing} pontos entre semanas" },
      discoveredDown: { name: "“Descoberta, mas não indexada no momento” continua diminuindo", rule: "Veja no relatório de indexação de páginas do Search Console; a API não fornece" },
      crawlUp: { name: "O Googlebot rastreia claramente mais por dia do que no lançamento", rule: "Veja nas estatísticas de rastreamento do Search Console; a API não fornece" },
      imprCoverage: { name: "A maioria das páginas indexadas tem impressões", rule: "Páginas com impressões em 28 dias são pelo menos {imprCoverage}% das indexadas" },
      visibleShare: { name: "A diferença entre cliques totais e cliques da tabela de consultas está diminuindo", rule: "Os cliques em consultas visíveis sobem {weeks} semanas seguidas ou chegam a {visibleShare}% do total" },
      brandFilter: { name: "O Search Console oferece o filtro de consultas de marca", rule: "Veja os filtros do relatório de desempenho" },
      gaThreshold: { name: "O GA4 não aplica mais limites com frequência", rule: "Nenhum relatório do GA4 nas últimas {weeks} semanas teve limite aplicado" },
      accessPass: { name: "A camada Acesso da auditoria passa", rule: "A última auditoria não tem problemas críticos nem avisos em Acesso" },
      discoverPass: { name: "A camada Descoberta da auditoria passa", rule: "A última auditoria não tem problemas críticos nem avisos em Descoberta" },
      recognized: { name: "Os mecanismos conhecem você", rule: "O reconhecimento nas perguntas com marca tem pelo menos {recognitionN} respostas e o limite inferior do intervalo de 95% é de pelo menos {recognitionLower}%" },
    },
  },
```

- [ ] **Step 4: Type-check**

Run: `cd web && ./node_modules/.bin/tsc --noEmit`
Expected: errors only in `catalog.ts` about `PLAYBOOK_TOPICS ... satisfies readonly TopicId[]` (the new topic ids do not exist until Task 4). No errors about missing `playbook.*` keys or zh/pt key mismatches. If `zh.ts` or `pt.ts` report a missing or extra key, fix the block so the three match en exactly.

- [ ] **Step 5: Checkpoint**

Run: `git status --short web/src/i18n`
Expected: three locale files modified.

---

### Task 3: Playbook renderer

**Files:**
- Create: `web/src/features/help/playbook.tsx`

- [ ] **Step 1: Create the renderer**

```tsx
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ReactNode } from "react";
import { Table, Tip, Warn, type Kit, type TopicId } from "./kit";
import {
  AVOID_TOPICS, BOOK_BASE, BOOKS, CADENCES, S0_BOOKS, SIGNALS, STEPS, THRESHOLDS,
  type BookRef, type PlaybookTopic,
} from "../playbook/catalog";

function BookLinks({ refs, t }: { refs: readonly BookRef[]; t: Kit["t"] }) {
  return <>{refs.map((r, i) => (
    <span key={r.book + (r.section || "")}>
      {i > 0 && " · "}
      <a className="text-primary-600 hover:underline" href={BOOK_BASE + BOOKS[r.book]} target="_blank" rel="noreferrer">
        {t(`playbook.books.${r.book}`)}{r.section ? ` §${r.section}` : ""}
      </a>
    </span>
  ))}</>;
}

// Reference topics each playbook links to at the end.
const RELATED: Record<PlaybookTopic, TopicId[]> = {
  start: ["terms", "google", "providers"],
  seoNew: ["audit", "google", "troubleshooting"],
  seoGrow: ["numbers", "troubleshooting"],
  aiReach: ["audit"],
  aiKnown: ["numbers", "manual"],
  aiRecommend: ["prompts", "numbers", "manual"],
  weekly: ["numbers", "troubleshooting"],
};

// playbookBodies renders every playbook topic from the step catalog:
// what the stage asks, what not to read yet, steps by cadence, and the
// signals that say the stage is passed.
export function playbookBodies(k: Kit): Record<PlaybookTopic, ReactNode> {
  const { t, page, topic } = k;
  const steps = (id: PlaybookTopic) => {
    const list = STEPS.filter((s) => s.topic === id);
    return CADENCES.filter((c) => list.some((s) => s.cadence === c)).map((c) => (
      <div key={c}>
        <h3>{t(`playbook.cadence.${c}`)}</h3>
        <ol>{list.filter((s) => s.cadence === c).map((s) => (
          <li key={s.id}>
            <strong>{t(`playbook.steps.${s.id}.do`, THRESHOLDS)}</strong>
            <div>{t("playbook.why")}: {t(`playbook.steps.${s.id}.why`, THRESHOLDS)} (<BookLinks refs={s.books} t={t} />)</div>
            <div>{t("playbook.done")}: {t(`playbook.steps.${s.id}.done`, THRESHOLDS)}</div>
            <div>{t("playbook.open")}: {page(s.page.path, s.page.label)}</div>
          </li>
        ))}</ol>
      </div>
    ));
  };
  const pass = (id: PlaybookTopic) => {
    const list = SIGNALS.filter((s) => s.topic === id);
    if (!list.length) return null;
    return <>
      <h3>{t("playbook.pass")}</h3>
      <Table head={[t("playbook.signal"), t("playbook.rule"), t("playbook.judged")]} rows={list.map((s) => [
        t(`playbook.signals.${s.id}.name`), t(`playbook.signals.${s.id}.rule`, THRESHOLDS), s.auto ? t("playbook.auto") : t("playbook.manual"),
      ])} />
      <p>{t("playbook.passNote")}</p>
    </>;
  };
  const avoid = (id: PlaybookTopic) => {
    const a = AVOID_TOPICS.find((x) => x === id);
    return a ? <Warn><strong>{t("playbook.avoid")}.</strong> {t(`playbook.avoids.${a}`)}</Warn> : null;
  };
  const related = (id: PlaybookTopic) => (
    <p>{t("helpUsage.related")}{": "}{RELATED[id].map((r, i) => <span key={r}>{i > 0 && " · "}{topic(r, t(`helpTopics.${r}.label`))}</span>)}</p>
  );
  const body = (id: PlaybookTopic, extra?: ReactNode) => <>
    <h2>{t(`helpTopics.${id}.label`)}</h2>
    <p><strong>{t("playbook.ask")}.</strong> {t(`playbook.asks.${id}`)}</p>
    {extra}
    {avoid(id)}
    {steps(id)}
    {pass(id)}
    <p className="text-gray-500">{t("playbook.bookNote")}</p>
    {related(id)}
  </>;
  const stages = <>
    <h3>{t("playbook.stage.title")}</h3>
    <Table head={[t("playbook.stage.stage"), t("playbook.stage.sign"), t("playbook.stage.weights"), t("playbook.stage.advice")]}
      rows={(["s0", "s1", "s2"] as const).map((s) => [
        t(`playbook.stage.${s}.name`), t(`playbook.stage.${s}.sign`), t(`playbook.stage.${s}.weights`), t(`playbook.stage.${s}.advice`),
      ])} />
    <Tip>{t("playbook.stage.s0Tip")} <BookLinks refs={S0_BOOKS} t={t} /></Tip>
    <p>{t("playbook.stage.source")} <BookLinks refs={[{ book: "formula", section: "4" }]} t={t} />.</p>
    <Tip>{t("helpUsage.demo")}</Tip>
  </>;
  return {
    start: body("start", stages),
    seoNew: body("seoNew"),
    seoGrow: body("seoGrow"),
    aiReach: body("aiReach"),
    aiKnown: body("aiKnown"),
    aiRecommend: body("aiRecommend"),
    weekly: body("weekly"),
  };
}
```

- [ ] **Step 2: Checkpoint**

Run: `git status --short web/src/features/help`
Expected: `?? web/src/features/help/playbook.tsx`. (`tsc` still fails until Task 4.)

---

### Task 4: Topics, groups and old links

**Files:**
- Modify: `web/src/features/help/kit.tsx:9-27`
- Modify: `web/src/features/help/content/en.tsx:1-8`, `zh.tsx:1-8`, `pt.tsx:1-8`
- Modify: `web/src/features/help/help.tsx:17-23`
- Modify: `web/src/i18n/locales/{en,zh,pt}.ts` (`helpTopics`, `helpCenter.description`)

- [ ] **Step 1: Replace topic ids and groups in `kit.tsx`**

Replace lines 9–27 (from `// Topic order is the reading order` to the end of `TOPICS`) with:

```tsx
// Topic order is the reading order; the Next button follows it. Playbook
// topics come first; reference topics explain numbers and settings.
export const TOPIC_IDS = [
  "start", "seoNew", "seoGrow", "aiReach", "aiKnown", "aiRecommend", "weekly", "troubleshooting",
  "terms", "numbers", "audit", "prompts", "manual", "google", "providers", "schedule", "access", "updates", "limits",
] as const;
export type TopicId = (typeof TOPIC_IDS)[number];
export type GroupId = "start" | "seo" | "ai" | "loop" | "fix" | "reference";

export type Topic = { group: GroupId; label: Key; keys: Key };
const TOPIC_GROUPS: Record<TopicId, GroupId> = {
  start: "start", seoNew: "seo", seoGrow: "seo", aiReach: "ai", aiKnown: "ai", aiRecommend: "ai",
  weekly: "loop", troubleshooting: "fix", terms: "reference", numbers: "reference", audit: "reference",
  prompts: "reference", manual: "reference", google: "reference", providers: "reference",
  schedule: "reference", access: "reference", updates: "reference", limits: "reference",
};
export const TOPICS = Object.fromEntries(TOPIC_IDS.map(id => [id, {
  group: TOPIC_GROUPS[id], label: `helpTopics.${id}.label`, keys: `helpTopics.${id}.keys`,
}])) as Record<TopicId, Topic>;
```

- [ ] **Step 2: Update `helpTopics` in the three locales**

In each locale's `helpTopics` block: delete the entries `setup`, `pages`, `search`, `channels`, `indexing`, `opportunities`, `observations`, `reports`; replace `start`; add six new entries. Keep `updates`, `prompts`, `terms`, `numbers`, `audit`, `manual`, `providers`, `google`, `schedule`, `access`, `troubleshooting`, `limits` unchanged.

en.ts:

```ts
  "start": { "label": "Start here", "keys": "stage s0 s1 s2 cold start setup project google engine first" },
  "seoNew": { "label": "SEO: new site", "keys": "indexing index sitemap impressions new site inspection crawl" },
  "seoGrow": { "label": "SEO: growth", "keys": "brand non-brand ctr curve striking distance opportunity decline alerts" },
  "aiReach": { "label": "AI: reachable", "keys": "robots llms.txt crawler bots javascript gptbot claudebot" },
  "aiKnown": { "label": "AI: knows you", "keys": "recognition branded brand facts definition wrong answer" },
  "aiRecommend": { "label": "AI: recommends you", "keys": "visibility share of voice citations competitors questions sources directories" },
  "weekly": { "label": "Weekly 30 minutes", "keys": "weekly routine experiment action plan observation release report" },
```

zh.ts:

```ts
  "start": { "label": "从这里开始", "keys": "阶段 冷启动 设置 项目 stage setup" },
  "seoNew": { "label": "SEO：新站阶段", "keys": "收录 索引 sitemap 展示 新站 indexing" },
  "seoGrow": { "label": "SEO：起量阶段", "keys": "品牌词 非品牌 CTR 机会 下滑 告警 ctr" },
  "aiReach": { "label": "AI：抓得到", "keys": "robots llms.txt 爬虫 javascript" },
  "aiKnown": { "label": "AI：认得你", "keys": "认知度 品牌问题 品牌事实 定义 recognition" },
  "aiRecommend": { "label": "AI：推荐你", "keys": "可见性 声量 引用 竞品 问题 来源 目录站 visibility" },
  "weekly": { "label": "每周 30 分钟", "keys": "每周 节奏 实验 行动清单 观察 上线 周报 weekly" },
```

pt.ts:

```ts
  "start": { "label": "Comece aqui", "keys": "estágio partida configuração projeto stage setup" },
  "seoNew": { "label": "SEO: site novo", "keys": "indexação sitemap impressões site novo indexing" },
  "seoGrow": { "label": "SEO: crescimento", "keys": "marca sem marca ctr oportunidade queda alertas" },
  "aiReach": { "label": "IA: acessível", "keys": "robots llms.txt rastreador javascript" },
  "aiKnown": { "label": "IA: conhece você", "keys": "reconhecimento marca fatos definição recognition" },
  "aiRecommend": { "label": "IA: recomenda você", "keys": "visibilidade participação de voz citações concorrentes perguntas fontes" },
  "weekly": { "label": "30 minutos por semana", "keys": "semanal rotina experimento plano de ação observação lançamento relatório" },
```

Then grep for any other use of the deleted topic labels:

Run: `cd web && grep -rn "helpTopics\.\(setup\|pages\|search\|channels\|indexing\|opportunities\|observations\|reports\)" src | grep -v locales`
Expected: matches only in `features/help/usage.tsx` (removed in Task 5).

- [ ] **Step 3: Update `helpCenter.description` in the three locales**

- en: `"Playbooks for each stage: what to look at, what to do, and when you have passed. Every page title has an info icon that opens the matching playbook."`
- zh: `"按阶段的剧本：看什么、做什么、什么时候算过关。每个页面标题旁的信息图标会打开对应的剧本。"`
- pt: `"Roteiros por estágio: o que olhar, o que fazer e quando você passou. Cada título de página tem um ícone de informação que abre o roteiro correspondente."`

- [ ] **Step 4: New group names and playbook bodies in `content/en.tsx`, `zh.tsx`, `pt.tsx`**

In each file, add the import after the existing `usage` import:

```tsx
import { playbookBodies } from "../playbook";
```

Replace the `groups:` line:

- en.tsx: `groups: { start: "Start", seo: "SEO playbook", ai: "AI playbook", loop: "Every week", fix: "When something is wrong", reference: "Reference" },`
- zh.tsx: `groups: { start: "开始", seo: "SEO 剧本", ai: "AI 剧本", loop: "每周", fix: "出问题时", reference: "参考" },`
- pt.tsx: `groups: { start: "Comece", seo: "Roteiro de SEO", ai: "Roteiro de IA", loop: "Toda semana", fix: "Quando algo dá errado", reference: "Referência" },`

In each `body:` object, change `...usageBodies(k),` to:

```tsx
    ...usageBodies(k),
    ...playbookBodies(k),
```

- [ ] **Step 5: Old `#topic` links in `help.tsx`**

Replace `topicFromHash` (lines 17–23) with:

```tsx
// Topics replaced when the help center became playbooks. Old links and
// bookmarks land on the playbook that covers the same work.
const LEGACY_TOPICS: Record<string, TopicId> = {
  setup: "start", pages: "start", search: "seoNew", channels: "seoNew", indexing: "seoNew",
  opportunities: "weekly", observations: "weekly", reports: "weekly",
};

// The hash is #<topic> or #<topic>/<button>, the second form from a "?" tip.
function topicFromHash(): TopicId {
  const [id, button] = window.location.hash.replace(/^#/, "").split("/");
  const tip = TIP_IDS.find(id => id === button);
  if (tip) return TIPS[tip].topic;
  if (id in LEGACY_TOPICS) return LEGACY_TOPICS[id];
  return (TOPIC_IDS as readonly string[]).includes(id) ? (id as TopicId) : "start";
}
```

- [ ] **Step 6: Checkpoint**

Run: `cd web && ./node_modules/.bin/tsc --noEmit 2>&1 | head -30`
Expected: remaining errors only in `usage.tsx` (returns removed topics) and `app/tips.ts` (points at removed topics). Both are fixed in Task 5 and Task 6.

---

### Task 5: Slim `usage.tsx` and rewrite troubleshooting by symptom

**Files:**
- Modify: `web/src/features/help/usage.tsx`
- Modify: `web/src/i18n/locales/{en,zh,pt}.ts` (`helpUsage`)

- [ ] **Step 1: Replace `usageBodies` in `usage.tsx`**

Replace the whole `usageBodies` function (from `export function usageBodies` to its closing `}` before `export function usageButtons`) with:

```tsx
export function usageBodies({ t, n, page, topic }: Kit) {
  const names = {
    projects: n("nav.projects"), overview: n("nav.overview"), schedule: n("nav.schedule"),
    questions: n("nav.questions"), google: n("nav.google"), search: n("nav.search"),
    seoNew: t("helpTopics.seoNew.label"), aiReach: t("helpTopics.aiReach.label"),
    aiKnown: t("helpTopics.aiKnown.label"), aiRecommend: t("helpTopics.aiRecommend.label"),
  };
  const text = (key: Key) => t(key, names);
  const steps = (keys: Key[]) => <ol>{keys.map(key => <li key={key}>{text(key)}</li>)}</ol>;
  const links = (...ids: TopicId[]) => <p>{t("helpUsage.related")}{": "}{ids.map((id, i) => <span key={id}>{i > 0 && " · "}{topic(id, t(`helpTopics.${id}.label`))}</span>)}</p>;
  const section = (id: TopicId, intro: Key, body: ReactNode) => <>
    <h2>{t(`helpTopics.${id}.label`)}</h2><p>{text(intro)}</p>{body}
  </>;
  // Symptoms the playbooks are about come first.
  const issues = [9, 10, 11, 0, 1, 2, 3, 4, 5, 6, 7, 8] as const;
  return {
    updates: <>
      <h2>{t("helpTopics.updates.label")}</h2>
      <p>{t("update.help", { system: n("nav.system") })}</p>
      <p>{page("settings/system", "nav.system")}</p>
      <Warn>{t("update.rollbackHelp")}</Warn>
      <p>{t("update.restartHelp")}</p><p>{t("update.restartUnavailable")}</p>
      <Tip>{t("update.containerHelp")}</Tip>
    </>,
    prompts: section("prompts", "helpUsage.promptsIntro", <>
      <Tip>{t("helpUsage.promptExample")}</Tip>
      {steps(["helpUsage.promptBrand", "helpUsage.promptVersions", "helpUsage.promptPreview"])}
      <Warn>{t("helpUsage.promptCompare")}</Warn>
      <p>{page("settings/questions", "nav.questions")}{" · "}{page("settings/schedule", "nav.schedule")}</p>
      {links("aiRecommend", "numbers", "manual")}
    </>),
    google: section("google", "helpUsage.googleIntro", <>
      {steps(["helpUsage.googleOAuth", "helpUsage.googleSA", "helpUsage.googleHistory", "helpUsage.googleCoverage"])}
      <Tip>{t("helpUsage.googleRetry")}</Tip>
      <p>{page("settings/google", "nav.google")}{" · "}{page("search", "nav.search")}</p>
      {links("seoNew", "troubleshooting")}
    </>),
    troubleshooting: <>
      <h2>{t("helpTopics.troubleshooting.label")}</h2>
      <Table head={[t("helpUsage.problem"), t("helpUsage.next")]}
        rows={issues.map(i => [t(`helpUsage.issue${i}`), text(`helpUsage.fix${i}`)])} />
      {links("seoNew", "aiReach", "weekly", "google", "access")}
    </>,
  };
}
```

- [ ] **Step 2: Add the three symptom rows to `helpUsage` in each locale**

Add after `"fix8"` in each file.

en.ts:

```ts
  "issue9": "New pages are not indexed",
  "fix9": "Follow {seoNew}: check the audit's Access and Discover layers, that the page is in a sitemap, and the indexing history. If pages are crawled but not indexed, the cause is usually quality, not a technical setting.",
  "issue10": "Search traffic dropped",
  "fix10": "First rule out a false alarm: same property, same days of the week, finished data. Then split by page, query, country and device to find where it fell.",
  "issue11": "AI engines do not mention me",
  "fix11": "Go through {aiReach}, {aiKnown} and {aiRecommend} in order. Most often engines cannot fetch the site, or the sources they cite never mention you.",
```

zh.ts:

```ts
  "issue9": "新页面不收录",
  "fix9": "按{seoNew}排查：体检的访问和发现层、页面是否在 sitemap 里、收录检查历史。已抓取却不收录，原因多半是质量，不是技术设置。",
  "issue10": "搜索流量掉了",
  "fix10": "先排除假警报：同一个资源、相同的星期几、数据已完整。再按页面、查询词、国家、设备切开，找到掉在哪里。",
  "issue11": "AI 不提我",
  "fix11": "按顺序过一遍{aiReach}、{aiKnown}、{aiRecommend}。最常见的原因是 AI 抓不到站，或者它引用的来源里从来没提过你。",
```

pt.ts:

```ts
  "issue9": "Páginas novas não são indexadas",
  "fix9": "Siga {seoNew}: verifique as camadas Acesso e Descoberta da auditoria, se a página está num sitemap e o histórico de indexação. Se a página é rastreada mas não indexada, a causa costuma ser qualidade, não configuração técnica.",
  "issue10": "O tráfego de busca caiu",
  "fix10": "Primeiro descarte um alarme falso: mesma propriedade, mesmos dias da semana, dados finalizados. Depois divida por página, consulta, país e dispositivo para achar onde caiu.",
  "issue11": "Os mecanismos de IA não me mencionam",
  "fix11": "Percorra {aiReach}, {aiKnown} e {aiRecommend} em ordem. O mais comum é os mecanismos não conseguirem buscar o site, ou as fontes citadas nunca mencionarem você.",
```

- [ ] **Step 3: Delete the `helpUsage` keys no topic uses any more**

Create `web/scripts/drop-help-keys.mjs` (temporary, deleted in Step 4):

```js
// One-off: remove helpUsage entries for help topics replaced by playbooks.
import { readFileSync, writeFileSync } from "node:fs";

const dead = [
  "startIntro", "startCreate", "startEvidence", "startAction", "startOptional",
  "setupIntro", "setupCreate", "setupLanguages", "setupReview", "setupRetry",
  "searchIntro", "searchFilter", "searchDrill", "searchQuality", "searchStage",
  "channelsIntro", "channelsSessions", "channelsEvents", "channelsMapping", "channelsUnavailable",
  "indexingIntro", "indexingDiscover", "indexingInspect", "indexingDates",
  "actionsIntro", "actionsRank", "actionsCTR", "actionsState",
  "observationsIntro", "observationsRecord", "observationsEvaluate", "observationsLimits",
  "reportsIntro", "reportsRead", "reportsExport", "reportsLimit",
  "pagesIntro", "pageColumn", "useColumn",
  "page_overview", "page_visibility", "page_sov", "page_citations", "page_fanout", "page_answers",
  "page_search", "page_indexing", "page_channels", "page_actions", "page_audit", "page_reports",
  "page_questions", "page_schedule",
];
for (const loc of ["en", "zh", "pt"]) {
  const file = new URL(`../src/i18n/locales/${loc}.ts`, import.meta.url);
  const lines = readFileSync(file, "utf8").split("\n");
  const kept = lines.filter((l) => !dead.some((k) => new RegExp(`^\\s*"${k}":`).test(l)));
  const removed = lines.length - kept.length;
  if (removed !== dead.length) throw new Error(`${loc}: removed ${removed}, expected ${dead.length}`);
  writeFileSync(file, kept.join("\n"));
  console.log(`${loc}: removed ${removed}`);
}
```

Before running, confirm each key is one line and used nowhere else:

Run: `cd web && grep -rn "helpUsage\.\(start\|setup\|search\|channels\|indexing\|actions\|observations\|reports\|page\)" src | grep -v locales`
Expected: no output.

Run: `cd web && node scripts/drop-help-keys.mjs`
Expected:
```
en: removed 53
zh: removed 53
pt: removed 53
```

If the count differs, a key spans several lines or is missing in one locale; fix by hand and do not proceed until all three match.

- [ ] **Step 4: Remove the one-off script**

Run: `rm web/scripts/drop-help-keys.mjs`

- [ ] **Step 5: Checkpoint**

Run: `cd web && ./node_modules/.bin/tsc --noEmit 2>&1 | head -20`
Expected: remaining errors only in `app/tips.ts`.

---

### Task 6: Point button tips and page icons at the playbooks

**Files:**
- Modify: `web/src/app/tips.ts:10-53`
- Modify: `web/src/app/nav.ts:104-125`

- [ ] **Step 1: Re-point tips in `tips.ts`**

Change only the `topic` of these entries; leave the others as they are:

```ts
  indexing: { topic: "seoNew", label: "schedule.actions.indexing" },
  createProject: { topic: "start", label: "projects.create" },
  saveBrand: { topic: "aiKnown", label: "brand.save" },
  addCompetitor: { topic: "aiRecommend", label: "competitors.add" },
  saveCompetitors: { topic: "aiRecommend", label: "common.save" },
  accept: { topic: "weekly", label: "plan.actions.accept" },
  dismiss: { topic: "weekly", label: "plan.actions.dismiss" },
  progress: { topic: "weekly", label: "plan.actions.markDone" },
  restore: { topic: "weekly", label: "plan.actions.restore" },
  buildReport: { topic: "weekly", label: "reports.build" },
  downloadReport: { topic: "weekly", label: "reports.html" },
  searchSync: { topic: "seoNew", label: "search.sync" },
  saveKeyword: { topic: "seoGrow", label: "search.save" },
```

- [ ] **Step 2: Replace `helpTopicFor` in `nav.ts`**

```ts
// helpTopicFor maps a page to the help topic its info icon opens: the
// playbook whose steps use the page, or a reference topic for settings.
export function helpTopicFor(path: string): string | null {
  const leaf = path.split("/").slice(3).join("/");
  if (leaf === "help") return null;
  if (!leaf || leaf === "overview") return "start";
  if (leaf === "ai/answers") return "manual";
  if (leaf.startsWith("ai/")) return "aiRecommend";
  if (leaf.startsWith("opportunities") || leaf === "reports") return "weekly";
  if (leaf.startsWith("audit")) return "audit";
  if (leaf === "search/keywords") return "seoGrow";
  if (leaf.startsWith("search")) return "seoNew";
  if (leaf === "settings/system") return "updates";
  if (leaf === "settings/google") return "google";
  if (leaf === "settings/questions") return "prompts";
  if (leaf === "settings/brand") return "aiKnown";
  if (leaf === "settings/competitors") return "aiRecommend";
  if (leaf === "settings/providers") return "providers";
  if (leaf === "settings/schedule") return "schedule";
  if (leaf === "settings/users" || leaf === "settings/password") return "access";
  return "start";
}
```

- [ ] **Step 3: Type-check and CJK check**

Run: `cd web && ./node_modules/.bin/tsc --noEmit && npm run check:cjk`
Expected: no `tsc` output; `ok: no Chinese text outside the i18n catalogs`.

- [ ] **Step 4: Check every `usageButtons` / `buttons` tip still has a topic that renders**

Run: `cd web && grep -n "topic:" src/app/tips.ts | grep -o 'topic: "[a-zA-Z]*"' | sort -u`
Expected: every topic listed is in `TOPIC_IDS` (start, seoNew, seoGrow, aiKnown, aiRecommend, weekly, schedule, audit, manual, prompts, google, providers, access). `tsc` already enforces this through `satisfies Record<string, { topic: TopicId; ... }>`; this is a read-back.

- [ ] **Step 5: Checkpoint**

Run: `git status --short web`

---

### Task 7: Check that every step links to a real page

`tsc` proves text keys exist; it cannot prove `page.path` is a page. This script does.

**Files:**
- Create: `web/scripts/check-playbook.mjs`
- Modify: `web/package.json` (scripts)
- Modify: `.github/workflows/ci.yml` (after the `check:cjk` step, line 38)

- [ ] **Step 1: Write the check**

```js
// SPDX-License-Identifier: AGPL-3.0-or-later

// Fails when a playbook step links to an address that is not a page or tab
// in the navigation catalog, so a renamed page cannot leave a dead step.
import { readFileSync } from "node:fs";

const read = (p) => readFileSync(new URL(p, import.meta.url), "utf8");
const nav = read("../src/app/nav.ts");
const catalog = read("../src/features/playbook/catalog.ts");

const pages = new Set([...nav.matchAll(/\bto:\s*"([^"]+)"/g)].map((m) => m[1]));
const used = [...catalog.matchAll(/\bpath:\s*"([^"]+)"/g)].map((m) => m[1]);
const bad = [...new Set(used)].filter((p) => !pages.has(p));
if (!used.length) {
  console.error("check-playbook: no step paths found; did the catalog format change?");
  process.exit(1);
}
if (bad.length) {
  console.error("Playbook steps link to addresses that are not in NAV:\n" + bad.join("\n"));
  process.exit(1);
}
console.log(`ok: ${used.length} playbook links, all in NAV`);
```

- [ ] **Step 2: Prove it fails on a bad path**

Temporarily change one step in `catalog.ts` from `path: "reports"` to `path: "reportz"`.

Run: `cd web && node scripts/check-playbook.mjs`
Expected: exit 1 and
```
Playbook steps link to addresses that are not in NAV:
reportz
```

Revert the change.

- [ ] **Step 3: Prove it passes**

Run: `cd web && node scripts/check-playbook.mjs`
Expected: `ok: 32 playbook links, all in NAV`

- [ ] **Step 4: Wire it in**

In `web/package.json` `scripts`, after `"check:cjk"`:

```json
    "check:playbook": "node scripts/check-playbook.mjs",
```

In `.github/workflows/ci.yml`, after `- run: npm run check:cjk`:

```yaml
      - run: npm run check:playbook
```

- [ ] **Step 5: License header check**

Run: `scripts/license-headers.sh`
Expected: passes (the new `.ts`, `.tsx`, `.mjs` files start with the SPDX line). If it reports a file, run `scripts/license-headers.sh --fix`.

- [ ] **Step 6: Checkpoint**

Run: `git status --short`

---

### Task 8: Full verification

- [ ] **Step 1: Static checks**

Run: `cd web && ./node_modules/.bin/tsc --noEmit && npm run check:cjk && npm run check:playbook && npm run build`
Expected: no `tsc` output, both checks print `ok`, Vite build succeeds.

- [ ] **Step 2: Go tests unaffected**

Run: `go test ./... -count=1`
Expected: all packages `ok` (no Go changed; this guards the embedded build).

- [ ] **Step 3: Run the demo and read the help center in three languages**

Use the `run` skill, or run `make demo` and open the URL it prints. In the demo project open `/p/<slug>/help`, then for each of `?lang=en`, `?lang=zh`, `?lang=pt` check:

1. The sidebar shows six groups in order: Start, SEO playbook, AI playbook, Every week, When something is wrong, Reference.
2. **Start here** shows the S0/S1/S2 table, the S0 tip with four book links, and three once-steps.
3. **SEO: new site** shows the “what this stage asks” line, the warning, steps under Once / Every day / Every week / Every month, and the 8-row pass table with `{…}` replaced by numbers (3, 80, 5, 4, 50). No literal `{` appears anywhere.
4. Every step's page link opens the right page in the demo project; every book link opens the chapter on GitHub (the AI chapter 404s until Task 0's file is pushed by the user; note it, do not treat as a failure).
5. Old links: `/help#search` opens **SEO: new site**; `/help#opportunities` opens **Weekly 30 minutes**; `/help#search/searchSync` opens **SEO: new site** and highlights the Sync button.
6. The info icon next to page titles: Overview → Start here; Visibility → AI: recommends you; Search → SEO: new site; Keywords → SEO: growth; Action plan → Weekly 30 minutes; Brand → AI: knows you.
7. Search the help sidebar for `sitemap`, `收录` and `indexação`: each finds SEO: new site.

- [ ] **Step 4: Report**

Tell the user what passed, what did not, and list the screens checked per language. Remind them nothing is committed in either repository, and that the AI chapter (Task 0) still needs their review.

---

## Next plans (not in this one)

- **Phase 2:** `internal/service/playbook` read-only status API, `product_stage` on the project, manual signal confirmations table (SQLite + MySQL 8), pass-signal and done-check computation using `THRESHOLDS`, help-center playbooks show the current stage, ✓/✗ per step and pass progress for the current project (overview gets only a link), `search_stage` promotion by the book's signals.
- **Phase 3:** search tabs 8→5 (countries/devices as a dimension selector, channels + landings → Arrivals), fan-out under Citations, CTR reference and page mapping into row detail, `LEGACY_REDIRECTS`, update `STEPS[].page` paths.
- **Phase 4:** weekly report by SEO stage using the two book templates.
