// SPDX-License-Identifier: AGPL-3.0-or-later

import { useState } from "react";
import { Link } from "react-router-dom";
import { AreaSeries, DonutSeries, LineSeries, StackSeries } from "../../components/charts/series";
import { TermBars, WordCloud } from "../../components/charts/terms";
import {
  keepSearch,
  pctText,
  shareColors,
  shareText,
  useMeasure,
  withDay,
  type MeasureView,
  type PromptChart,
} from "./data";
import { FilterBar, Help } from "./filters";
import { useI18n, type Key } from "../../i18n";

const CAT: Record<string, string> = {
  brand: "#2563eb", competitor: "#f59e0b", social: "#8b5cf6", reviews: "#ec4899", reference: "#14b8a6",
  ecommerce: "#f97316", pr: "#84cc16", institutional: "#64748b", other: "#94a3b8",
};

const PAGE: Record<string, string> = {
  homepage: "#2563eb", article: "#14b8a6", comparison: "#f59e0b", review: "#ec4899", howto: "#8b5cf6",
  forum: "#f97316", product: "#84cc16", doc: "#64748b", video: "#0ea5e9", other: "#94a3b8",
};

// metaFor labels chart keys in the interface language; unknown keys keep
// their raw name and a neutral color.
function metaFor(keys: string[], colors: Record<string, string>, t: (k: Key) => string, ns: "cats" | "pages") {
  const meta: Record<string, { label: string; color: string }> = {};
  keys.forEach((key, index) => {
    meta[key] = colors[key]
      ? { label: t(`measure.${ns}.${key}` as Key), color: colors[key] }
      : { label: key, color: ["#94a3b8", "#64748b", "#cbd5e1"][index % 3] };
  });
  return meta;
}

function Status({ err, loading, data }: { err: string; loading: boolean; data: MeasureView | null }) {
  const { t } = useI18n();
  if (err) return <p className="text-sm text-red-700">{err}</p>;
  if (loading && !data) return <p className="text-sm text-gray-500">{t("measure.loadingSamples")}</p>;
  return null;
}



function RateNote({ data }: { data: MeasureView | null }) {
  const { t } = useI18n();
  if (!data || data.visibility == null) return null;
  return (
    <div className="mt-1 space-y-0.5 text-center text-xs text-gray-500">
      {data.visibility_ci && (
        <p>
          {t("measure.ci", { lo: data.visibility_ci.lo.toFixed(0), hi: data.visibility_ci.hi.toFixed(0) })} · n={data.visibility_n} · {data.access === "web" ? "Web" : "API"}
          {data.low_sample && <span className="ml-1 rounded border border-gray-300 px-1 text-gray-600">{t("measure.smallSample")}</span>}
        </p>
      )}
      {data.recognition != null && <p>{t("measure.recognition", { v: data.recognition.toFixed(0), n: data.recognition_n ?? 0 })}</p>}
    </div>
  );
}

function visBadge(value: number) {
  if (value >= 75) return "bg-emerald-600 text-white";
  if (value >= 45) return "bg-amber-500 text-white";
  return "bg-rose-600 text-white";
}

function PromptCard({ slug, search, data, prompt }: { slug: string; search: string; data: MeasureView; prompt: PromptChart }) {
  const { t } = useI18n();
  const colors = shareColors((data.leaders || []).map((row) => ({ name: row.name, isBrand: row.is_brand, mentions: row.mentions })));
  const names = new Set<string>();
  (prompt.series ?? []).forEach((row) => {
    Object.keys(row).forEach((key) => {
      if (key !== "date" && key !== "brand") names.add(key);
    });
  });
  const series = [
    { key: "brand", name: data.brand || t("measure.brand"), color: colors.get(data.brand) || "#2563eb" },
    ...[...names].map((name) => ({ key: name, name, color: colors.get(name) || "#94a3b8" })),
  ];
  return (
    <article className="card p-5">
      <div className="mb-3 flex items-start justify-between gap-3">
        <h3 className="text-base font-medium text-gray-900">{prompt.text || prompt.id}</h3>
        {prompt.runs > 0 ? <span className={`shrink-0 rounded-full px-2 py-0.5 text-xs ${visBadge(prompt.visibility)}`}>{t("measure.visibilityPct", { v: pctText(prompt.visibility) })}</span> : <span className="shrink-0 text-xs text-gray-500">{t("measure.failedNotCounted", { n: prompt.failed })}</span>}
      </div>
      <LineSeries rows={withDay(prompt.series ?? [])} series={series} />
      <div className="mt-3">
        <Link className="text-sm no-underline" to={`/p/${slug}/ai/prompts/${prompt.id}${keepSearch(search, { tab: "mentions" })}`}>
          {t("measure.viewDetails")}
        </Link>
      </div>
    </article>
  );
}

export function MeasureVisibility() {
  const m = useMeasure();
  const { t, num } = useI18n();
  const charts = (m.data?.prompt_charts ?? []).filter((prompt) => !m.q || (prompt.text || "").includes(m.q) || prompt.id.includes(m.q));
  return (
    <div className="space-y-4">
      <p className="page-description mb-5">{t("measure.visDescription")}</p>
      <FilterBar {...m} onChange={m.setFilter} showSort showSearch />
      <Status err={m.err} loading={m.loading} data={m.data} />
      <div className={`flex flex-wrap items-end justify-between gap-3 rounded-xl border px-4 py-3 border-gray-200 bg-white`}>
        <div>
          <div className="flex items-center gap-1 text-sm text-gray-600">{t("overview.visibility")} <Help text={t("notes.visibility")} /></div>
          <div className={`text-4xl font-semibold tabular-nums ${m.data?.visibility != null ? "text-gray-900" : "text-gray-400"}`}>{pctText(m.data?.visibility ?? null)}</div>
          <RateNote data={m.data} />
        </div>
        <div className="text-sm text-gray-600">
          {t("measure.totals", { q: num(m.data?.prompts ?? 0), r: num(m.data?.runs ?? 0), c: num(m.data?.citations ?? 0) })}
        </div>
      </div>
      <div className="space-y-3">
        {charts.map((prompt) => (
          <PromptCard key={prompt.id} slug={m.slug} search={m.search} data={m.data!} prompt={prompt} />
        ))}
        {!m.loading && charts.length === 0 && <p className="text-sm text-gray-500">{t("measure.noSamples")}</p>}
      </div>
    </div>
  );
}

export function MeasureShare() {
  const m = useMeasure();
  const { t } = useI18n();
  const leaders = m.data?.leaders ?? [];
  const colors = shareColors(leaders.map((row) => ({ name: row.name, isBrand: row.is_brand, mentions: row.mentions })));
  const slices = leaders.filter((row) => row.mentions > 0).map((row) => ({
    name: row.name,
    value: row.mentions,
    color: colors.get(row.name) || "#94a3b8",
    tip: `${row.name} ${shareText(row.share)}`,
  }));
  return (
    <div className="space-y-4">
      <p className="page-description mb-5">{t("measure.sovDescription")}</p>
      <FilterBar {...m} onChange={m.setFilter} />
      <Status err={m.err} loading={m.loading} data={m.data} />
      <div className="grid gap-3 lg:grid-cols-2">
        <div className="card p-5">
          <div className="mb-2 flex items-center gap-1 text-sm text-gray-600">{t("overview.sov")} <Help text={t("notes.share")} /></div>
          <div className="flex items-center gap-4">
            <div>
              <div className={`text-5xl font-semibold tabular-nums ${m.data?.share == null ? "text-gray-400" : "text-gray-900"}`}>{pctText(m.data?.share ?? null)}</div>
              <p className="mt-1 text-sm text-gray-500">{t("measure.sovSummary", { brand: m.data?.brand || t("measure.brand"), runs: m.data?.runs ?? 0, comps: Math.max(0, leaders.length - 1) })}</p>
            </div>
            <div className="h-36 w-36 shrink-0">{slices.length > 0 && <DonutSeries slices={slices} height={144} />}</div>
          </div>
        </div>
        <div className="card p-5">
          <div className="mb-2 text-sm text-gray-600">{t("measure.sovTrend")}</div>
          <AreaSeries label={t("overview.sov")} points={(m.data?.share_series ?? []).map((point) => ({ day: point.date, value: point.value }))} color="#e11d48" height={144} />
        </div>
      </div>
      <div className="table-container">
        <table className="table min-w-[640px]">
          <thead className="text-left text-gray-500">
            <tr>
              <th className="px-3 py-2 font-medium">#</th>
              <th className="px-3 py-2 font-medium">{t("measure.brand")}</th>
              <th className="px-3 py-2 font-medium">{t("measure.mentions")} <Help text={t("notes.mentions")} /></th>
              <th className="px-3 py-2 font-medium">{t("measure.share")} <Help text={t("notes.shareCol")} /></th>
              <th className="px-3 py-2 font-medium">{t("measure.questions")} <Help text={t("notes.questionsCol")} /></th>
            </tr>
          </thead>
          <tbody>
            {leaders.map((row, index) => (
              <tr key={row.name} className="border-t border-gray-100">
                <td className="px-3 py-2">{index + 1}</td>
                <td className="px-3 py-2">
                  <span className="mr-2 inline-block h-2 w-8 rounded-full align-middle" style={{ background: colors.get(row.name) || "#cbd5e1", width: `${Math.max(8, row.share)}%`, maxWidth: 96 }} />
                  {row.name}
                  {row.is_brand && <span className="badge badge-primary ml-2">{t("measure.you")}</span>}
                </td>
                <td className="px-3 py-2 tabular-nums">{row.mentions}</td>
                <td className="px-3 py-2 tabular-nums">{shareText(row.share)}</td>
                <td className="px-3 py-2 tabular-nums">{row.prompts}</td>
              </tr>
            ))}
            {!m.loading && leaders.length === 0 && (
              <tr><td className="px-3 py-4 text-gray-500" colSpan={5}>{t("measure.noMentions")}</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

export function MeasureFanout() {
  const m = useMeasure();
  const { t } = useI18n();
  const [tab, setTab] = useState<"added" | "preserved" | "dropped">("added");
  const words = m.data?.[tab] ?? [];
  const cards = [
    ["runs", t("measure.fanRuns"), m.data?.fanout_total ?? 0, t("notes.fanRuns")],
    ["unknown", t("measure.fanUnknown"), m.data?.fanout_unknown ?? 0, t("notes.fanUnknown")],
    ["known", t("measure.fanKnown"), m.data?.fanout_known ?? 0, t("notes.fanKnown")],
    ["avg", t("measure.fanAvg"), m.data?.fanout_avg ?? 0, t("notes.fanAvg")],
  ] as const;
  return (
    <div className="space-y-4">
      <p className="page-description mb-5">{t("measure.fanDescription")}</p>
      <FilterBar {...m} onChange={m.setFilter} />
      <Status err={m.err} loading={m.loading} data={m.data} />
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        {cards.map(([id, label, value, note]) => (
          <div key={id} className="card p-4">
            <div className="flex items-center gap-1 text-sm text-gray-500">{label} <Help text={note} /></div>
            <div className="stat-value mt-2 tabular-nums">{typeof value === "number" && id === "avg" ? value.toFixed(1) : value}</div>
          </div>
        ))}
      </div>
      {(m.data?.words?.length ?? 0) > 0 ? (
        <div className="card p-5">
          <WordCloud terms={(m.data?.words ?? []).map((word) => ({ term: word.word, count: word.count }))} />
        </div>
      ) : (
        !m.loading && <p className="text-sm text-gray-500">{t("measure.noCloud")}</p>
      )}
      <div className="card p-5">
        <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
          <div className="text-sm font-medium text-gray-800">{t("measure.queryWords")} <Help text={t("notes.words")} /></div>
          <div className="tabs">
            {(["added", "preserved", "dropped"] as const).map((key) => (
              <button key={key} type="button" className={`tab ${tab === key ? "tab-active" : ""}`} onClick={() => setTab(key)}>
                {t(`measure.${key}` as Key)}
              </button>
            ))}
          </div>
        </div>
        {words.length > 0 ? (
          <TermBars items={words.slice(0, 12).map((word) => ({ label: word.word, count: word.count, suffix: shareText(word.share) }))} />
        ) : (
          <p className="text-sm text-gray-500">{t("measure.noWords")}</p>
        )}
      </div>
    </div>
  );
}

export function MeasureCitations() {
  const m = useMeasure();
  const { t } = useI18n();
  const catKeys = m.data?.category_keys ?? [];
  const pageKeys = m.data?.page_type_keys ?? [];
  return (
    <div className="space-y-4">
      <p className="page-description mb-5">{t("measure.citDescription")}</p>
      <FilterBar {...m} onChange={m.setFilter} />
      <Status err={m.err} loading={m.loading} data={m.data} />
      <div className="grid gap-3 sm:grid-cols-3">
        <div className="card p-4">
          <div className="flex items-center gap-1 text-sm text-gray-500">{t("measure.ownedShare")} <Help text={t("notes.citationShare")} /></div>
          <div className={`stat-value mt-2 tabular-nums ${m.data?.citation_share == null ? "text-gray-400" : ""}`}>{pctText(m.data?.citation_share ?? null)}</div>
        </div>
        <div className="card p-4">
          <div className="text-sm text-gray-500">{t("measure.uniqueDomains")}</div>
          <div className="stat-value mt-2 tabular-nums">{m.data?.unique_domains ?? 0}</div>
        </div>
        <div className="card p-4">
          <div className="text-sm text-gray-500">{t("measure.citationRows")}</div>
          <div className="stat-value mt-2 tabular-nums">{m.data?.citations ?? 0}</div>
        </div>
      </div>
      <div className="card p-5">
        <div className="mb-2 flex items-center gap-1 text-sm font-medium">{t("measure.sourceType")} <Help text={t("notes.stack")} /></div>
        <StackSeries rows={withDay(m.data?.category_series ?? [])} keys={catKeys} meta={metaFor(catKeys, CAT, t, "cats")} />
      </div>
      <div className="card p-5">
        <div className="mb-2 flex items-center gap-1 text-sm font-medium">{t("measure.pageType")} <Help text={t("notes.stack")} /></div>
        <StackSeries rows={withDay(m.data?.page_type_series ?? [])} keys={pageKeys} meta={metaFor(pageKeys, PAGE, t, "pages")} />
      </div>
      <div className="card p-5">
        <div className="mb-2 text-sm font-medium">{t("measure.gaps")}</div>
        {(m.data?.gaps ?? []).length === 0 && <p className="text-sm text-gray-500">{t("measure.noGaps")}</p>}
        <ul className="space-y-2">
          {(m.data?.gaps ?? []).map((gap) => (
            <li key={gap.qid}>
              <Link className="text-sm no-underline" to={`/p/${m.slug}/ai/prompts/${gap.qid}${keepSearch(m.search, { tab: "citations" })}`}>
                {gap.text}
              </Link>
              <span className="ml-2 text-xs text-gray-500">{gap.domains.join("、")}</span>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}


