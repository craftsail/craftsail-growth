// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ReactNode } from "react";
import { IconCircleCheck, IconCircleDashed, IconMinus } from "@tabler/icons-react";
import { itemFor } from "../../app/nav";
import type { PlaybookStatus } from "../../api";
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
  const { t, n, page, topic } = k;
  const sep = t("playbook.sep");
  // where renders an unambiguous page link: when the step's page is a tab,
  // prefix it with the page that owns the tab, since tab labels alone
  // (e.g. "Pages") are not unique across pages.
  const where = (s: (typeof STEPS)[number]) => {
    const parentLabel = s.page.label.startsWith("nav.tabs.") ? itemFor("/p/_/" + s.page.path)?.label : undefined;
    return <>{parentLabel && <>{n(parentLabel)}{" › "}</>}{page(s.page.path, s.page.label)}</>;
  };
  const { status, error, canEdit, setStage, confirm } = k.playbook;
  const order: Record<PlaybookTopic, number> = { start: 0, seoNew: 1, seoGrow: 2, aiReach: 1, aiKnown: 2, aiRecommend: 3, weekly: 0 };
  const stageOf = (id: PlaybookTopic): string | undefined =>
    id.startsWith("seo") ? status?.seo_stage : id.startsWith("ai") ? status?.ai_stage : undefined;
  // banner says where the project is relative to this stage topic.
  const banner = (id: PlaybookTopic) => {
    const cur = stageOf(id);
    if (!status || order[id] === 0) return null;
    if (id.startsWith("seo") && !cur) return <Tip>{t("playbook.status.noSite")}</Tip>;
    const rel = order[id] < order[cur as PlaybookTopic] ? "passed" : order[id] === order[cur as PlaybookTopic] ? "current" : "upcoming";
    const cls = rel === "current" ? "bg-primary-100 text-primary-700" : rel === "passed" ? "bg-emerald-100 text-emerald-700" : "bg-gray-100 text-gray-700";
    return <p><span className={"rounded-full px-2.5 py-0.5 text-xs font-medium " + cls}>{t(`playbook.status.${rel}`)}</span></p>;
  };
  const s0 = (id: PlaybookTopic) =>
    status?.product_stage === "s0" && id !== "start" && id !== "weekly" ? <Warn>{t("playbook.status.s0Warn")}</Warn> : null;
  // established: search data already qualified the property before every
  // signal passed, so seoNew explains why it shows as passed anyway.
  const established = (id: PlaybookTopic) =>
    id === "seoNew" && status && status.seo_stage === "seoGrow" && status.seo_met < status.seo_needed
      ? <p>{t("playbook.status.established")}</p> : null;
  const mark = (check?: string) => {
    const c = check ? status?.checks[check] : undefined;
    if (!c) return null;
    const label = t(c.state === "done" ? "playbook.status.done" : c.state === "todo" ? "playbook.status.todo" : "playbook.status.noData");
    const Icon = c.state === "done" ? IconCircleCheck : c.state === "todo" ? IconCircleDashed : IconMinus;
    const color = c.state === "done" ? "text-emerald-700" : "text-gray-400";
    return <span className={"mr-1.5 inline-flex items-center gap-1 align-middle text-xs font-medium " + color}><Icon size={16} aria-hidden />{label}</span>;
  };
  const steps = (id: PlaybookTopic) => {
    const list = STEPS.filter((s) => s.topic === id);
    return CADENCES.filter((c) => list.some((s) => s.cadence === c)).map((c) => (
      <div key={c}>
        <h3>{t(`playbook.cadence.${c}`)}</h3>
        <ol>{list.filter((s) => s.cadence === c).map((s) => (
          <li key={s.id}>
            <div>{mark("check" in s ? s.check : undefined)}<strong>{t(`playbook.steps.${s.id}.do`, THRESHOLDS)}</strong></div>
            <div>{t("playbook.why")}{sep}{t(`playbook.steps.${s.id}.why`, THRESHOLDS)}</div>
            <div>{t("playbook.from")}{sep}<BookLinks refs={s.books} t={t} /></div>
            <div>{t("playbook.done")}{sep}{t(`playbook.steps.${s.id}.done`, THRESHOLDS)}</div>
            <div>{t("playbook.open")}{sep}{where(s)}</div>
          </li>
        ))}</ol>
      </div>
    ));
  };
  const signalNow = (id: string) => {
    const sg = status?.signals[id];
    if (!sg) return "";
    const word = t(sg.state === "met" ? "playbook.status.met" : sg.state === "unmet" ? "playbook.status.unmet" : sg.state === "manual" ? "playbook.status.manual" : "playbook.status.noData");
    const v = sg.value != null && (id === "newFast" || id === "imprCoverage" || id === "visibleShare" || id === "recognized")
      ? ` · ${t(`playbook.values.${id}`, { v: Math.round(sg.value) })}` : "";
    const manual = (sg.state === "manual" || sg.confirmed) && canEdit ? (
      <label className="ml-2 inline-flex items-center gap-1 text-xs text-gray-600">
        <input type="checkbox" checked={sg.confirmed} onChange={(e) => confirm(id, e.target.checked)} />{t("playbook.status.confirm")}
      </label>
    ) : null;
    const cls = sg.state === "met" ? "bg-emerald-100 text-emerald-700" : sg.state === "unmet" ? "bg-amber-100 text-amber-700" : "bg-gray-100 text-gray-700";
    return <><span className={"rounded-full px-2 py-0.5 text-xs font-medium " + cls}>{word}{v}</span>{manual}</>;
  };
  const pass = (id: PlaybookTopic) => {
    const list = SIGNALS.filter((s) => s.topic === id);
    if (!list.length) return null;
    const head = [t("playbook.signal"), t("playbook.rule"), t("playbook.judged")];
    if (status) head.push(t("playbook.status.now"));
    return <>
      <h3>{t("playbook.pass")}</h3>
      {status && id === "seoNew" && status.seo_stage && <p>{t("playbook.status.progress", { met: status.seo_met, total: list.length, needed: status.seo_needed })}</p>}
      <Table head={head} rows={list.map((s) => {
        const row: ReactNode[] = [t(`playbook.signals.${s.id}.name`), t(`playbook.signals.${s.id}.rule`, THRESHOLDS), s.auto ? t("playbook.auto") : t("playbook.manual")];
        if (status) row.push(signalNow(s.id));
        return row;
      })} />
      <p>{t("playbook.passNote")}</p>
    </>;
  };
  const avoid = (id: PlaybookTopic) => {
    const a = AVOID_TOPICS.find((x) => x === id);
    return a ? <Warn><strong>{t("playbook.avoid")}{sep}</strong>{t(`playbook.avoids.${a}`)}</Warn> : null;
  };
  const related = (id: PlaybookTopic) => (
    <p>{t("helpUsage.related")}{sep}{RELATED[id].map((r, i) => <span key={r}>{i > 0 && " · "}{topic(r, t(`helpTopics.${r}.label`))}</span>)}</p>
  );
  const body = (id: PlaybookTopic, extra?: ReactNode) => <>
    <h2>{t(`helpTopics.${id}.label`)}</h2>
    {error && <p className="text-sm text-red-700">{error}</p>}
    {banner(id)}
    {established(id)}
    {s0(id)}
    <p><strong>{t("playbook.ask")}{sep}</strong>{t(`playbook.asks.${id}`)}</p>
    {extra}
    {avoid(id)}
    {steps(id)}
    {pass(id)}
    <p className="text-gray-500">{t("playbook.bookNote")}</p>
    {related(id)}
  </>;
  const stages = <>
    {status && <>
      <h3>{t("playbook.status.yours")}</h3>
      <p>
        {t("playbook.status.productStage")}{sep}
        {canEdit ? (
          <select className="input inline-block w-auto py-1" value={status.product_stage}
            onChange={(e) => setStage(e.target.value as PlaybookStatus["product_stage"])} aria-label={t("playbook.status.productStage")}>
            <option value="">{t("playbook.status.unset")}</option>
            {(["s0", "s1", "s2"] as const).map((s) => <option key={s} value={s}>{t(`playbook.stage.${s}.name`)}</option>)}
          </select>
        ) : status.product_stage ? t(`playbook.stage.${status.product_stage}.name`) : t("playbook.status.unset")}
      </p>
      <p>
        {status.seo_stage && <>SEO{sep}{topic(status.seo_stage, t(`helpTopics.${status.seo_stage}.label`))}{" · "}</>}
        AI{sep}{topic(status.ai_stage, t(`helpTopics.${status.ai_stage}.label`))}
      </p>
    </>}
    <h3>{t("playbook.stage.title")}</h3>
    <Table head={[t("playbook.stage.stage"), t("playbook.stage.sign"), t("playbook.stage.weights"), t("playbook.stage.advice")]}
      rows={(["s0", "s1", "s2"] as const).map((s) => [
        t(`playbook.stage.${s}.name`), t(`playbook.stage.${s}.sign`), t(`playbook.stage.${s}.weights`), t(`playbook.stage.${s}.advice`),
      ])} />
    <Tip><div>{t("playbook.stage.s0Tip")}</div><div><BookLinks refs={S0_BOOKS} t={t} /></div></Tip>
    <p>{t("playbook.stage.source")}{sep}<BookLinks refs={[{ book: "formula", section: "4" }]} t={t} /></p>
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
