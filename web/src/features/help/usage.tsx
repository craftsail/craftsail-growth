// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ReactNode } from "react";
import type { Key } from "../../i18n";
import { Table, Tip, Warn, type Kit, type TopicId } from "./kit";

// Shared structure keeps the three guides aligned; all prose lives in i18n.
export function usageBodies({ t, n, page, topic }: Kit) {
  const names = {
    projects: n("nav.projects"), overview: n("nav.overview"), actions: n("nav.actionPlan"),
    schedule: n("nav.schedule"), brand: n("nav.brand"), competitors: n("nav.competitors"),
    questions: n("nav.questions"), google: n("nav.google"), search: n("nav.search"),
    channels: n("nav.tabs.channels"), landings: n("nav.tabs.landings"), reports: n("nav.reports"),
  };
  const text = (key: Key) => t(key, names);
  const steps = (keys: Key[]) => <ol>{keys.map(key => <li key={key}>{text(key)}</li>)}</ol>;
  const links = (...ids: TopicId[]) => <p>{t("helpUsage.related")}{": "}{ids.map((id, i) => <span key={id}>{i > 0 && " · "}{topic(id, t(`helpTopics.${id}.label`))}</span>)}</p>;
  const section = (id: TopicId, intro: Key, body: ReactNode) => <>
    <h2>{t(`helpTopics.${id}.label`)}</h2><p>{text(intro)}</p>{body}
  </>;
  const destinations: [string, Key, Key][] = [
    ["overview", "nav.overview", "helpUsage.page_overview"],
    ["ai/visibility", "nav.visibility", "helpUsage.page_visibility"],
    ["ai/share-of-voice", "nav.sov", "helpUsage.page_sov"],
    ["ai/citations", "nav.citations", "helpUsage.page_citations"],
    ["ai/fan-out", "nav.fanout", "helpUsage.page_fanout"],
    ["ai/answers", "nav.answers", "helpUsage.page_answers"],
    ["search", "nav.search", "helpUsage.page_search"],
    ["search/indexing", "nav.tabs.indexing", "helpUsage.page_indexing"],
    ["search/channels", "nav.tabs.channels", "helpUsage.page_channels"],
    ["opportunities", "nav.actionPlan", "helpUsage.page_actions"],
    ["audit", "nav.audit", "helpUsage.page_audit"],
    ["reports", "nav.reports", "helpUsage.page_reports"],
    ["settings/questions", "nav.questions", "helpUsage.page_questions"],
    ["settings/schedule", "nav.schedule", "helpUsage.page_schedule"],
  ];
  const issues = [0, 1, 2, 3, 4, 5, 6, 7, 8] as const;
  return {
    start: section("start", "helpUsage.startIntro", <>
      {steps(["helpUsage.startCreate", "helpUsage.startEvidence", "helpUsage.startAction", "helpUsage.startOptional"])}
      <p>{page("settings/projects", "nav.projects")}{" · "}{page("overview", "nav.overview")}</p>
      <Tip>{t("helpUsage.demo")}</Tip>{links("setup", "search", "observations")}
    </>),
    setup: section("setup", "helpUsage.setupIntro", <>
      {steps(["helpUsage.setupCreate", "helpUsage.setupLanguages", "helpUsage.setupReview"])}
      <Tip>{t("helpUsage.setupRetry")}</Tip>
      <p>{page("settings/projects", "nav.projects")}{" · "}{page("settings/brand", "nav.brand")}{" · "}{page("settings/questions", "nav.questions")}</p>
      {links("prompts", "google", "providers")}
    </>),
    prompts: section("prompts", "helpUsage.promptsIntro", <>
      <Tip>{t("helpUsage.promptExample")}</Tip>
      {steps(["helpUsage.promptBrand", "helpUsage.promptVersions", "helpUsage.promptPreview"])}
      <Warn>{t("helpUsage.promptCompare")}</Warn>
      <p>{page("settings/questions", "nav.questions")}{" · "}{page("settings/schedule", "nav.schedule")}</p>
      {links("numbers", "manual")}
    </>),
    pages: section("pages", "helpUsage.pagesIntro", <Table
      head={[t("helpUsage.pageColumn"), t("helpUsage.useColumn")]}
      rows={destinations.map(([path, label, description]) => [page(path, label), t(description)])}
    />),
    google: section("google", "helpUsage.googleIntro", <>
      {steps(["helpUsage.googleOAuth", "helpUsage.googleSA", "helpUsage.googleHistory", "helpUsage.googleCoverage"])}
      <Tip>{t("helpUsage.googleRetry")}</Tip>
      <p>{page("settings/google", "nav.google")}{" · "}{page("search", "nav.search")}</p>
      {links("search", "channels", "indexing", "troubleshooting")}
    </>),
    search: section("search", "helpUsage.searchIntro", <>
      {steps(["helpUsage.searchFilter", "helpUsage.searchDrill"])}
      <Warn>{t("helpUsage.searchQuality")}</Warn><p>{t("helpUsage.searchStage")}</p>
      <p>{page("search", "nav.search")}{" · "}{page("search/keywords", "nav.tabs.keywords")}{" · "}{page("search/pages", "nav.tabs.pages")}</p>
      {links("google", "channels", "indexing", "opportunities")}
    </>),
    channels: section("channels", "helpUsage.channelsIntro", <>
      {steps(["helpUsage.channelsSessions", "helpUsage.channelsEvents", "helpUsage.channelsMapping"])}
      <Warn>{t("helpUsage.channelsUnavailable")}</Warn>
      <p>{page("search/channels", "nav.tabs.channels")}{" · "}{page("search/landings", "nav.tabs.landings")}</p>
      {links("search", "google")}
    </>),
    indexing: section("indexing", "helpUsage.indexingIntro", <>
      {steps(["helpUsage.indexingDiscover", "helpUsage.indexingInspect"])}
      <Warn>{t("helpUsage.indexingDates")}</Warn>
      <p>{page("search/indexing", "nav.tabs.indexing")}{" · "}{page("settings/schedule", "nav.schedule")}</p>
      {links("google", "search")}
    </>),
    opportunities: section("opportunities", "helpUsage.actionsIntro", <>
      {steps(["helpUsage.actionsRank", "helpUsage.actionsCTR", "helpUsage.actionsState"])}
      <p>{page("opportunities", "nav.actionPlan")}</p>{links("observations", "reports", "audit")}
    </>),
    observations: section("observations", "helpUsage.observationsIntro", <>
      {steps(["helpUsage.observationsRecord", "helpUsage.observationsEvaluate"])}
      <Warn>{t("helpUsage.observationsLimits")}</Warn>
      <p>{page("opportunities", "nav.actionPlan")}</p>{links("opportunities", "reports")}
    </>),
    reports: section("reports", "helpUsage.reportsIntro", <>
      {steps(["helpUsage.reportsRead", "helpUsage.reportsExport"])}
      <Tip>{t("helpUsage.reportsLimit")}</Tip>
      <p>{page("reports", "nav.reports")}</p>{links("observations", "google")}
    </>),
    troubleshooting: <>
      <h2>{t("helpTopics.troubleshooting.label")}</h2>
      <Table head={[t("helpUsage.problem"), t("helpUsage.next")]}
        rows={issues.map(i => [t(`helpUsage.issue${i}`), t(`helpUsage.fix${i}`)])} />
      {links("google", "prompts", "observations", "access")}
    </>,
  };
}

export function usageButtons({ t }: Kit) {
  return {
    createProject: t("helpUsage.btnCreate"), buildReport: t("helpUsage.btnReport"),
    searchSync: t("helpUsage.btnSync"), syncGoogle: t("helpUsage.btnSync"),
    googleSyncNow: t("helpUsage.btnSync"), progress: t("helpUsage.btnProgress"),
    redraft: t("helpUsage.btnRedraft"),
  };
}
