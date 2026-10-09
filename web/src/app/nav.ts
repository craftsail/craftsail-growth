// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Key } from "../i18n";

export type NavIcon =
  | "home" | "eye" | "pie" | "quote" | "split" | "messages"
  | "listcheck" | "stethoscope" | "search" | "report"
  | "folders" | "id" | "users" | "list" | "calendar" | "cpu" | "google" | "usercog";

// Navigation rule: the sidebar is a flat list of titled sections, one click to
// any page. A page with sub-views (Site audit, Search traffic) shows them as
// tabs inside the page, never as nested menu items.
type NavTab = { label: Key; to: string };
// admin: only admins see the item (workspace settings).
type NavItem = { id: string; label: Key; icon: NavIcon; to: string; tabs?: NavTab[]; admin?: true };
type NavSection = { id: string; label?: Key; items: NavItem[] };

export const NAV: NavSection[] = [
  { id: "home", items: [{ id: "overview", label: "nav.overview", icon: "home", to: "overview" }] },
  {
    id: "measure", label: "nav.sections.measure", items: [
      { id: "visibility", label: "nav.visibility", icon: "eye", to: "ai/visibility" },
      { id: "sov", label: "nav.sov", icon: "pie", to: "ai/share-of-voice" },
      { id: "citations", label: "nav.citations", icon: "quote", to: "ai/citations" },
      { id: "fanout", label: "nav.fanout", icon: "split", to: "ai/fan-out" },
      { id: "answers", label: "nav.answers", icon: "messages", to: "ai/answers" },
    ],
  },
  {
    id: "improve", label: "nav.sections.improve", items: [
      { id: "plan", label: "nav.actionPlan", icon: "listcheck", to: "opportunities" },
      {
        id: "audit", label: "nav.audit", icon: "stethoscope", to: "audit", tabs: [
          { label: "nav.tabs.readiness", to: "audit" },
          { label: "nav.tabs.issues", to: "audit/issues" },
          { label: "nav.tabs.pages", to: "audit/pages" },
        ],
      },
      {
        id: "search", label: "nav.search", icon: "search", to: "search", tabs: [
          { label: "nav.tabs.performance", to: "search" },
          { label: "nav.tabs.indexing", to: "search/indexing" },
          { label: "nav.tabs.keywords", to: "search/keywords" },
          { label: "nav.tabs.pages", to: "search/pages" },
          {label:"searchSegments.countries",to:"search/countries"},
          {label:"searchSegments.devices",to:"search/devices"},
          { label: "nav.tabs.channels", to: "search/channels" },
          { label: "nav.tabs.landings", to: "search/landings" },
        ],
      },
      { id: "reports", label: "nav.reports", icon: "report", to: "reports" },
    ],
  },
  {
    id: "project", label: "nav.sections.project", items: [
      { id: "brand", label: "nav.brand", icon: "id", to: "settings/brand" },
      { id: "competitors", label: "nav.competitors", icon: "users", to: "settings/competitors" },
      { id: "questions", label: "nav.questions", icon: "list", to: "settings/questions" },
      { id: "schedule", label: "nav.schedule", icon: "calendar", to: "settings/schedule" },
    ],
  },
  {
    id: "workspace", label: "nav.sections.workspace", items: [
      { id: "projects", label: "nav.projects", icon: "folders", to: "settings/projects", admin: true },
      { id: "users", label: "nav.users", icon: "usercog", to: "settings/users", admin: true },
      { id: "providers", label: "nav.providers", icon: "cpu", to: "settings/providers", admin: true },
      { id: "google", label: "nav.google", icon: "google", to: "settings/google", admin: true },
      { id: "system", label: "nav.system", icon: "cpu", to: "settings/system", admin: true },
    ],
  },
];

// leafOf returns the part of the address after /p/<slug>/.
function leafOf(path: string): string {
  return path.split("/").slice(3).join("/").replace(/\/$/, "");
}

// itemFor finds the menu item a page belongs to, including its tab pages.
export function itemFor(path: string): NavItem | undefined {
  const leaf = leafOf(path) || "overview";
  for (const sec of NAV) {
    for (const it of sec.items) {
      if (it.to === leaf || it.tabs?.some((t) => t.to === leaf)) return it;
    }
  }
  if (leaf.startsWith("ai/prompts/")) return NAV[1].items[0];
  return undefined;
}

// titleFor gives the page title key: the menu item, or a special page.
export function titleFor(path: string): Key {
  const leaf = leafOf(path);
  if (leaf === "help") return "nav.help";
  if (leaf.startsWith("ai/prompts/")) return "nav.question";
  if (leaf === "settings/password") return "access.changePassword";
  return itemFor(path)?.label || "nav.overview";
}

// Old addresses keep working for bookmarks: when a page moves, map its old
// leaf here to the new one.
export const LEGACY_REDIRECTS: Record<string, string> = {};


// helpTopicFor maps a page to the help topic its info icon opens.
export function helpTopicFor(path: string): string | null {
  const leaf = path.split("/").slice(3).join("/");
  if (leaf === "help") return null;
  if (!leaf || leaf === "overview") return "start";
  if (leaf === "ai/answers") return "manual";
  if (leaf.startsWith("ai/")) return "numbers";
  if (leaf.startsWith("opportunities")) return "opportunities";
  if (leaf.startsWith("audit")) return "audit";
  if (leaf === "search/indexing") return "indexing";
  if (leaf === "search/channels" || leaf === "search/landings") return "channels";
  if (leaf.startsWith("search")) return "search";
  if (leaf === "settings/system") return "updates";
  if (leaf === "settings/google") return "google";
  if (leaf === "reports") return "reports";
  if (leaf === "settings/questions") return "prompts";
  if (leaf === "settings/providers") return "providers";
  if (leaf === "settings/schedule") return "schedule";
  if (leaf === "settings/users" || leaf === "settings/password") return "access";
  if (leaf.startsWith("settings/")) return "setup";
  return "start";
}
