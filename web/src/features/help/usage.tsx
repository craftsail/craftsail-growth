// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ReactNode } from "react";
import type { Key } from "../../i18n";
import { Table, Tip, Warn, type Kit, type TopicId } from "./kit";

// Shared structure keeps the three guides aligned; all prose lives in i18n.
export function usageBodies({ t, n, page, topic }: Kit) {
  const names = {
    projects: n("nav.projects"), schedule: n("nav.schedule"),
    google: n("nav.google"), search: n("nav.search"),
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

export function usageButtons({ t }: Kit) {
  return {
    createProject: t("helpUsage.btnCreate"), buildReport: t("helpUsage.btnReport"),
    searchSync: t("helpUsage.btnSync"), syncGoogle: t("helpUsage.btnSync"),
    googleSyncNow: t("helpUsage.btnSync"), progress: t("helpUsage.btnProgress"),
    redraft: t("helpUsage.btnRedraft"),
  };
}
