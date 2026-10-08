// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useState, type FormEvent } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { exportGACSV, getGAExplore, type GAExplore, type GoogleQuality, type GrainCoverage } from "../../api";
import { useI18n } from "../../i18n";

const metrics = ["sessions", "engaged", "engagement_rate", "key_events", "duration", "duration_per_session", "sessions_change"] as const;
const qualityFlags = ["sampled", "thresholded", "other_row", "restricted", "empty_reason"] as const;
function trustworthy(quality: GoogleQuality) { return quality.known && !qualityFlags.some(flag => quality[flag]); }

export function GAExplorer({ report }: { report: "channel" | "landing" }) {
  const { slug } = useParams();
  const { t, tn, num } = useI18n();
  const [params, setParams] = useSearchParams();
  const query = params.toString();
  const [data, setData] = useState<GAExplore | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [exporting, setExporting] = useState(false);
  const [refresh, setRefresh] = useState(0);
  useEffect(() => {
    let stale = false;
    setData(null); setError(""); setLoading(true);
    if (slug) getGAExplore(slug, report, new URLSearchParams(query)).then(value => { if (!stale) setData(value); })
      .catch((err: Error) => { if (!stale) setError(err.message); }).finally(() => { if (!stale) setLoading(false); });
    return () => { stale = true; };
  }, [slug, report, query, refresh]);
  function apply(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const next = new URLSearchParams();
    if (params.has("lang")) next.set("lang", params.get("lang")!);
    new FormData(event.currentTarget).forEach((v, k) => { if (String(v)) next.set(k, String(v)); });
    setParams(next);
  }
  function resolved() {
    const next = new URLSearchParams(params);
    if (data) { next.set("from", data.filters.from); next.set("through", data.filters.through); }
    return next;
  }
  function page(n: number) { const next = resolved(); next.set("page", String(n)); setParams(next); }
  async function exportRows() {
    if (!slug) return;
    setExporting(true); setError("");
    try {
      const blob = await exportGACSV(slug, report, resolved());
      const href = URL.createObjectURL(blob); const link = document.createElement("a");
      link.href = href; link.download = `ga-${report}.csv`; link.click(); setTimeout(() => URL.revokeObjectURL(href), 1000);
    } catch (err) { setError((err as Error).message); } finally { setExporting(false); }
  }
  const pages = data ? Math.max(1, Math.ceil(data.total / data.page_size)) : 1;
  const number = (value: number | null) => value == null ? "—" : num(value, { maximumFractionDigits: 2 });
  return <section>
    <p className="page-description mb-5">{t(`search.ga.${report}Description`)}</p>
    <form key={`${report}/${query}/${data?.filters.from || ""}`} onSubmit={apply} className="mb-5 flex flex-wrap items-end gap-3">
      {(["from", "through"] as const).map(key => <label key={key} className="flex flex-col text-sm text-gray-700">{t(`search.explore.${key}`)}<input className="input mt-1" type="date" name={key} defaultValue={params.get(key) || data?.filters[key] || ""} /></label>)}
      <label className="flex flex-col text-sm text-gray-700">{t("search.explore.contains")}<input className="input mt-1" name="q" maxLength={1000} defaultValue={params.get("q") || ""} /></label>
      <label className="flex flex-col text-sm text-gray-700">{t("search.explore.sort")}<select className="input mt-1" name="sort" defaultValue={params.get("sort") || "sessions"}>{metrics.map(key => <option key={key} value={key}>{t(`search.ga.${key}`)}</option>)}</select></label>
      <label className="flex flex-col text-sm text-gray-700">{t("search.explore.direction")}<select className="input mt-1" name="direction" defaultValue={params.get("direction") || "desc"}><option value="desc">{t("search.explore.desc")}</option><option value="asc">{t("search.explore.asc")}</option></select></label>
      <label className="flex flex-col text-sm text-gray-700">{t("search.explore.pageSize")}<select className="input mt-1" name="page_size" defaultValue={params.get("page_size") || "50"}>{[25, 50, 100, 200].map(n => <option key={n} value={n}>{num(n)}</option>)}</select></label>
      <button className="btn btn-primary" disabled={loading}>{t("search.explore.apply")}</button>
      <button type="button" className="btn btn-secondary" onClick={() => setParams(params.has("lang") ? { lang: params.get("lang")! } : {})}>{t("search.explore.reset")}</button>
    </form>
    {error && <p role="alert" className="alert alert-error">{error}</p>}
    {loading && <p role="status" className="hint">{t("common.loading")}</p>}
    {data && <>
      <p className="mb-3 break-all text-sm text-gray-500">{t("search.ga.context", { property: data.property || "—", timezone: data.timezone || "—" })}</p>
      {["missing", "unsupported", "failed", "partial", "paused"].includes(data.report_state) && <p className="mb-4 text-sm text-amber-700">{t(`search.ga.${data.report_state === "missing" ? "notReady" : data.report_state === "unsupported" ? "unsupported" : "failed"}`)} <Link className="text-primary-700 underline" to={`/p/${slug}/search`}>{t("search.grain.viewSync")}</Link></p>}
      <div className="mb-4 grid min-w-0 gap-4 md:grid-cols-2">
        <Period coverage={data.coverage} quality={data.quality} title={t("search.ga.current")} />
        <Period coverage={data.previous_coverage} quality={data.previous_quality} title={t("search.ga.previousPeriod")} />
      </div>
      {!data.comparable && <p className="mb-4 text-sm text-amber-700">{t("search.ga.noComparison")}</p>}
      <div className="mb-3 flex flex-wrap items-center gap-3"><span className="text-sm text-gray-700">{tn("search.explore.results", data.total)}</span><button className="btn btn-secondary" onClick={exportRows} disabled={exporting}>{t("search.explore.exportAll")}</button><button className="btn btn-secondary" onClick={() => setRefresh(n => n + 1)}>{t("common.refresh")}</button></div>
      <div className="table-container"><table className="table min-w-[1000px]">
        <thead><tr>{(report === "channel" ? ["channel", "source", "medium"] as const : ["landing"] as const).map(key => <th key={key}>{t(`search.ga.${key}`)}</th>)}<th>{t("search.ga.sessions")}</th><th>{t("search.ga.previous")}</th><th>{t("search.ga.sessions_change")}</th>{metrics.slice(1, -1).map(key => <th key={key}>{t(`search.ga.${key}`)}</th>)}</tr></thead>
        <tbody>{(data.items || []).map(row => {
          const measured = row.current_rows > 0 || (data.coverage.state === "covered" && trustworthy(data.quality));
          const previousMeasured = row.previous_rows > 0 || (data.previous_coverage.state === "covered" && trustworthy(data.previous_quality));
          return <tr key={JSON.stringify([row.channel, row.source, row.medium, row.landing])}>
            {report === "channel" ? <><td>{row.channel || "—"}</td><td className="break-all">{row.source || "—"}</td><td>{row.medium || "—"}</td></> : <td className="max-w-sm break-all">{row.landing || "—"}</td>}
            <td>{measured ? number(row.sessions) : "—"}</td><td>{previousMeasured ? number(row.previous_sessions) : "—"}</td><td>{data.comparable ? num(row.sessions_change, { signDisplay: "exceptZero" }) : "—"}</td>
            <td>{measured ? number(row.engaged) : "—"}</td><td>{measured && row.engagement_rate != null ? num(row.engagement_rate, { style: "percent", maximumFractionDigits: 2 }) : "—"}</td><td>{measured ? number(row.key_events) : "—"}</td><td>{measured ? number(row.duration) : "—"}</td><td>{measured ? number(row.duration_per_session) : "—"}</td>
          </tr>;
        })}{!data.items?.length && <tr><td colSpan={report === "channel" ? 11 : 9}>{t("search.explore.noResults")}</td></tr>}</tbody>
      </table></div>
      <nav className="mt-4 flex flex-wrap items-center gap-3" aria-label={t("search.explore.pagination")}><button className="btn btn-secondary" disabled={data.page <= 1} onClick={() => page(data.page - 1)}>{t("search.explore.previous")}</button><span className="text-sm text-gray-700">{t("search.explore.page", { page: num(data.page), total: num(pages) })}</span><button className="btn btn-secondary" disabled={data.page >= pages} onClick={() => page(data.page + 1)}>{t("search.explore.next")}</button></nav>
    </>}
  </section>;
}

function Period({ coverage, quality, title }: { coverage: GrainCoverage; quality: GoogleQuality; title: string }) {
  const { t, tn } = useI18n();
  const { slug } = useParams();
  return <div className="min-w-0 rounded-lg border border-gray-200 bg-gray-50 p-4 text-sm text-gray-700">
    <p className="font-semibold">{title}</p><p className="mt-1">{coverage.from} – {coverage.through} · {tn("search.imports.days", coverage.covered_days, { total: coverage.total_days })}</p>
    {coverage.state !== "covered" && <p className="mt-2 text-amber-700">{t(coverage.state === "missing" ? "search.grain.missing" : "search.grain.partial")} <Link className="text-primary-700 underline" to={`/p/${slug}/search`}>{t("search.grain.viewSync")}</Link></p>}
    <p className="mt-2 text-gray-500">{t("search.ga.quality")}</p>
    {!quality.known && <p className="text-amber-700">{t("search.quality.unknown")}</p>}
    {qualityFlags.filter(flag => quality[flag]).map(flag => <p key={flag} className="text-amber-700">{t(`search.quality.${flag}`)}</p>)}
    {trustworthy(quality) && <p>{t("search.quality.noFlags")}</p>}
    <p className="break-all text-gray-500">{quality.time_zones?.join(" · ")}</p>
  </div>;
}
