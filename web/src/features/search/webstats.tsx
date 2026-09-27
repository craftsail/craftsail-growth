// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { TimeSeries, type ChartPoint } from "../../components/charts/series";
import { getJob, getWebstats, listJobs, startJob, type OfficialRow, type SourceView, type WebQueryRow, type WebstatsSnapshot } from "../../api";
import { useI18n, type Key, type Vars } from "../../i18n";
import { HelpTip } from "../../components/HelpTip";
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
  const { t } = useI18n();
  const { canEdit } = useAccess();
  const [snap, setSnap] = useState<WebstatsSnapshot | null>(null);
  const [err, setErr] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);

  function load() {
    if (!slug) return;
    getWebstats(slug)
      .then((d) => setSnap(d || {}))
      .catch((e: Error) => setErr(e.message));
  }

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [slug]);

  async function pull() {
    if (!slug) return;
    setBusy(true);
    setErr("");
    setNote(t("search.syncing"));
    try {
      const started = await startJob(slug, "webstats");
      let id = started.job?.id;
      if (!id) {
        const listed = await listJobs(slug);
        id = listed.running || listed.jobs?.[0]?.id;
      }
      if (id) {
        for (let i = 0; i < 240; i++) {
          const r = await getJob(id);
          const status = r.job?.status;
          if (status && status !== "running" && status !== "queued") {
            if (r.job.error) setErr(r.job.error);
            break;
          }
          if (i % 3 === 2) {
            getWebstats(slug).then(setSnap).catch(() => undefined);
          }
          await new Promise((resolve) => setTimeout(resolve, 1000));
        }
      }
      setSnap((await getWebstats(slug)) || {});
    } catch (e) {
      setErr((e as Error).message);
    }
    setBusy(false);
  }

  const sources = snap?.sources || [];
  const windows = snap?.windows || [];
  const official = chartSource(snap?.official || []);
  const insight = snap?.insight;
  const top = rowsOf(insight?.top_queries);
  const gaps = rowsOf(insight?.gap_queries);
  const sitemaps = snap?.sitemaps || [];
  const indexed = snap?.index || [];
  const submitted = sitemaps.reduce((n, row) => n + (row.submitted || 0), 0);
  const indexedN = indexed.filter((row) => row.verdict === "PASS").length;
  const gscWindow = windows.find((row) => row.source === "gsc");
  const gaWindow = windows.find((row) => row.source === "ga4");

  return (
    <section>
      <p className="page-description mb-5">{t("search.perfDescription")}</p>
      <div className="mb-5 flex gap-2">
        {canEdit && <span className="inline-flex items-center gap-1"><button className="btn btn-primary" disabled={busy} onClick={pull}>{t("search.sync")}</button><HelpTip id="searchSync" /></span>}
        <button className="btn btn-secondary" disabled={busy} onClick={load}>{t("common.refresh")}</button>
      </div>
      {err && <pre className="whitespace-pre-wrap alert alert-error">{err}</pre>}
      {note && <p className="hint">{note}</p>}
      <div className="mb-6 grid grid-cols-1 gap-4 md:grid-cols-2">
        {sources.map((row) => (
          <StatusCard key={row.source} row={row} slug={slug || ""} />
        ))}
      </div>
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
        <TimeSeries label={t("search.position")} points={seriesOf(official, "gsc", "position")} reversed empty={t("search.emptyPosition")} height={260} />
      </div>
      {snap && (
        <>
          <div className="mt-5 card p-5">
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
          <div className="mt-5 card p-5">
            <div className="stat-label">{t("search.index")}</div>
            <p className="hint">{t("search.indexNote", { submitted: num(submitted), checked: indexed.length, indexed: indexedN })}</p>
            <table className="table">
              <thead>
                <tr><th>{t("search.url")}</th><th>{t("search.verdict")}</th><th>{t("search.coverage")}</th><th>{t("search.lastCrawl")}</th></tr>
              </thead>
              <tbody>
                {indexed.map((row) => (
                  <tr key={row.url}>
                    <td>{row.url}</td>
                    <td>{verdictLabel(t, row.verdict)}</td>
                    <td>{row.coverage_state || "—"}</td>
                    <td>{dayOf(row.last_crawl)}</td>
                  </tr>
                ))}
                {indexed.length === 0 && <tr><td colSpan={4} className="hint">{t("search.noIndex")}</td></tr>}
              </tbody>
            </table>
          </div>
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

function verdictLabel(t: (k: Key) => string, v: string | undefined) {
  if (v === "PASS") return t("search.indexed");
  if (v === "FAIL") return t("search.notIndexed");
  if (v === "NEUTRAL") return t("search.excluded");
  return v || "—";
}

function QueryTable({ title, rows, showPosition, hint }: { title: string; rows: WebQueryRow[]; showPosition?: boolean; hint?: string }) {
  const { t } = useI18n();
  return (
    <div className="mt-5 card p-5">
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
