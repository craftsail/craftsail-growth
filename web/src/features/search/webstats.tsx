// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { TimeSeries, type ChartPoint } from "../../components/charts/series";
import { getJob, getWebstats, runningJob, startJob, stopJob, type JobRow, type GoogleSyncProgress, type OfficialRow, type SourceView, type WebQueryRow, type WebstatsSnapshot } from "../../api";
import { useI18n, type Key, type Vars } from "../../i18n";
import { HelpTip } from "../../components/HelpTip";
import { SearchObservationCard } from "./search-observation";
import { useAccess } from "../../app/access";

function num(v: number | null | undefined) {
  if (v == null || Number.isNaN(v)) return "—";
  return Number.isInteger(v) ? String(v) : String(Math.round(v * 100) / 100);
}

function deltaLabel(t: (k: Key, v?: Vars) => string, rows: { source: string; metric: string; kind: string; dir: string; value: number }[] | undefined, source: string, metric: string) {
  const row = (rows || []).find((item) => item.source === source && item.metric === metric);
  if (!row) return t("search.delta.noPrev");
  if (row.kind === "new") return t("search.delta.new");
  if (row.kind === "no_data") return t("search.delta.noData");
  if (row.kind !== "changed") return t("search.delta.flat");
  const pct = Math.round(row.value * 1000) / 10;
  return row.dir === "up" ? t("search.delta.up", { pct }) : t("search.delta.down", { pct });
}

function deltaDir(rows: { source: string; metric: string; dir: string }[] | undefined, source: string, metric: string) {
  return (rows || []).find((item) => item.source === source && item.metric === metric)?.dir;
}

function dayOf(v: string | undefined) {
  return (v || "").slice(0, 10);
}

function rowsOf(v: WebQueryRow[] | undefined) {
  return v || [];
}

function sourceName(source: string) {
  return source === "ga4" ? "GA4" : "Search Console";
}

function chartSource(official: OfficialRow[]) {
  return official;
}

function seriesOf(rows: OfficialRow[], source: string, field: "clicks" | "impressions" | "sessions" | "position"): ChartPoint[] {
  return rows
    .filter((row) => row.source === source)
    .map((row) => ({ day: dayOf(row.day), value: row[field] ?? null }))
    .sort((a, b) => a.day.localeCompare(b.day));
}

function byDay(rows: OfficialRow[]) {
  const map = new Map<string, { day: string; clicks?: number; impressions?: number; sessions?: number; engaged?: number | null }>();
  for (const row of rows) {
    const day = dayOf(row.day);
    const cur = map.get(day) || { day };
    if (row.source === "gsc") {
      cur.clicks = row.clicks;
      cur.impressions = row.impressions;
    } else {
      cur.sessions = row.sessions;
      cur.engaged = row.engaged;
    }
    map.set(day, cur);
  }
  return [...map.values()].sort((a, b) => b.day.localeCompare(a.day));
}

export function Webstats() {
  const { slug } = useParams();
  const { t, date } = useI18n();
  const { canEdit } = useAccess();
  const [snap, setSnap] = useState<WebstatsSnapshot | null>(null);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [jobId, setJobId] = useState<number | null>(null);
  const [activeJob, setActiveJob] = useState<JobRow | null>(null);
  const [stopping, setStopping] = useState(false);

  function load() {
    if (!slug) return;
    getWebstats(slug).then((d) => { setSnap(d || {}); setErr(""); })
      .catch((e: Error) => setErr(e.message));
  }

  useEffect(() => {
    let disposed = false;
    setSnap(null); setJobId(null); setActiveJob(null); setStopping(false); setBusy(false); setErr("");
    if (slug) {
      getWebstats(slug).then(d => { if (!disposed) setSnap(d || {}); })
        .catch((e: Error) => { if (!disposed) setErr(e.message); });
      runningJob(slug, ["webstats"]).then(job => {
        if (!disposed && job) { setJobId(job.id); setActiveJob(job); setBusy(true); }
      }).catch((e: Error) => { if (!disposed) setErr(e.message); });
    }
    return () => { disposed = true; };
  }, [slug]);

  useEffect(() => {
    if (!jobId || !slug) return;
    let disposed = false;
    let timer: ReturnType<typeof setTimeout>;
    async function poll() {
      try {
        const result = await getJob(jobId!);
        if (disposed) return;
        const done = !["running", "queued"].includes(result.job.status);
        setActiveJob(done ? null : result.job);
        if (done) { setBusy(false); setJobId(null); }
        setErr(result.job.error || "");
        const data = await getWebstats(slug!);
        if (disposed) return;
        setSnap(data || {});
        if (done) return;
      } catch (e) {
        if (disposed) return;
        setErr((e as Error).message);
      }
      if (!disposed) timer = setTimeout(poll, 3000);
    }
    void poll();
    return () => { disposed = true; clearTimeout(timer); };
  }, [jobId, slug]);

  async function pull() {
    if (!slug) return;
    setBusy(true); setErr("");
    try {
      const started = await startJob(slug, "webstats");
      const job = started.job || await runningJob(slug, ["webstats"]);
      if (!job) throw new Error(t("search.imports.noJob"));
      setJobId(job.id); setActiveJob(job);
    } catch (e) {
      setErr((e as Error).message); setBusy(false);
    }
  }


  async function stop() {
    if (!jobId) return;
    setStopping(true);
    try { await stopJob(jobId); setJobId(null); setActiveJob(null); setBusy(false); load(); }
    catch (e) { setErr((e as Error).message); }
    finally { setStopping(false); }
  }

  const sources = snap?.sources || [];
  const windows = snap?.windows || [];
  const official = chartSource(snap?.official || []);
  const insight = snap?.insight;
  const top = rowsOf(insight?.top_queries);
  const gaps = rowsOf(insight?.gap_queries);

  const gscWindow = windows.find((row) => row.source === "gsc");
  const gaWindow = windows.find((row) => row.source === "ga4");

  return (
    <section>
      <p className="page-description mb-5">{t("search.perfDescription")}</p>
      <div className="mb-5 flex flex-wrap gap-2">
        {canEdit && <span className="inline-flex items-center gap-1"><button className="btn btn-primary" disabled={busy} onClick={pull}>{t("search.sync")}</button><HelpTip id="searchSync" /></span>}
        {canEdit && jobId && <button className="btn btn-secondary" disabled={stopping} onClick={stop}>{t("search.imports.stop")}</button>}
        <button className="btn btn-secondary" onClick={load}>{t("common.refresh")}</button>
      </div>
      {err && <pre className="whitespace-pre-wrap alert alert-error">{err}</pre>}
      {busy && <p className="hint" role="status">{activeJob?.status === "queued" && activeJob.resume_at ? t("search.imports.continuing", { time: date(activeJob.resume_at * 1000, { hour: "2-digit", minute: "2-digit", second: "2-digit" }) }) : t("search.syncing")}</p>}
      <div className="mb-6 grid grid-cols-1 gap-4 md:grid-cols-2">
        {sources.map((row) => (
          <StatusCard key={row.source} row={row} slug={slug || ""} />
        ))}
      </div>
      {snap?.observation && <SearchObservationCard key={slug} slug={slug || ""} value={snap.observation} onSaved={load} />}
      <SyncProgressTable rows={snap?.sync || []} busy={busy} />
      <div className="my-2 grid grid-cols-1 gap-3 md:grid-cols-3">
        <Kpi title={t("search.clicks")} value={num(gscWindow?.clicks)} note={deltaLabel(t, snap?.deltas, "gsc", "clicks")} up={deltaDir(snap?.deltas, "gsc", "clicks") === "up"} />
        <Kpi title={t("search.impressions")} value={num(gscWindow?.impressions)} note={deltaLabel(t, snap?.deltas, "gsc", "impressions")} up={deltaDir(snap?.deltas, "gsc", "impressions") === "up"} />
        <Kpi title={t("search.sessions")} value={num(gaWindow?.sessions)} note={deltaLabel(t, snap?.deltas, "ga4", "sessions")} up={deltaDir(snap?.deltas, "ga4", "sessions") === "up"} />
      </div>
      <p className="hint">{t("search.totalsNote")}</p>
      <div className="mb-6 grid grid-cols-1 gap-4 md:grid-cols-2">
        <TimeSeries label={t("search.clicks")} points={seriesOf(official, "gsc", "clicks")} empty={t("search.emptyClicks")} height={260} />
        <TimeSeries label={t("search.impressions")} points={seriesOf(official, "gsc", "impressions")} empty={t("search.emptyImpressions")} height={260} />
        <TimeSeries label={t("search.sessions")} points={seriesOf(official, "ga4", "sessions")} empty={t("search.emptySessions")} height={260} />
        {snap?.observation?.mode !== "new_site" && <TimeSeries label={t("search.position")} points={seriesOf(official, "gsc", "position")} reversed empty={t("search.emptyPosition")} height={260} />}
      </div>
      {snap && (
        <>
          <div className="mt-5 card overflow-x-auto p-5">
            <div className="stat-label">{t("search.dailyTotals")}</div>
            <table className="table">
              <thead>
                <tr><th>{t("search.date")}</th><th>{t("search.clicks")}</th><th>{t("search.impressions")}</th><th>{t("search.sessions")}</th><th>{t("search.engaged")}</th></tr>
              </thead>
              <tbody>
                {byDay(official).map((row) => (
                  <tr key={row.day}>
                    <td>{row.day}</td>
                    <td>{num(row.clicks)}</td>
                    <td>{num(row.impressions)}</td>
                    <td>{num(row.sessions)}</td>
                    <td>{num(row.engaged)}</td>
                  </tr>
                ))}
                {official.length === 0 && <tr><td colSpan={5} className="hint">{t("search.noDaily")}</td></tr>}
              </tbody>
            </table>
          </div>
          <div className="mt-5 card p-5"><Link className="text-primary-700 underline" to={`/p/${slug}/search/indexing`}>{t("nav.tabs.indexing")}</Link><p className="hint mt-2">{t("search.indexing.description")}</p></div>
          {insight && (
            <p className="hint">{t("search.aiSessions", { n: num(insight.ai_sessions) })}</p>
          )}
          <QueryTable title={t("search.drill")} rows={top} showPosition hint={t("search.drillHint")} />
          <QueryTable title={t("search.gaps")} rows={gaps} />
        </>
      )}
    </section>
  );
}

function SyncProgressTable({ rows, busy }: { rows: GoogleSyncProgress[]; busy: boolean }) {
  const { t, tn } = useI18n();
  const reports = ["channel", "landing", "daily", "query", "country_device", "page_country_device", "query_country_device", "query_page", "query_page_country_device", "country", "device", "appearance", "hour", "session", "page", "event"];
  const searchTypes = ["web", "image", "video", "news", "discover", "googleNews"];
  const states = ["running", "backfilling", "completed", "paused", "failed", "partial", "unsupported"];
  const errors = ["budget", "restricted", "cancelled", "truncated", "storage", "rate_limited", "needs_reauth", "config_invalid", "unsupported"];
  return <details className="card mb-5 p-5" open={busy || undefined}>
    <summary className="cursor-pointer font-medium text-gray-900">{t("search.imports.title")}</summary>
    <p className="hint">{t("search.imports.description")}</p>
    <div className="overflow-x-auto">
      <table className="table min-w-[640px]">
        <thead><tr><th>{t("search.imports.report")}</th><th>{t("search.imports.state")}</th><th>{t("search.imports.recent")}</th><th>{t("search.imports.history")}</th><th>{t("search.quality.title")}</th></tr></thead>
        <tbody>{rows.map(row => {
          const state = row.state === "running" && !busy ? "paused" : states.includes(row.state) ? row.state : "failed";
          return <tr key={`${row.source}/${row.property}/${row.report}/${row.search_type}`}>
            <td><div>{sourceName(row.source)} · {t(`search.imports.reports.${reports.includes(row.report) ? row.report : "other"}` as Key)}{searchTypes.includes(row.search_type) && ` · ${t(`search.imports.types.${row.search_type}` as Key)}`}</div><div className="text-xs text-gray-500">{row.property}</div></td>
            <td><span className={state === "completed" ? "text-emerald-700" : ["failed", "partial", "paused"].includes(state) ? "text-amber-700" : "text-gray-700"}>{t(`search.imports.states.${state}` as Key)}</span>{row.error_class && <div className="text-xs text-gray-500">{t(`search.imports.errors.${errors.includes(row.error_class) ? row.error_class : "provider"}` as Key)}</div>}</td>
            <td>{tn("search.imports.days", row.recent_covered_days, { total: row.recent_total_days })}</td>
            <td>{tn("search.imports.days", row.covered_days, { total: row.total_days })}<div className="text-xs text-gray-500">{row.from} – {row.through}</div>
 <div className="text-xs text-gray-500">{t("google.lastSuccess")}: {row.last_success_at ? new Date(row.last_success_at*1000).toLocaleString() : t("common.none")}</div>
 {!!row.retry_at && <div className="text-xs text-amber-700">{t("google.retryAt")}: {new Date(row.retry_at*1000).toLocaleString()}</div>}
 {!!row.gaps?.length && <details><summary className="cursor-pointer text-xs text-gray-500">{t("google.missingDates")}</summary>{row.gaps.map(g=><div key={g.from} className="text-xs text-gray-500">{g.from} – {g.through}</div>)}</details>}
 </td>
            <td>{row.source === "ga4" ? <>
              {!row.quality?.known && <div>{t("search.quality.unknown")}</div>}
              {(["sampled", "thresholded", "other_row", "restricted", "empty_reason"] as const).filter(flag => row.quality?.[flag]).map(flag => <div key={flag} className="text-amber-700">{t(`search.quality.${flag}`)}</div>)}
              {row.quality?.known && ![row.quality.sampled, row.quality.thresholded, row.quality.other_row, row.quality.restricted, row.quality.empty_reason].some(Boolean) && <div>{t("search.quality.noFlags")}</div>}
              <div className="text-xs text-gray-500">{[...(row.quality?.currencies || []), ...(row.quality?.time_zones || [])].join(" · ")}</div>
            </> : t("search.quality.gsc")}</td>
          </tr>;
        })}{rows.length === 0 && <tr><td colSpan={5} className="hint">{t("search.imports.empty")}</td></tr>}</tbody>
      </table>
    </div>
  </details>;
}

function StatusCard({ row, slug }: { row: SourceView; slug: string }) {
  const { t } = useI18n();
  const { isAdmin } = useAccess();
  const known = ["ready", "waiting", "choose_property", "not_connected", "needs_reauth", "rate_limited", "config_invalid", "failed"].includes(row.state);
  return (
    <div className="card p-5">
      <div className="stat-label">{sourceName(row.source)}</div>
      <div className="text-lg stat-value">{known ? t(`search.states.${row.state}` as Key) : row.label}</div>
      <p className="hint">{row.detail}</p>
      {row.property && <p className="hint">{t("search.property", { p: row.property })}{row.calendar ? ` · ${row.calendar}` : ""}{row.through ? ` · ${t("search.through", { d: row.through })}` : ""}</p>}
      {isAdmin && row.can_reconnect && <Link to={`/p/${slug}/settings/google`}>{t("search.reconnect")}</Link>}
    </div>
  );
}

function Kpi({ title, value, note, up }: { title: string; value: string; note: string; up?: boolean }) {
  return (
    <div className="card p-5">
      <div className="stat-label">{title}</div>
      <div className="stat-value">{value}</div>
      <p className="hint" style={up ? { color: "#0f766e" } : undefined}>{note}</p>
    </div>
  );
}

function QueryTable({ title, rows, showPosition, hint }: { title: string; rows: WebQueryRow[]; showPosition?: boolean; hint?: string }) {
  const { t } = useI18n();
  return (
    <div className="mt-5 card overflow-x-auto p-5">
      <div className="stat-label">{title}</div>
      {hint && <p className="hint">{hint}</p>}
      <table className="table">
        <thead>
          <tr>
            <th>{t("search.query")}</th><th>{t("search.clicks")}</th><th>{t("search.impressions")}</th>
            {showPosition && <th>{t("search.positionCol")}</th>}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, i) => (
            <tr key={`${row.query}-${i}`}>
              <td>{row.query || "—"}</td>
              <td>{num(row.clicks)}</td>
              <td>{num(row.impressions)}</td>
              {showPosition && <td>{num(row.position)}</td>}
            </tr>
          ))}
          {rows.length === 0 && (
            <tr><td colSpan={showPosition ? 4 : 3} className="hint">{t("search.noRows")}</td></tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
