// SPDX-License-Identifier: AGPL-3.0-or-later

import type { OpportunityItem } from "../../api";
import type { Key, Locale, Vars } from "../../i18n";
import { ruleText } from "../../i18n/rules";

type T = (key: Key, vars?: Vars) => string;

const CITATION_TITLES: Record<string, Key> = {
  social: "plan.titles.citation_social",
  outreach: "plan.titles.citation_outreach",
  "existing-content": "plan.titles.citation_refresh",
  creation: "plan.titles.citation_creation",
};

// opportunityTitle translates an item's title. The server's English title is
// the fallback for kinds the catalog does not cover.
export function opportunityTitle(it: OpportunityItem, t: T, locale: Locale): string {
  if (it.source === "audit") return ruleText(locale, it.kind, "title", it.title);
  if (it.source === "metric" && it.kind === "visibility_down") return t("plan.titles.visibility_down");
  if (it.source === "search" && it.kind === "traffic_drop") return t("plan.titles.traffic_drop");
  if (it.source === "search" && it.kind === "content_decay" && it.urls?.[0]) return t("plan.titles.content_decay", { q: it.urls[0] });
  const q = typeof it.detail?.query === "string" ? (it.detail.query as string) : "";
  if (it.source === "search" && q && ["striking_distance", "low_ctr", "content_decay", "cannibalization"].includes(it.kind)) {
    return t(`plan.titles.${it.kind}` as Key, { q });
  }
  if (it.source === "citation") {
    const prompt = typeof it.detail?.prompt === "string" ? (it.detail.prompt as string) : "";
    if (!prompt && it.kind === "existing-content") return t("plan.titles.citation_recover");
    const key = CITATION_TITLES[it.kind];
    if (key && prompt) return t(key, { q: prompt });
  }
  return it.title;
}

export function opportunityWhy(it: OpportunityItem, t: T, locale: Locale): string {
  if (it.source === "audit") return ruleText(locale, it.kind, "why", it.why);
  if (it.source === "search" && it.detail?.facts && typeof it.detail.facts === "object") {
    const facts = it.detail.facts as Record<string, unknown>;
    if (it.detail.reason === "local_ctr" && typeof facts.reference_ctr === "number" && typeof facts.scenario_clicks === "number") return t("prioritization.scenario", {ctr: Math.round(facts.reference_ctr * 10000) / 100, clicks: Math.round(facts.scenario_clicks)});
    if (it.detail.reason === "candidate" && typeof facts.position === "number" && typeof facts.impressions === "number") {
      return t("plan.titles.search_candidate_why", { position: Math.round(facts.position * 10) / 10, impressions: facts.impressions });
    }
    if (it.detail.reason === "sustained_decline" && typeof facts.current === "number" && typeof facts.previous === "number") {
      return t("plan.titles.search_decline_why", { current: facts.current, previous: facts.previous });
    }
  }
  return it.why;
}
