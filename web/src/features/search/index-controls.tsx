// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useRef, useState, type FormEvent } from "react";
import { addIndexSitemap, getJob, listJobs, setURLPublished, startJob, stopJob, type IndexInventory, type IndexURL, type JobRow } from "../../api";
import { useAccess } from "../../app/access";
import { useI18n } from "../../i18n";

export function IndexJob({ slug, onUpdate }: { slug: string; onUpdate: () => void }) {
  const { t, date } = useI18n(); const { canEdit } = useAccess();
  const [job, setJob] = useState<JobRow | null>(null); const [busy, setBusy] = useState(false); const [pending, setPending] = useState(false); const [error, setError] = useState(""); const [reload, setReload] = useState(0);
  const updated = useRef(onUpdate); updated.current = onUpdate;
  useEffect(() => {
    let disposed = false; let timer: ReturnType<typeof setTimeout>; let wasActive = false;
    setJob(null); setBusy(false); setError("");
    async function load() {
      let again = false;
      try {
        const rows = await listJobs(slug); if (disposed) return;
        setBusy(Boolean(rows.running)); again = Boolean(rows.running);
        const current = (rows.jobs || []).find(j => j.action === "indexing") || null;
        if (current) { const res = await getJob(current.id); if (disposed) return; setJob(res.job); }
        if (wasActive && !rows.running) updated.current();
        wasActive = Boolean(rows.running); setError("");
      } catch(e) { if (!disposed) setError((e as Error).message); again = true; }
      if (!disposed && again) timer = setTimeout(load, 3000);
    }
    void load(); return () => { disposed = true; clearTimeout(timer); };
  }, [slug, reload]);
  async function run(stop = false) { setPending(true); setError(""); try { if (stop && job) await stopJob(job.id); else await startJob(slug, "indexing"); updated.current(); setReload(n => n + 1); } catch(e) { setError((e as Error).message); } finally { setPending(false); } }
  const active = job && ["running", "queued"].includes(job.status);
  return <div className="mb-4 flex flex-wrap items-center gap-3">
    {canEdit && <button className="btn btn-primary" disabled={busy || pending} onClick={() => run()}>{t("search.indexing.run")}</button>}
    {active && <><p role="status" className="text-sm text-gray-700">{t(job.status === "queued" ? "search.indexing.queued" : "search.indexing.running")} {job.resume_at ? date(job.resume_at * 1000, { dateStyle: "medium", timeStyle: "short" }) : ""}</p>{canEdit && <button className="btn btn-secondary" disabled={pending} onClick={() => run(true)}>{t("schedule.stop")}</button>}</>}
    {(error || job?.error) && <p role="alert" className="w-full break-words text-sm text-red-700">{error || job?.error}</p>}
  </div>;
}
export function SitemapPanel({ slug, data, onUpdate }: { slug: string; data: IndexInventory; onUpdate: () => void }) {
  const { t, tn, date } = useI18n(); const { canEdit } = useAccess(); const [error, setError] = useState(""); const [busy, setBusy] = useState(false);
  async function add(e: FormEvent<HTMLFormElement>) { e.preventDefault(); const form = e.currentTarget; setBusy(true); setError(""); try { await addIndexSitemap(slug, String(new FormData(form).get("url"))); form.reset(); onUpdate(); } catch(e) { setError((e as Error).message); } finally { setBusy(false); } }
  return <details className="card mb-4 p-4"><summary className="cursor-pointer font-medium text-gray-900">{t("search.indexing.sitemaps")}</summary><p className="hint my-3">{t("search.indexing.sitemapNote")}</p>
    {canEdit && <form onSubmit={add} className="mb-3 flex flex-wrap items-end gap-3"><label className="min-w-0 flex-1 text-sm text-gray-700">{t("search.indexing.sitemapURL")}<input className="input mt-1 w-full" name="url" type="url" required maxLength={8192} /></label><button className="btn btn-secondary" disabled={busy}>{t("search.indexing.addSitemap")}</button></form>}
    {error && <p role="alert" className="text-sm text-red-700">{error}</p>}
    <ul className="space-y-3">{(data.sitemaps || []).map(s => <li key={s.id} className="border-t border-gray-100 pt-3 text-sm"><p className="break-all text-gray-700">{s.url}</p><p className="text-gray-500">{tn(s.is_index ? "search.indexing.childMaps" : "search.indexing.discovered", s.discovered)} · {s.fetched_at ? date(s.fetched_at * 1000) : t("search.indexing.unread")}</p>{s.last_error && <p className="break-words text-amber-700">{s.last_error}</p>}</li>)}</ul>
  </details>;
}
export function IndexLifecycle({ slug, row, onUpdate }: { slug: string; row: IndexURL; onUpdate: () => void }) {
  const { t, date } = useI18n(); const { canEdit } = useAccess(); const [error, setError] = useState(""); const [busy, setBusy] = useState(false);
  const local = (n: number | null) => { if (!n) return ""; const d = new Date(n * 1000); return new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().slice(0,16); };
  async function save(e: FormEvent<HTMLFormElement>) { e.preventDefault(); const value = String(new FormData(e.currentTarget).get("published")); setBusy(true); setError(""); try { await setURLPublished(slug, row.url, value ? Math.floor(new Date(value).getTime() / 1000) : null); onUpdate(); } catch(e) { setError((e as Error).message); } finally { setBusy(false); } }
  const group = ["homepage", "article", "documentation", "product", "document"].includes(row.content_group) ? row.content_group as "homepage" | "article" | "documentation" | "product" | "document" : "page";
  return <div className="card mt-4 p-5"><h2 className="font-semibold text-gray-900">{t("search.indexing.lifecycle")}</h2><p className="hint my-2">{t("search.indexing.publicationNote")}</p>
    {canEdit ? <form onSubmit={save} className="my-3 flex flex-wrap items-end gap-3"><label className="text-sm text-gray-700">{t("search.indexing.published")}<input className="input mt-1 block" type="datetime-local" name="published" defaultValue={local(row.published_at)} /></label><button className="btn btn-secondary" disabled={busy}>{t("common.save")}</button></form> : <p className="text-sm text-gray-700">{t("search.indexing.published")}: {row.published_at ? date(row.published_at * 1000) : t("common.notMeasured")}</p>}
    {error && <p role="alert" className="text-sm text-red-700">{error}</p>}
    <dl className="grid gap-3 text-sm sm:grid-cols-2"><div><dt className="text-gray-500">{t("search.indexing.group")}</dt><dd>{t(`search.indexing.groups.${group}`)}</dd></div><div><dt className="text-gray-500">{t("search.indexing.firstImpression")}</dt><dd>{row.first_impression_at ? date(row.first_impression_at * 1000, { dateStyle: "medium", timeZone: "UTC" }) : t("common.notMeasured")}</dd></div><div><dt className="text-gray-500">{t("search.indexing.sitemaps")}</dt><dd className="break-all">{(row.sitemaps || []).join(" · ") || "—"}</dd></div>
      {row.crawl && <><div><dt className="text-gray-500">{t("search.indexing.httpStatus")}</dt><dd>{row.crawl.status_code || "—"} · {row.crawl.fetched_at ? date(row.crawl.fetched_at * 1000) : "—"}</dd></div><div><dt className="text-gray-500">{t("search.indexing.finalURL")}</dt><dd className="break-all">{row.crawl.final_url || "—"}</dd></div><div><dt className="text-gray-500">{t("search.indexing.crawlDirectives")}</dt><dd className="break-all">{[row.crawl.analysis?.meta_robots, row.crawl.analysis?.x_robots_tag, row.crawl.analysis?.canonical].filter(Boolean).join(" · ") || "—"}</dd></div></>}
    </dl>
  </div>;
}
