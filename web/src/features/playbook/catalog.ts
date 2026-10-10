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
  visibleShare: 50, recognitionN: 30, recognitionLower: 50, indexDrop: 5, seoPassSignals: 6,
} as const;

// Done checks the server can judge for a step (internal/service/playbook).
export const CHECKS = [
  "first_check", "google_connected", "engines_answered", "audit_layers_ok", "sitemap_ok", "index_checked_7d",
  "audit_access_ok", "llms_published", "brand_facts", "recognition_n", "questions_confirmed", "competitors_3",
  "experiment_open", "experiments_reviewed", "report_this_week",
] as const;
export type CheckId = (typeof CHECKS)[number];

type Step = { id: string; topic: PlaybookTopic; cadence: Cadence; page: { path: string; label: Key }; books: readonly BookRef[]; check?: CheckId };

// Order inside a topic and cadence is the reading order.
export const STEPS = [
  { id: "project", topic: "start", cadence: "once", page: { path: "settings/projects", label: "nav.projects" }, books: [{ book: "model" }], check: "first_check" },
  { id: "google", topic: "start", cadence: "once", page: { path: "settings/google", label: "nav.google" }, books: [{ book: "seoMonitor", section: "2" }], check: "google_connected" },
  { id: "engines", topic: "start", cadence: "once", page: { path: "settings/providers", label: "nav.providers" }, books: [{ book: "model" }], check: "engines_answered" },

  { id: "seoLayers", topic: "seoNew", cadence: "once", page: { path: "audit", label: "nav.audit" }, books: [{ book: "seoSetup", section: "15" }], check: "audit_layers_ok" },
  { id: "seoSitemap", topic: "seoNew", cadence: "once", page: { path: "audit/issues", label: "nav.tabs.issues" }, books: [{ book: "seoSetup", section: "8.2" }, { book: "seoNew", section: "3.1" }], check: "sitemap_ok" },
  { id: "seoInspect", topic: "seoNew", cadence: "daily", page: { path: "search/indexing", label: "nav.tabs.indexing" }, books: [{ book: "seoNew", section: "3.4" }], check: "index_checked_7d" },
  { id: "seoIndexWeek", topic: "seoNew", cadence: "weekly", page: { path: "search/indexing", label: "nav.tabs.indexing" }, books: [{ book: "seoNew", section: "3" }] },
  { id: "seoImprWeek", topic: "seoNew", cadence: "weekly", page: { path: "search", label: "nav.tabs.performance" }, books: [{ book: "seoNew", section: "4" }] },
  { id: "seoPeopleWeek", topic: "seoNew", cadence: "weekly", page: { path: "search/arrivals", label: "nav.tabs.arrivals" }, books: [{ book: "seoNew", section: "5" }] },
  { id: "seoZeroMonth", topic: "seoNew", cadence: "monthly", page: { path: "search/pages", label: "nav.tabs.pages" }, books: [{ book: "seoNew", section: "7" }] },
  { id: "seoTitlesMonth", topic: "seoNew", cadence: "monthly", page: { path: "search/pages", label: "nav.tabs.pages" }, books: [{ book: "seoNew", section: "7" }] },

  { id: "growAlerts", topic: "seoGrow", cadence: "daily", page: { path: "search", label: "nav.tabs.performance" }, books: [{ book: "seoMonitor", section: "6.13" }] },
  { id: "growBrand", topic: "seoGrow", cadence: "weekly", page: { path: "search/keywords", label: "nav.tabs.keywords" }, books: [{ book: "seoMonitor", section: "3.2" }] },
  { id: "growStriking", topic: "seoGrow", cadence: "weekly", page: { path: "opportunities", label: "nav.actionPlan" }, books: [{ book: "seoMonitor", section: "6.4" }] },
  { id: "growDecline", topic: "seoGrow", cadence: "weekly", page: { path: "search/pages", label: "nav.tabs.pages" }, books: [{ book: "seoMonitor", section: "6.14" }] },
  { id: "growCtrMonth", topic: "seoGrow", cadence: "monthly", page: { path: "search/keywords", label: "nav.tabs.keywords" }, books: [{ book: "seoMonitor", section: "6.2" }, { book: "seoMonitor", section: "6.3" }] },

  { id: "aiBots", topic: "aiReach", cadence: "once", page: { path: "audit/issues", label: "nav.tabs.issues" }, books: [{ book: "seoSetup", section: "8.1" }, { book: "ai", section: "4" }], check: "audit_access_ok" },
  { id: "aiLlms", topic: "aiReach", cadence: "once", page: { path: "audit", label: "nav.audit" }, books: [{ book: "seoSetup", section: "8.3" }, { book: "ai", section: "4" }], check: "llms_published" },
  { id: "aiNoJs", topic: "aiReach", cadence: "once", page: { path: "audit/pages", label: "nav.tabs.pages" }, books: [{ book: "seoSetup", section: "10.3" }] },

  { id: "knownFacts", topic: "aiKnown", cadence: "once", page: { path: "settings/brand", label: "nav.brand" }, books: [{ book: "toolGrowth", section: "3.1" }, { book: "ai", section: "5" }], check: "brand_facts" },
  { id: "knownRecognition", topic: "aiKnown", cadence: "weekly", page: { path: "ai/visibility", label: "nav.visibility" }, books: [{ book: "ai", section: "3" }], check: "recognition_n" },
  { id: "knownRead", topic: "aiKnown", cadence: "weekly", page: { path: "ai/answers", label: "nav.answers" }, books: [{ book: "ai", section: "5" }] },

  { id: "recQuestions", topic: "aiRecommend", cadence: "once", page: { path: "settings/questions", label: "nav.questions" }, books: [{ book: "toolGrowth", section: "3" }, { book: "pmf", section: "3.2" }], check: "questions_confirmed" },
  { id: "recCompetitors", topic: "aiRecommend", cadence: "once", page: { path: "settings/competitors", label: "nav.competitors" }, books: [{ book: "toolGrowth", section: "3.2" }], check: "competitors_3" },
  { id: "recVisibility", topic: "aiRecommend", cadence: "weekly", page: { path: "ai/visibility", label: "nav.visibility" }, books: [{ book: "ai", section: "3" }] },
  { id: "recCitations", topic: "aiRecommend", cadence: "weekly", page: { path: "ai/citations", label: "nav.citations" }, books: [{ book: "ai", section: "6" }, { book: "directories", section: "6" }] },
  { id: "recGetListed", topic: "aiRecommend", cadence: "weekly", page: { path: "opportunities", label: "nav.actionPlan" }, books: [{ book: "directories", section: "8" }, { book: "communities", section: "3" }] },

  { id: "weekRead", topic: "weekly", cadence: "weekly", page: { path: "overview", label: "nav.overview" }, books: [{ book: "seoNew", section: "7" }, { book: "seoMonitor", section: "7" }] },
  { id: "weekPick", topic: "weekly", cadence: "weekly", page: { path: "opportunities", label: "nav.actionPlan" }, books: [{ book: "toolGrowth", section: "11.1" }], check: "experiment_open" },
  { id: "weekShip", topic: "weekly", cadence: "weekly", page: { path: "opportunities", label: "nav.actionPlan" }, books: [{ book: "toolGrowth", section: "11.2" }] },
  { id: "weekReview", topic: "weekly", cadence: "weekly", page: { path: "opportunities", label: "nav.actionPlan" }, books: [{ book: "toolData", section: "9.3" }], check: "experiments_reviewed" },
  { id: "weekReport", topic: "weekly", cadence: "weekly", page: { path: "reports", label: "nav.reports" }, books: [{ book: "seoNew", section: "7" }, { book: "seoMonitor", section: "8" }], check: "report_this_week" },
] as const satisfies readonly Step[];
export type StepId = (typeof STEPS)[number]["id"];

type Signal = { id: string; topic: PlaybookTopic; auto: boolean };

// Pass signals per stage. auto: phase 2 can judge it from stored data;
// otherwise the user confirms it by hand.
export const SIGNALS = [
  { id: "newFast", topic: "seoNew", auto: true },
  { id: "sitemapStable", topic: "seoNew", auto: false },
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
