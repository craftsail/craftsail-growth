// SPDX-License-Identifier: AGPL-3.0-or-later

import { Link, useParams } from "react-router-dom";
import { LineSeries } from "../../components/charts/series";
import { fmtTime, pctText, shareColors, useMeasure, withDay, type Run } from "./data";
import { FilterBar } from "./filters";
import { useI18n, type Key } from "../../i18n";
import { groupLabelKey } from "../../labels";

const TABS: { id: string; label: Key }[] = [
  { id: "mentions", label: "measure.detail.mentions" },
  { id: "queries", label: "measure.detail.queries" },
  { id: "citations", label: "measure.detail.citations" },
  { id: "answers", label: "measure.detail.rawAnswer" },
];

const CATS = ["brand", "competitor", "social", "reviews", "reference", "ecommerce", "pr", "institutional", "other"];
const PAGES = ["homepage", "article", "comparison", "review", "howto", "forum", "product", "doc", "video", "other"];

export function MeasurePrompt() {
  const { qid = "" } = useParams();
  const m = useMeasure(qid);
  const { t, tn } = useI18n();
  const prompt = m.data?.prompt;
  const runs = m.data?.run_list ?? [];
  const tab = TABS.some((item) => item.id === m.tab) ? m.tab : "mentions";
  const colors = shareColors((m.data?.leaders ?? []).map((row) => ({ name: row.name, isBrand: row.is_brand, mentions: row.mentions })));
  const names = new Set<string>();
  (prompt?.series || []).forEach((row) => {
    Object.keys(row).forEach((key) => {
      if (key !== "date" && key !== "brand") names.add(key);
    });
  });
  return (
    <div className="space-y-4">
      <h2 className="text-xl font-semibold text-gray-900">{prompt?.text || qid}</h2>
      <div className="flex flex-wrap items-center gap-3 text-sm text-gray-600">
        <span className={prompt?.active ? "text-emerald-700" : "text-gray-400"}>{prompt?.active ? t("measure.detail.active") : t("measure.detail.notInLibrary")}</span>
        {prompt?.group && <span>{groupLabelKey(prompt.group) ? t(groupLabelKey(prompt.group)!) : prompt.group}</span>}
        <Link to={`/p/${m.slug}/settings/questions`}>{t("measure.detail.editQuestions")}</Link>
      </div>
      <FilterBar {...m} onChange={m.setFilter} />
      {m.err && <p className="text-sm text-red-700">{m.err}</p>}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="tabs">
          {TABS.map((item) => (
            <button key={item.id} type="button" className={`tab ${tab === item.id ? "tab-active" : ""}`} onClick={() => m.setFilter("tab", item.id)}>
              {t(item.label)}
            </button>
          ))}
        </div>
        <div className="text-sm text-gray-500">{tn("measure.detail.runs", runs.length)}</div>
      </div>
      {tab === "mentions" && (
        <div className="card p-5">
          <div className="mb-2 text-sm text-gray-500">{t("measure.detail.visHere", { v: pctText(prompt?.visibility ?? null) })}</div>
          <LineSeries
            rows={withDay(prompt?.series ?? [])}
            series={[
              { key: "brand", name: m.data?.brand || t("measure.brand"), color: "#2563eb" },
              ...[...names].map((name) => ({ key: name, name, color: colors.get(name) || "#94a3b8" })),
            ]}
          />
        </div>
      )}
      {tab === "queries" && <QueryList runs={runs} />}
      {tab === "citations" && <CitationList runs={runs} />}
      {tab === "answers" && (
        <div className="space-y-3">
          {runs.map((run) => <RunCard key={run.id} run={run} />)}
          {runs.length === 0 && !m.loading && <p className="text-sm text-gray-500">{t("measure.detail.noRunsHere")}</p>}
        </div>
      )}
    </div>
  );
}

function QueryList({ runs }: { runs: Run[] }) {
  const { t, intl } = useI18n();
  if (runs.length === 0) return <p className="text-sm text-gray-500">{t("measure.detail.noRuns")}</p>;
  return (
    <div className="space-y-2">
      {runs.map((run) => (
        <div key={run.id} className="card p-4 text-sm">
          <div className="text-gray-500">{run.platform_name} · {run.access} · {fmtTime(run.at, intl) || t("measure.detail.noRunYet")}</div>
          {!run.ok ? (
            <span className="mt-2 inline-flex rounded-full bg-orange-50 px-2 py-0.5 text-xs text-orange-700">{t("measure.detail.failed")}{run.error ? `: ${run.error}` : ""}</span>
          ) : run.unknown ? (
            <span className="mt-2 inline-flex rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600">{t("measure.detail.unknown")}</span>
          ) : (
            <ul className="mt-2 space-y-1">
              {run.queries.map((query) => <li key={query}>{query}</li>)}
            </ul>
          )}
        </div>
      ))}
    </div>
  );
}

function CitationList({ runs }: { runs: Run[] }) {
  const { t } = useI18n();
  const rows = runs.flatMap((run) => run.citations.map((cite) => ({ ...cite, run: run.id })));
  if (rows.length === 0) return <p className="text-sm text-gray-500">{t("measure.detail.noUrls")}</p>;
  return (
    <ul className="space-y-2">
      {rows.map((row) => (
        <li key={`${row.run}-${row.url}`} className="card p-4 text-sm">
          <a href={row.url} target="_blank" rel="noreferrer">{row.title || row.url}</a>
          <div className="text-xs text-gray-500">{row.domain} · {CATS.includes(row.category) ? t(`measure.cats.${row.category}` as Key) : row.category || t("measure.detail.uncategorized")} · {PAGES.includes(row.page_type) ? t(`measure.pages.${row.page_type}` as Key) : row.page_type || t("measure.detail.unlabeledPage")}</div>
        </li>
      ))}
    </ul>
  );
}

function RunCard({ run }: { run: Run }) {
  const { t, intl } = useI18n();
  return (
    <article className={`rounded-xl border bg-white p-4 ${run.ok ? "border-gray-200" : "border-orange-200"}`}>
      <div className="flex flex-wrap gap-x-4 gap-y-1 text-sm text-gray-600">
        <span>{t("measure.detail.model", { v: run.platform_name })}</span>
        <span>{t("measure.detail.access", { v: run.access })}</span>
        <span>{fmtTime(run.at, intl) || t("measure.detail.noRunYet")}</span>
        {!run.ok && <span className="text-orange-700">{t("measure.detail.failed")}{run.error ? `: ${run.error}` : ""}</span>}
      </div>
      {run.ok && (
        <div className="mt-2">
          {run.unknown ? <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600">{t("measure.detail.unknown")}</span> : <span className="text-sm text-gray-700">{run.queries.join("；")}</span>}
        </div>
      )}
      <div className="mt-2 flex flex-wrap gap-1">
        {run.chips.map((chip) => (
          <span key={chip.name} className={`rounded-full px-2 py-0.5 text-xs ${chip.self ? "bg-primary-100 text-primary-700" : "border border-gray-200 text-gray-600"}`}>{chip.name}</span>
        ))}
      </div>
      {run.answer && <p className="mt-3 whitespace-pre-wrap text-sm leading-6 text-gray-800">{run.answer}</p>}
      {run.raw && (
        <details className="mt-3">
          <summary className="cursor-pointer text-sm text-gray-500">{t("measure.detail.rawJson")}</summary>
          <pre className="mt-2 overflow-x-auto rounded-lg bg-gray-50 p-3 text-xs text-gray-700">{JSON.stringify(run.raw, null, 2)}</pre>
        </details>
      )}
    </article>
  );
}
