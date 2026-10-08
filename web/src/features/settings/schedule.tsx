// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useState } from "react";
import { useOutletContext, useParams } from "react-router-dom";
import { getJob, listJobs, listRuns, patchMonitor, retryRun, startJob, stopJob, type JobRow, type Project, type SampleRunRow } from "../../api";
import { useI18n, type Key } from "../../i18n";
import { HelpTip } from "../../components/HelpTip";
import { useAccess } from "../../app/access";

function when(sec: number | null | undefined, intl: string) {
  if (!sec) return "-";
  return new Date(sec * 1000).toLocaleString(intl, { hour12: false });
}

const ACTIONS = ["serve", "sample", "webstats", "indexing", "crawl", "audit", "verify"];
const ACTION_TIPS = { serve: "serve", sample: "sample", webstats: "syncGoogle", indexing: "indexing", crawl: "crawl", audit: "audit", verify: "verify" } as const;
const KNOWN_ACTIONS = [...ACTIONS, "bootstrap", "report", "first-check"];
const JOB_STATUS = ["running", "queued", "done", "failed", "error", "stopped", "interrupted", "ok"];

// Schedule & Runs: how often a period runs, how many samples per prompt, and
// what ran. Everything starts through the job system, one job per project.
export function Schedule() {
  const { slug = "" } = useParams();
  const { t, intl, num } = useI18n();
  const { canEdit } = useAccess();
  const ctx = useOutletContext<{ project?: Project; onProject?: (p: Project) => void } | undefined>();
  const project = ctx?.project;
  const [every, setEvery] = useState(project?.monitor_every_days ?? 0);
  const [perDay, setPerDay] = useState(project?.monitor_runs_per_day || 3);
  const [runs, setRuns] = useState<SampleRunRow[]>([]);
  const [jobs, setJobs] = useState<JobRow[]>([]);
  const [log, setLog] = useState<{ id: number; text: string } | null>(null);
  const [msg, setMsg] = useState("");
  const [err, setErr] = useState("");

  function load() {
    listRuns(slug).then((d) => setRuns(d.items || [])).catch(() => setRuns([]));
    listJobs(slug).then((d) => setJobs(d.jobs || [])).catch((e: Error) => setErr(e.message));
  }
  useEffect(load, [slug]);
  useEffect(() => {
    setEvery(project?.monitor_every_days ?? 0);
    setPerDay(project?.monitor_runs_per_day || 3);
  }, [project?.slug]);
  useEffect(() => {
    if (!jobs.some((j) => j.status === "running" || j.status === "queued")) return;
    const t = window.setInterval(load, 3000);
    return () => window.clearInterval(t);
  }, [jobs, slug]);

  async function save() {
    setErr(""); setMsg("");
    try {
      const p = await patchMonitor(slug, every, perDay);
      ctx?.onProject?.(p);
      setMsg(every > 0 ? t("schedule.savedNext", { date: (p.monitor_next_run || "").slice(0, 10) }) : t("schedule.savedOff"));
    } catch (e) {
      setErr((e as Error).message);
    }
  }

  async function act(fn: () => Promise<unknown>) {
    setErr(""); setMsg("");
    try {
      await fn();
      load();
    } catch (e) {
      setErr((e as Error).message);
    }
  }

  async function showLog(id: number) {
    try {
      const d = await getJob(id);
      setLog({ id, text: d.log || t("schedule.emptyLog") });
    } catch (e) {
      setErr((e as Error).message);
    }
  }

  const ghost = "btn btn-secondary";
  const card = "card p-5";
  return (
    <section className="space-y-4">
      <div className={card}>
        <h2 className="card-title">{t("schedule.title")}</h2>
        <p className="mt-1 max-w-3xl text-sm text-gray-600">{t("schedule.intro")}</p>
        <fieldset disabled={!canEdit} className="mt-3 flex flex-wrap items-end gap-3 text-sm">
          <label className="flex flex-col gap-1">
            <span className="text-xs text-gray-500">{t("schedule.every")}</span>
            <select className="input w-auto" value={every} onChange={(e) => setEvery(Number(e.target.value))}>
              <option value={0}>{t("schedule.off")}</option>
              <option value={1}>{t("schedule.day")}</option>
              <option value={7}>{t("schedule.week")}</option>
              <option value={14}>{t("schedule.twoWeeks")}</option>
              <option value={30}>{t("schedule.thirtyDays")}</option>
            </select>
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-xs text-gray-500">{t("schedule.perDay")}</span>
            <input type="number" min={1} max={10} className="input w-24" value={perDay} onChange={(e) => setPerDay(Number(e.target.value))} />
          </label>
          {canEdit && <span className="inline-flex items-center gap-1"><button type="button" className="btn btn-primary" onClick={save}>{t("common.save")}</button><HelpTip id="scheduleSave" /></span>}
          {msg && <span className="text-gray-600">{msg}</span>}
        </fieldset>
      </div>

      {canEdit && <div className="flex flex-wrap gap-2">
        {ACTIONS.map((action) => (
          <button key={action} type="button" className={action === "serve" ? "btn btn-primary" : "btn btn-secondary"} onClick={() => act(() => startJob(slug, action))}>{t(`schedule.actions.${action}` as Key)}</button>
        )).flatMap((b, i) => [b, <HelpTip key={`tip-${ACTIONS[i]}`} id={ACTION_TIPS[ACTIONS[i] as keyof typeof ACTION_TIPS]} />])}
      </div>}
      {err && <p className="text-sm text-red-700">{err}</p>}

      <div className={card}>
        <h2 className="card-title mb-3">{t("schedule.runsTitle")}</h2>
        {runs.length === 0 ? <p className="text-sm text-gray-500">{t("schedule.noRuns")}</p> : (
          <table className="table">
            <thead><tr className="text-left text-xs text-gray-500"><th className="py-1">{t("schedule.started")}</th><th>{t("schedule.trigger")}</th><th>{t("schedule.result")}</th><th className="text-right">{t("schedule.answers")}</th><th className="text-right">{t("schedule.failed")}</th><th className="text-right">{t("schedule.tokens")}</th><th /></tr></thead>
            <tbody>
              {runs.map((r) => (
                <tr key={r.id} className="border-t border-gray-100">
                  <td className="py-1.5">{when(r.started_at, intl)}</td>
                  <td>{["manual", "schedule", "retry", "import"].includes(r.trigger) ? t(`schedule.triggers.${r.trigger}` as Key) : r.trigger}</td>
                  <td>{r.status === "running" ? t("schedule.running") : r.status === "cancelled" ? t("schedule.cancelled") : ["succeeded", "partial", "failed"].includes(r.outcome) ? t(`schedule.outcomes.${r.outcome}` as Key) : r.outcome}</td>
                  <td className="text-right tabular-nums">{r.succeeded} / {r.planned}</td>
                  <td className="text-right tabular-nums">{r.failed}</td>
                  <td className="text-right tabular-nums">{num(r.est_tokens)} / {r.used_tokens ? num(r.used_tokens) : t("schedule.notReported")}</td>
                  <td className="text-right">{canEdit && r.failed > 0 && r.status !== "running" && <button type="button" className={ghost} onClick={() => act(() => retryRun(slug, r.id))}>{t("schedule.retry")}</button>}{canEdit && r.failed > 0 && r.status !== "running" && <HelpTip id="retry" />}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <div className={card}>
        <h2 className="card-title mb-3">{t("schedule.jobsTitle")}</h2>
        {jobs.length === 0 ? <p className="text-sm text-gray-500">{t("schedule.noJobs")}</p> : (
          <table className="table">
            <thead><tr className="text-left text-xs text-gray-500"><th className="py-1">{t("schedule.job")}</th><th>{t("schedule.status")}</th><th>{t("schedule.started")}</th><th>{t("schedule.error")}</th><th /></tr></thead>
            <tbody>
              {jobs.map((j) => (
                <tr key={j.id} className="border-t border-gray-100 align-top">
                  <td className="py-1.5"><button type="button" className="text-left text-primary-700" onClick={() => showLog(j.id)}>{KNOWN_ACTIONS.includes(j.action) ? t(`schedule.actions.${j.action}` as Key) : j.label || j.action}</button></td>
                  <td>{JOB_STATUS.includes(j.status) ? t(`status.${j.status}` as Key) : j.status}</td>
                  <td>{when(j.started_at, intl)}</td>
                  <td className="max-w-md break-words text-gray-600">{j.error || "-"}</td>
                  <td className="text-right">{canEdit && (j.status === "running" || j.status === "queued") && <button type="button" className={ghost} onClick={() => window.confirm(t("schedule.stopConfirm")) && act(() => stopJob(j.id))}>{t("schedule.stop")}</button>}{canEdit && (j.status === "running" || j.status === "queued") && <HelpTip id="stop" />}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
        {log && (
          <div className="mt-3">
            <div className="mb-1 flex items-center justify-between text-xs text-gray-500"><span>{t("schedule.logOf", { id: log.id })}</span><button type="button" onClick={() => setLog(null)}>{t("common.close")}</button></div>
            <pre className="code-block max-h-80 text-xs">{log.text}</pre>
          </div>
        )}
      </div>
    </section>
  );
}
