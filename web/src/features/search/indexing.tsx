// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useState, type FormEvent } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { getIndexInventory, getIndexHistory, type IndexInventory, type IndexHistory, type GscIndexRow } from "../../api";
import { IndexJob, IndexLifecycle, SitemapPanel } from "./index-controls";
import { useI18n } from "../../i18n";

export function Indexing() {
  const { slug } = useParams();
  const { t, tn, num, date } = useI18n();
  const [params, setParams] = useSearchParams();
  const query = params.toString();
  const [data, setData] = useState<IndexInventory | null>(null);
  const [error, setError] = useState("");
  const [refresh, setRefresh] = useState(0);
  const [selected, setSelected] = useState("");
  useEffect(() => {
    let stale = false;
    setData(null); setError("");
    if (slug) getIndexInventory(slug, new URLSearchParams(query)).then(d => { if (!stale) setData(d); }).catch((e: Error) => { if (!stale) setError(e.message); });
    return () => { stale = true; };
  }, [slug, query, refresh]);
  useEffect(() => { setSelected(""); }, [slug, query]);
  function apply(e: FormEvent<HTMLFormElement>) {
    e.preventDefault(); const next = new URLSearchParams();
    if (params.has("lang")) next.set("lang", params.get("lang")!);
    new FormData(e.currentTarget).forEach((v, k) => { if (String(v)) next.set(k, String(v)); });
    setParams(next);
  }
  function page(n: number) { const next = new URLSearchParams(params); next.set("page", String(n)); setParams(next); }
  const time = (n: number | null) => n ? date(n * 1000, { dateStyle: "medium", timeStyle: "short" }) : "—";
  const pages = data ? Math.max(1, Math.ceil(data.total / data.page_size)) : 1;
  return <section>
    <p className="page-description mb-3">{t("search.indexing.description")}</p>
    <p className="hint mb-5">{t("search.indexing.scope")} <Link className="text-primary-700 underline" to={`/p/${slug}/search`}>{t("search.grain.viewSync")}</Link></p>
    {slug && <IndexJob slug={slug} onUpdate={() => setRefresh(n => n + 1)} />}
    <form key={query} onSubmit={apply} className="mb-5 flex flex-wrap items-end gap-3">
      <label className="flex flex-col text-sm text-gray-700">{t("search.explore.contains")}<input name="q" className="input mt-1" maxLength={2000} defaultValue={params.get("q") || ""} /></label>
      <label className="flex flex-col text-sm text-gray-700">{t("search.verdict")}<select name="state" className="input mt-1" defaultValue={params.get("state") || ""}>{(["all", "pending", "indexed", "other", "error", "due"] as const).map(s => <option key={s} value={s === "all" ? "" : s}>{t(`search.indexing.${s}`)}</option>)}</select></label>
      <label className="flex flex-col text-sm text-gray-700">{t("search.indexing.group")}<select className="input mt-1" name="group" defaultValue={params.get("group") || ""}><option value="">{t("search.indexing.all")}</option>{(["homepage", "article", "documentation", "product", "document", "page"] as const).map(g => <option key={g} value={g}>{t(`search.indexing.groups.${g}`)}</option>)}</select></label>
      <button className="btn btn-primary">{t("search.explore.apply")}</button>
      <button className="btn btn-secondary" type="button" onClick={() => setRefresh(n => n + 1)}>{t("common.refresh")}</button>
    </form>
    {error && <p role="alert" className="alert alert-error">{error}</p>}
    {!data && !error && <p role="status" className="hint">{t("common.loading")}</p>}
    {data && <>
      <p className="mb-3 break-all text-sm text-gray-500">{t("search.indexing.property", { property: data.property || "—" })}</p>
      <div className="mb-5 grid grid-cols-2 gap-3 lg:grid-cols-4">{(["known", "inspected", "indexed", "due"] as const).map(k => <div key={k} className="card p-4"><p className="stat-label">{t(`search.indexing.${k}`)}</p><p className="text-2xl font-semibold text-gray-900">{num(data[k])}</p></div>)}</div>
      <p className="hint mb-3">{t("search.indexing.schedule")}</p>
      {!!data.quota && data.quota.blocked_until > Date.now() / 1000 && <p role="status" className="mb-3 text-sm text-amber-700">{t("search.indexing.quota", { when: time(data.quota.blocked_until) })}</p>}
      <p className="hint mb-3">{t("search.indexing.week", { indexed: num(data.indexed_within_week || 0), total: num(data.published_mature || 0) })}</p>
      {slug && <SitemapPanel slug={slug} data={data} onUpdate={() => setRefresh(n => n + 1)} />}

      <p className="mb-3 text-sm text-gray-700">{tn("search.explore.results", data.total)}</p>
      <div className="table-container"><table className="table min-w-[1050px]">
        <thead><tr><th>{t("search.url")}</th><th>{t("search.indexing.source")}</th><th>{t("search.verdict")}</th><th>{t("search.indexing.checked")}</th><th>{t("search.indexing.firstIndexed")}</th><th>{t("search.indexing.next")}</th><th>{t("search.indexing.history")}</th></tr></thead>
        <tbody>{data.items.map(row => <tr key={row.id}>
          <td className="max-w-xs break-all whitespace-normal">{row.url}</td>
          <td>{[row.from_crawl && t("search.indexing.crawl"), row.from_search && t("search.indexing.search"), row.from_sitemap && t("search.indexing.sitemapSource")].filter(Boolean).join(" · ") || "—"}</td>
          <td><Verdict value={row.last_success_at ? row.verdict : undefined} />{row.last_error && <span className="mt-1 block text-sm text-amber-700">{t("search.indexing.failed")}</span>}</td>
          <td>{time(row.last_success_at)}</td><td>{time(row.first_indexed_at)}</td><td>{row.next_inspect_at ? time(row.next_inspect_at) : t("search.indexing.due")}</td>
          <td><button className="btn btn-secondary" aria-expanded={selected === row.url} onClick={() => setSelected(selected === row.url ? "" : row.url)}>{t(selected === row.url ? "common.close" : "search.indexing.history")}</button></td>
        </tr>)}{!data.items.length && <tr><td colSpan={7} className="hint">{t("search.indexing.empty")}</td></tr>}</tbody>
      </table></div>
      <div className="my-4 flex flex-wrap items-center gap-3"><button className="btn btn-secondary" disabled={data.page <= 1} onClick={() => page(data.page - 1)}>{t("search.explore.previous")}</button><span className="text-sm text-gray-500">{t("search.explore.page", { page: num(data.page), total: num(pages) })}</span><button className="btn btn-secondary" disabled={data.page >= pages} onClick={() => page(data.page + 1)}>{t("search.explore.next")}</button></div>
      {selected && slug && <>{data.items.find(r => r.url === selected) && <IndexLifecycle key={`${slug}/${selected}`} slug={slug} row={data.items.find(r => r.url === selected)!} onUpdate={() => setRefresh(n => n + 1)} />}<History key={`${slug}/${selected}/${refresh}`} slug={slug} url={selected} /></>}
    </>}
  </section>;
}
function Verdict({ value }: { value?: string }) {
  const { t } = useI18n();
  const label = value === "PASS" ? t("search.indexed") : value === "FAIL" ? t("search.notIndexed") : value === "NEUTRAL" ? t("search.excluded") : value ? t("search.indexing.unknown") : t("search.indexing.pending");
  return <span className={value === "PASS" ? "text-emerald-700" : "text-gray-700"}>{label}</span>;
}
function Details({ row }: { row: GscIndexRow }) {
  const { t } = useI18n();
  return <dl className="mt-3 grid gap-x-6 gap-y-2 text-sm sm:grid-cols-2">{(["google_canonical", "user_canonical", "robots_txt_state", "page_fetch_state", "indexing_state", "last_crawl", "coverage_state"] as const).map(k => <div key={k}><dt className="text-gray-500">{t(`search.indexing.${k}`)}</dt><dd className="break-all text-gray-700">{row[k] || "—"}</dd></div>)}</dl>;
}
function History({ slug, url }: { slug: string; url: string }) {
  const { t, date } = useI18n();
  const [page, setPage] = useState(1);
  const [data, setData] = useState<IndexHistory | null>(null);
  const [error, setError] = useState("");
  const [refresh, setRefresh] = useState(0);
  useEffect(() => {
    let stale = false; setData(null); setError("");
    getIndexHistory(slug, url, page).then(d => { if (!stale) setData(d); }).catch((e: Error) => { if (!stale) setError(e.message); });
    return () => { stale = true; };
  }, [slug, url, page, refresh]);
  const pages = data ? Math.max(1, Math.ceil(data.total / data.page_size)) : 1;
  return <div className="card mt-5 p-5"><h2 className="font-semibold text-gray-900">{t("search.indexing.history")}</h2><p className="my-2 break-all text-sm text-gray-700">{url}</p><p className="hint">{t("search.indexing.historyNote")}</p>
    {error && <div role="alert" className="alert alert-error">{error}<button className="btn btn-secondary ml-3" onClick={() => setRefresh(n => n + 1)}>{t("common.refresh")}</button></div>}
    {!data && !error && <p role="status" className="hint">{t("common.loading")}</p>}
    {data?.items.map(row => <article key={row.id} className="mt-4 border-t border-gray-100 pt-4"><p className="text-sm text-gray-700">{date(row.checked_at * 1000, { dateStyle: "medium", timeStyle: "short" })}</p>{row.error ? <><p className="mt-2 text-sm text-amber-700">{t("search.indexing.failedNote")}</p><p className="mt-1 break-words text-sm text-gray-500">{row.error}</p></> : <><Verdict value={row.result?.verdict} />{row.result && <Details row={row.result} />}</>}</article>)}
    {data && !data.items.length && <p className="hint mt-3">{t("search.indexing.noHistory")}</p>}
    {data && data.total > data.page_size && <div className="mt-4 flex flex-wrap items-center gap-3"><button className="btn btn-secondary" disabled={page <= 1} onClick={() => setPage(n => n - 1)}>{t("search.explore.previous")}</button><span className="text-sm text-gray-500">{t("search.explore.page", { page, total: pages })}</span><button className="btn btn-secondary" disabled={page >= pages} onClick={() => setPage(n => n + 1)}>{t("search.explore.next")}</button></div>}
  </div>;
}
