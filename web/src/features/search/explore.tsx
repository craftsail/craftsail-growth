// SPDX-License-Identifier: AGPL-3.0-or-later

import { CTRReferenceCard } from "./ctr-reference";
import { useEffect, useState, type FormEvent } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { exportSearchCSV, getSavedKeywords, getSearchDetail, getSearchExplore, saveKeyword, type SearchDetail, type SearchExplore, type SearchMetric } from "../../api";
import { useI18n } from "../../i18n";
import { useAccess } from "../../app/access";
import { TimeSeries } from "../../components/charts/series";
import { ReportCoverage } from "./report-coverage";

type Kind = "query" | "page";

export function SearchExplorer({ kind }: { kind: Kind }) {
  const { slug } = useParams();
  const [params, setParams] = useSearchParams();
  const query = params.toString();
  const value = params.get("value") || "";
  const { t, tn, num } = useI18n();
  const { canEdit } = useAccess();
  const [data, setData] = useState<SearchExplore | null>(null);
  const [detail, setDetail] = useState<SearchDetail | null>(null);
  const [saved, setSaved] = useState<Set<string>>(new Set());
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [exporting, setExporting] = useState(false);
  const [refresh, setRefresh] = useState(0);

  useEffect(() => {
    let stale = false;
    setData(null); setDetail(null); setError(""); setLoading(true);
    if (!slug) return;
    const input = new URLSearchParams(query);
    const request = value ? getSearchDetail(slug, kind, input).then(result => {
      if (!stale) { setDetail(result); setData(result.related); }
    }) : getSearchExplore(slug, kind, input).then(result => { if (!stale) setData(result); });
    request.catch((err: Error) => { if (!stale) setError(err.message); }).finally(() => { if (!stale) setLoading(false); });
    return () => { stale = true; };
  }, [slug, kind, query, value, refresh]);

  useEffect(() => {
    let stale = false; setSaved(new Set());
    if (slug) getSavedKeywords(slug).then(result => { if (!stale) setSaved(new Set((result.items || []).map(row => row.query))); }).catch(() => undefined);
    return () => { stale = true; };
  }, [slug]);

  function apply(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const next = new URLSearchParams();
    new FormData(event.currentTarget).forEach((v, k) => { if (String(v)) next.set(k, String(v)); });
    if (value) next.set("value", value);
    setParams(next);
  }
  function page(number: number) {
    const next = new URLSearchParams(params); if (data) { next.set("from", data.filters.from); next.set("through", data.filters.through); } next.set("page", String(number)); setParams(next);
  }
  function target(name: string, targetKind: Kind) {
    const next = new URLSearchParams(params); if (data) { next.set("from", data.filters.from); next.set("through", data.filters.through); } next.set("value", name); next.delete("page"); next.delete("q");
    return `/p/${slug}/search/${targetKind === "query" ? "keywords" : "pages"}?${next}`;
  }
  async function exportRows() {
    if (!slug) return;
    setExporting(true); setError("");
    try {
      const input = new URLSearchParams(params); if (data) { input.set("from", data.filters.from); input.set("through", data.filters.through); }
      const blob = await exportSearchCSV(slug, kind, input);
      const href = URL.createObjectURL(blob); const link = document.createElement("a");
      link.href = href; link.download = `search-${kind}.csv`; link.click();
      setTimeout(() => URL.revokeObjectURL(href), 1000);
    } catch (err) { setError((err as Error).message); }
    finally { setExporting(false); }
  }
  async function save(name: string) {
    if (!slug) return;
    try { await saveKeyword(slug, name); setSaved(prev => new Set([...prev, name])); }
    catch (err) { setError((err as Error).message); }
  }
  const tableKind: Kind = detail ? (kind === "query" ? "page" : "query") : kind;
  const pages = data ? Math.max(1, Math.ceil(data.total / data.page_size)) : 1;
  const filters = data?.filters;
  return <section>
    <p className="page-description mb-5">{t(kind === "query" ? "search.kwDescription" : "search.pagesDescription")}</p>
    {value && <Link className="mb-4 inline-block text-primary-700 underline" to={`/p/${slug}/search/${kind === "query" ? "keywords" : "pages"}?${(() => { const next = new URLSearchParams(params); next.delete("value"); next.delete("page"); return next; })()}`}>{t("search.explore.back")}</Link>}
    <form key={`${query}/${filters?.from || ""}`} onSubmit={apply} className="mb-5 flex flex-wrap items-end gap-3">
      <label className="flex flex-col text-sm text-gray-700">{t("search.explore.from")}<input aria-label={t("search.explore.from")} className="input mt-1" type="date" name="from" defaultValue={params.get("from") || filters?.from || ""} /></label>
      <label className="flex flex-col text-sm text-gray-700">{t("search.explore.through")}<input aria-label={t("search.explore.through")} className="input mt-1" type="date" name="through" defaultValue={params.get("through") || filters?.through || ""} /></label>
      <label className="flex flex-col text-sm text-gray-700">{t("search.explore.contains")}<input className="input mt-1" name="q" maxLength={1000} defaultValue={params.get("q") || ""} /></label>
      <label className="flex flex-col text-sm text-gray-700">{t("search.explore.country")}<input className="input mt-1 max-w-32" name="country" maxLength={3} pattern="[a-zA-Z]{3}" defaultValue={params.get("country") || ""} placeholder={t("search.explore.countryHint")} /></label>
      <label className="flex flex-col text-sm text-gray-700">{t("search.explore.device")}<select className="input mt-1" name="device" defaultValue={params.get("device") || ""}>
        <option value="">{t("search.explore.allDevices")}</option>{(["desktop", "mobile", "tablet"] as const).map(device => <option key={device} value={device}>{t(`search.explore.${device}`)}</option>)}
      </select></label>
      <label className="flex flex-col text-sm text-gray-700">{t("search.explore.sort")}<select className="input mt-1" name="sort" defaultValue={params.get("sort") || "clicks"}>
        <option value="clicks">{t("search.clicks")}</option><option value="impressions">{t("search.impressions")}</option><option value="position">{t("search.positionCol")}</option><option value="ctr">{t("search.ctr")}</option><option value="clicks_change">{t("search.explore.change")}</option><option value="name">{t("search.explore.name")}</option>
      </select></label>
      <label className="flex flex-col text-sm text-gray-700">{t("search.explore.direction")}<select className="input mt-1" name="direction" defaultValue={params.get("direction") || "desc"}><option value="desc">{t("search.explore.desc")}</option><option value="asc">{t("search.explore.asc")}</option></select></label>
      <label className="flex flex-col text-sm text-gray-700">{t("search.explore.pageSize")}<select className="input mt-1" name="page_size" defaultValue={params.get("page_size") || "50"}>{[25, 50, 100, 200].map(n => <option key={n} value={n}>{num(n)}</option>)}</select></label>
      <button className="btn btn-primary" disabled={loading}>{t("search.explore.apply")}</button>
      <button type="button" className="btn btn-secondary" onClick={() => setParams(value ? { value } : {})}>{t("search.explore.reset")}</button>
    </form>
    {error && <p className="alert alert-error" role="alert">{error}</p>}
    {loading && <p role="status" className="hint">{t("common.loading")}</p>}
    {data && <>
      {detail ? <>
        <h2 className="mb-4 break-all text-xl font-semibold text-gray-900">{detail.value}</h2>
        <ReportCoverage value={detail.coverage} report={kind} />
        {!detail.comparable && <p className="mb-4 text-sm text-amber-700">{t("search.explore.noComparison")}</p>}
        {detail.ctr_reference && <CTRReferenceCard value={detail.ctr_reference} />}
        <Comparison value={detail.summary} comparable={detail.comparable} measured={detail.coverage.state === "covered" || detail.summary.current_rows > 0} />
        <div className="mb-5 grid min-w-0 gap-4 md:grid-cols-2 [&>*]:min-w-0 [&>*]:overflow-hidden">
          <TimeSeries label={t("search.clicks")} points={detail.daily.map(day => ({ day: day.day, value: day.clicks }))} empty={t("search.noDaily")} />
          <TimeSeries label={t("search.impressions")} points={detail.daily.map(day => ({ day: day.day, value: day.impressions }))} empty={t("search.noDaily")} />
        </div>
        <h3 className="mb-2 font-semibold text-gray-900">{t(kind === "page" ? "search.explore.relatedQueries" : "search.explore.relatedPages")}</h3>
        <p className="hint">{t("search.explore.relatedNote")}</p>
      </> : <ReportCoverage value={data.coverage} report={kind} />}
      {detail && <p className="hint">{tn("search.imports.days", data.coverage.covered_days, { total: data.coverage.total_days })} · {t("search.explore.relatedCoverage")}</p>}
      <p className="mb-3 text-sm text-gray-500">{t("search.explore.previousRange", { from: data.previous_coverage.from, through: data.previous_coverage.through })} · {tn("search.imports.days", data.previous_coverage.covered_days, { total: data.previous_coverage.total_days })}</p>
      {!data.comparable && <p className="mb-4 text-sm text-amber-700">{t("search.explore.noComparison")}</p>}
      <div className="mb-3 flex flex-wrap items-center gap-3">
        <span className="text-sm text-gray-700">{tn("search.explore.results", data.total)}</span>
        {!detail && <button className="btn btn-secondary" disabled={exporting} onClick={exportRows}>{t("search.explore.exportAll")}</button>}
        <button className="btn btn-secondary" onClick={() => setRefresh(n => n + 1)}>{t("common.refresh")}</button>
      </div>
      <div className="table-container"><table className="table min-w-[800px]">
        <thead><tr><th>{t(tableKind === "query" ? "search.query" : "search.url")}</th><th>{t("search.clicks")}</th><th>{t("search.explore.previousClicks")}</th><th>{t("search.explore.change")}</th><th>{t("search.impressions")}</th><th>{t("search.ctr")}</th><th>{t("search.positionCol")}</th>{tableKind === "query" && canEdit && <th>{t("search.save")}</th>}</tr></thead>
        <tbody>{(data.items || []).map(row => <tr key={row.name}>
          <td><Link className="break-all text-primary-700 underline" to={target(row.name, tableKind)}>{row.name}</Link></td>
          <td>{(data.coverage.state !== "covered" && !row.current_rows) ? "—" : num(row.clicks)}</td>
          <td>{(data.previous_coverage.state !== "covered" && !row.previous_rows) ? "—" : num(row.previous_clicks)}</td>
          <td>{data.comparable ? num(row.clicks_change, { signDisplay: "exceptZero" }) : "—"}</td>
          <td>{(data.coverage.state !== "covered" && !row.current_rows) ? "—" : num(row.impressions)}</td>
          <td>{(data.coverage.state !== "covered" && !row.current_rows) || !row.impressions ? "—" : num(row.ctr, { style: "percent", maximumFractionDigits: 2 })}</td>
          <td>{(data.coverage.state !== "covered" && !row.current_rows) || !row.impressions ? "—" : num(row.position, { maximumFractionDigits: 2 })}</td>
          {tableKind === "query" && canEdit && <td><button className="btn btn-secondary btn-sm" disabled={saved.has(row.name)} onClick={() => save(row.name)}>{t(saved.has(row.name) ? "search.saved" : "search.save")}</button></td>}
        </tr>)}{!data.items?.length && <tr><td colSpan={tableKind === "query" && canEdit ? 8 : 7}>{t("search.explore.noResults")}</td></tr>}</tbody>
      </table></div>
      <nav className="mt-4 flex flex-wrap items-center gap-3" aria-label={t("search.explore.pagination")}>
        <button className="btn btn-secondary" disabled={data.page <= 1} onClick={() => page(data.page - 1)}>{t("search.explore.previous")}</button>
        <span className="flex flex-col text-sm text-gray-700">{t("search.explore.page", { page: num(data.page), total: num(pages) })}</span>
        <button className="btn btn-secondary" disabled={data.page >= pages} onClick={() => page(data.page + 1)}>{t("search.explore.next")}</button>
      </nav>
    </>}
  </section>;
}

function Comparison({ value, comparable, measured }: { value: SearchMetric; comparable: boolean; measured: boolean }) {
  const { t, num } = useI18n();
  return <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">{([
    ["search.clicks", measured ? num(value.clicks) : "—"],
    ["search.impressions", measured ? num(value.impressions) : "—"],
    ["search.positionCol", measured && value.impressions ? num(value.position, { maximumFractionDigits: 2 }) : "—"],
    ["search.explore.change", comparable ? num(value.clicks_change, { signDisplay: "exceptZero" }) : "—"],
  ] as const).map(([key, label]) => <div className="card p-4" key={key}><p className="text-sm text-gray-500">{t(key)}</p><p className="text-xl text-gray-900">{label}</p></div>)}</div>;
}
