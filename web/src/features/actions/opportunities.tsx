// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useMemo, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { IconCheck, IconChevronDown } from "@tabler/icons-react";
import { acceptOpportunity, dismissOpportunity, listOpportunities, patchTask, type OpportunityItem } from "../../api";
import { EmptyState } from "../../components/EmptyState";
import { EvidenceBadge } from "../../components/metrics/EvidenceBadge";
import { useI18n, type Key } from "../../i18n";
import { HelpTip } from "../../components/HelpTip";
import { useAccess } from "../../app/access";
import { ruleText } from "../../i18n/rules";
import { opportunityTitle, opportunityWhy } from "./title";

type Tab = "suggested" | "progress" | "verified" | "dismissed";
const TABS: Tab[] = ["suggested", "progress", "verified", "dismissed"];
const SOURCES = ["", "audit", "citation", "search", "metric"] as const;
const PRIORITIES = ["P0", "P1", "P2"] as const;
const STEPS = ["open", "doing", "done", "verified"] as const;
const NEXT: Record<string, { to: string; label: Key }> = {
  open: { to: "doing", label: "plan.actions.start" },
  doing: { to: "done", label: "plan.actions.markDone" },
  regressed: { to: "doing", label: "plan.actions.reopen" },
};

function tabOf(it: OpportunityItem): Tab {
  if (!it.status) return "suggested";
  if (it.status === "verified") return "verified";
  if (it.status === "dismissed") return "dismissed";
  return "progress";
}

function priorityClass(p: string) {
  if (p === "P0") return "border-red-200 bg-red-50 text-red-700";
  if (p === "P1") return "border-amber-200 bg-amber-50 text-amber-700";
  return "border-gray-200 bg-gray-50 text-gray-600";
}

// Action plan: suggestions from the audit, citations, search data and metric
// drops, and the actions you accepted with their verification status.
export function Opportunities() {
  const { slug = "" } = useParams();
  const { t, tn, locale } = useI18n();
  const { canEdit, isAdmin } = useAccess();
  const [params, setParams] = useSearchParams();
  const tab = (TABS.includes(params.get("tab") as Tab) ? params.get("tab") : "suggested") as Tab;
  const source = params.get("source") || "";
  const focus = params.get("key") || "";
  const [items, setItems] = useState<OpportunityItem[] | null>(null);
  const [dismissed, setDismissed] = useState<OpportunityItem[]>([]);
  const [noSearchEngine, setNoSearchEngine] = useState(false);
  const [open, setOpen] = useState<string>(focus);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState("");

  function load() {
    listOpportunities(slug)
      .then((d) => { setItems(d.items || []); setNoSearchEngine((d.hints || []).some((h) => h.code === "no_search_engine")); })
      .catch((e: Error) => setErr(e.message));
    listOpportunities(slug, { status: "dismissed" }).then((d) => setDismissed(d.items || [])).catch(() => setDismissed([]));
  }
  useEffect(load, [slug]);
  useEffect(() => {
    if (focus && items) document.getElementById("opp-" + focus)?.scrollIntoView({ block: "center" });
  }, [focus, items]);

  const all = useMemo(() => [...(items || []), ...dismissed], [items, dismissed]);
  const byTab = useMemo(() => {
    const m: Record<Tab, OpportunityItem[]> = { suggested: [], progress: [], verified: [], dismissed: [] };
    for (const it of all) m[tabOf(it)].push(it);
    return m;
  }, [all]);
  const inTab = byTab[tab];
  const shown = inTab.filter((it) => !source || it.source === source);
  const sourceCount = (s: string) => inTab.filter((it) => !s || it.source === s).length;

  function set(key: string, value: string) {
    const next = new URLSearchParams(params);
    if (value) next.set(key, value); else next.delete(key);
    if (key === "tab") next.delete("source");
    setParams(next, { replace: true });
  }

  async function act(key: string, fn: () => Promise<unknown>) {
    setBusy(key); setErr("");
    try { await fn(); load(); } catch (e) { setErr((e as Error).message); } finally { setBusy(""); }
  }

  const title = (it: OpportunityItem) => opportunityTitle(it, t, locale);

  const doneWhen = (a: OpportunityItem["acceptance"]) => {
    const check = a?.check || "";
    if (check.startsWith("issue.absent:")) return t("plan.checks.issue", { code: check.slice("issue.absent:".length) });
    if (check.startsWith("metrics.prompt_up:")) return t("plan.checks.prompt");
    if (check.startsWith("metrics.visibility_not_down:")) return t("plan.checks.notDown");
    return t("plan.checks.manual");
  };

  const suggested = byTab.suggested;
  const urgent = suggested.filter((it) => it.priority === "P0").length;
  const awaiting = byTab.progress.filter((it) => it.status === "done").length;
  const regressed = byTab.progress.filter((it) => it.status === "regressed").length;

  // Tips go on the first row only, so a long list is not full of "?".
  const firstKey = (tab === "suggested" ? PRIORITIES.map((p) => shown.find((x) => x.priority === p)).find(Boolean) : shown[0])?.key;

  function Row({ it }: { it: OpportunityItem }) {
    const tips = it.key === firstKey;
    const expanded = open === it.key;
    const variants = Array.isArray(it.detail?.variants) ? (it.detail!.variants as string[]) : [];
    const why = opportunityWhy(it, t, locale);
    const fix = it.source === "audit" ? ruleText(locale, it.kind, "fix", it.fix) : it.fix;
    return (
      <li id={"opp-" + it.key} className="card overflow-hidden">
        <div className="flex items-start gap-3 px-4 py-3">
          <span className={`mt-0.5 rounded-md border px-1.5 text-xs font-semibold ${priorityClass(it.priority)}`}>{it.priority}</span>
          <button type="button" className="min-w-0 flex-1 text-left" onClick={() => setOpen(expanded ? "" : it.key)} aria-expanded={expanded}>
            <span className="block text-sm font-medium text-gray-900">{title(it)}</span>
            <span className="mt-1 flex flex-wrap items-center gap-2 text-xs text-gray-500">
              <span>{t(`plan.sources.${it.source}` as Key)}</span>
              <EvidenceBadge level={it.evidence} refs={it.refs} />
              {(it.urls?.length ?? 0) > 0 && <span>{tn("common.pages", it.urls!.length)}</span>}
              {variants.length > 0 && <span>{tn("plan.similar", variants.length)}</span>}
              {it.task_code && <span className="font-mono">{it.task_code}</span>}
            </span>
          </button>
          <div className="flex shrink-0 items-center gap-2">
            {canEdit && !it.status && (
              <>
                <button type="button" disabled={busy === it.key} className="btn btn-primary btn-sm" onClick={() => act(it.key, () => acceptOpportunity(slug, it.key, expanded))}>{t("plan.actions.accept")}</button>
                <button type="button" disabled={busy === it.key} className="btn btn-ghost btn-sm" onClick={() => act(it.key, () => dismissOpportunity(slug, it.key))}>{t("plan.actions.dismiss")}</button>
                {tips && <><HelpTip id="accept" /><HelpTip id="dismiss" /></>}
              </>
            )}
            {canEdit && it.status && it.task_code && NEXT[it.status] && (
              <button type="button" disabled={busy === it.key} className="btn btn-primary btn-sm" onClick={() => act(it.key, () => patchTask(slug, it.task_code!, NEXT[it.status!].to))}>{t(NEXT[it.status].label)}</button>
            )}
            {tips && canEdit && it.status && it.task_code && NEXT[it.status] && <HelpTip id="progress" />}
            {canEdit && it.status === "dismissed" && it.task_code && (
              <button type="button" disabled={busy === it.key} className="btn btn-secondary btn-sm" onClick={() => act(it.key, () => patchTask(slug, it.task_code!, "open"))}>{t("plan.actions.restore")}</button>
            )}
            {tips && canEdit && it.status === "dismissed" && it.task_code && <HelpTip id="restore" />}
            <button type="button" className="btn btn-ghost btn-icon p-1.5" onClick={() => setOpen(expanded ? "" : it.key)} aria-label={expanded ? t("plan.actions.hideDetails") : t("plan.actions.showDetails")}>
              <IconChevronDown size={16} className={"transition-transform " + (expanded ? "rotate-180" : "")} />
            </button>
          </div>
        </div>
        {it.status && it.status !== "dismissed" && <Progress status={it.status} />}
        {expanded && (
          <div className="space-y-3 border-t border-gray-100 bg-gray-50/60 px-4 py-3 text-sm text-gray-700">
            {why && <p><span className="font-medium text-gray-900">{t("plan.why")}. </span>{why}</p>}
            {fix && <p><span className="font-medium text-gray-900">{t("plan.how")}. </span>{fix}</p>}
            <p><span className="font-medium text-gray-900">{t("plan.doneWhen")}. </span>{doneWhen(it.acceptance)}</p>
            {variants.length > 0 && <p><span className="font-medium text-gray-900">{t("plan.alsoCovers")}. </span>{variants.join(", ")}</p>}
            <div className="flex flex-wrap gap-3">
              {it.qid && <Link to={`/p/${slug}/ai/prompts/${it.qid}`}>{t("plan.openQuestion")}</Link>}
              {it.source === "audit" && <Link to={`/p/${slug}/audit/issues`}>{t("plan.seeIssues")}</Link>}
            </div>
            {(it.urls?.length ?? 0) > 0 && (
              <ul className="list-disc space-y-0.5 pl-5 text-xs text-gray-600">
                {it.urls!.slice(0, 20).map((u) => <li key={u} className="break-all">{u}</li>)}
                {it.urls!.length > 20 && <li>{t("plan.moreUrls", { n: it.urls!.length - 20 })}</li>}
              </ul>
            )}
          </div>
        )}
      </li>
    );
  }

  function Progress({ status }: { status: string }) {
    const at = status === "regressed" ? 3 : STEPS.indexOf(status as (typeof STEPS)[number]);
    return (
      <ol className="flex items-center gap-2 border-t border-gray-100 px-4 py-2 text-xs">
        {STEPS.map((s, i) => {
          const done = i < at || (i === at && s === "verified");
          const cur = i === at;
          const label = s === "verified" && status === "regressed" ? t("plan.steps.regressed") : t(`plan.steps.${s}` as Key);
          return (
            <li key={s} className="flex items-center gap-2">
              {i > 0 && <span className={"h-px w-6 " + (i <= at ? "bg-primary-400" : "bg-gray-200")} />}
              <span className={"inline-flex items-center gap-1 rounded-full px-2 py-0.5 " + (status === "regressed" && s === "verified" ? "bg-red-50 text-red-700" : cur ? "bg-primary-50 font-medium text-primary-700" : done ? "text-primary-700" : "text-gray-400")}>
                {done && <IconCheck size={12} stroke={2.4} />}{label}
              </span>
            </li>
          );
        })}
      </ol>
    );
  }

  return (
    <section className="space-y-5">
      <p className="page-description">{t("plan.description")}</p>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <button type="button" className={"stat-card text-left " + (tab === "suggested" ? "ring-2 ring-primary-200" : "")} onClick={() => set("tab", "suggested")}>
          <div><div className="stat-label">{t("plan.tiles.suggested")}</div><div className="stat-value">{suggested.length}</div>
            {urgent > 0 && <div className="mt-1 text-xs text-red-700">{t("plan.tiles.urgent", { n: urgent })}</div>}</div>
        </button>
        <button type="button" className={"stat-card text-left " + (tab === "progress" ? "ring-2 ring-primary-200" : "")} onClick={() => set("tab", "progress")}>
          <div><div className="stat-label">{t("plan.tiles.inProgress")}</div><div className="stat-value">{byTab.progress.length}</div>
            {awaiting > 0 && <div className="mt-1 text-xs text-gray-500">{t("plan.tiles.awaiting", { n: awaiting })}</div>}</div>
        </button>
        <button type="button" className={"stat-card text-left " + (tab === "verified" ? "ring-2 ring-primary-200" : "")} onClick={() => set("tab", "verified")}>
          <div><div className="stat-label">{t("plan.tiles.verified")}</div><div className="stat-value">{byTab.verified.length}</div>
            {regressed > 0 && <div className="mt-1 text-xs text-red-700">{t("plan.tiles.regressed", { n: regressed })}</div>}</div>
        </button>
      </div>

      {noSearchEngine && tab === "suggested" && (
        <div className="alert alert-info my-0 flex flex-wrap items-center justify-between gap-3">
          <span>{t("plan.hintNoSearch")}</span>
          {isAdmin && <Link className="btn btn-secondary btn-sm" to={`/p/${slug}/settings/providers?connect=perplexity`}>{t("plan.connectPerplexity")}</Link>}
        </div>
      )}

      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="tabs">
          {TABS.map((x) => (
            <button key={x} type="button" className={"tab " + (tab === x ? "tab-active" : "")} onClick={() => set("tab", x)}>
              {t(`plan.tabs.${x}` as Key)} <span className="ml-1 text-xs text-gray-400">{byTab[x].length}</span>
            </button>
          ))}
        </div>
        <div className="flex flex-wrap gap-1.5">
          {SOURCES.map((s) => (
            <button key={s || "all"} type="button" className={"chip py-1 text-xs " + (source === s ? "chip-active" : "")} onClick={() => set("source", s)}>
              {t(`plan.sources.${s || "all"}` as Key)} <span className="text-gray-400">{sourceCount(s)}</span>
            </button>
          ))}
        </div>
      </div>

      {err && <div className="alert alert-error">{err}</div>}
      {items === null && !err && <p className="text-sm text-gray-500">{t("common.loading")}</p>}
      {items && shown.length === 0 && <EmptyState title={t(`plan.tabs.${tab}` as Key)}>{t(`plan.empty.${tab}` as Key)}</EmptyState>}

      {tab === "suggested"
        ? PRIORITIES.map((p) => {
          const rows = shown.filter((it) => it.priority === p);
          if (rows.length === 0) return null;
          return (
            <div key={p}>
              <h2 className="mb-2 text-sm font-semibold text-gray-700">{t(`plan.groups.${p}` as Key)} <span className="font-normal text-gray-400">{rows.length}</span></h2>
              <ul className="space-y-2">{rows.map((it) => <Row key={it.key} it={it} />)}</ul>
            </div>
          );
        })
        : <ul className="space-y-2">{shown.map((it) => <Row key={it.key} it={it} />)}</ul>}
    </section>
  );
}
