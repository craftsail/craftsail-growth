// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { getAudit, getJob, listJobs, startJob, type AuditReport, type JobRow, type Project } from "../../api";
import { useAccess } from "../../app/access";
import { SetupProgress } from "../onboarding/progress";
import { useI18n } from "../../i18n";

export function FirstCheck({ project, onAudit }: { project: Project; onAudit: (audit: AuditReport | null) => void }) {
  const { t, tn, date } = useI18n();
  const { canEdit } = useAccess();
  const location = useLocation();
  const nav = useNavigate();
  const [startError, setStartError] = useState<string>((location.state as { firstCheckError?: string } | null)?.firstCheckError || "");
  const [loadError, setLoadError] = useState("");
  const [job, setJob] = useState<JobRow | null>(null);
  const [audit, setAudit] = useState<AuditReport | null>(null);
  const [running, setRunning] = useState(false);
  const [loading, setLoading] = useState(true);
  const [starting, setStarting] = useState(false);
  const [refresh, setRefresh] = useState(0);

  useEffect(() => {
    let disposed = false;
    let timer: ReturnType<typeof setTimeout>;
    async function load() {
      let pollAgain = false;
      try {
        const jobs = await listJobs(project.slug);
        if (disposed) return;
        // Read evidence after job status so a just-completed task cannot
        // stop polling with an audit snapshot taken before it finished.
        const report = await getAudit(project.slug);
        if (disposed) return;
        const current = (jobs.jobs || []).find(item => ["first-check", "serve"].includes(item.action)) || null;
        pollAgain = Boolean(jobs.running);
        setJob(current); setRunning(Boolean(jobs.running)); setAudit(report); onAudit(report);
        setLoadError(""); setLoading(false);
        // A lost start response can still have created a running job.
        if (current && ["running", "queued", "done", "ok"].includes(current.status)) setStartError("");
        if (current?.status === "running") {
          const detail = await getJob(current.id);
          if (disposed) return;
          setJob({ ...detail.job, log: detail.log });
        }
      } catch (e) {
        if (disposed) return;
        setLoadError((e as Error).message); setLoading(false);
      }
      if (!disposed && pollAgain) timer = setTimeout(load, 3000);
    }
    void load();
    return () => { disposed = true; clearTimeout(timer); };
  }, [project.slug, refresh]);

  async function start() {
    setStarting(true); setStartError("");
    try {
      const result = await startJob(project.slug, "first-check");
      setJob(result.job); setRunning(true);
      nav(location.pathname + location.search, { replace: true, state: { ...location.state, firstCheckError: "" } });
    } catch (e) { setStartError((e as Error).message); }
    finally { setStarting(false); setRefresh(n => n + 1); }
  }

  const active = job && ["running", "queued"].includes(job.status);
  const failed = job && ["failed", "error", "stopped", "interrupted"].includes(job.status);
  const ready = audit && !audit.no_site;
  const status = loading ? "loading" : active ? (job.action === "first-check" && job.log?.includes("=== 2/2 audit ===") ? "auditing" : "running") : failed ? "failed" : ready ? "ready" : "notStarted";
  const prefix = `/p/${project.slug}`;

  return <section className="card p-5" aria-label={t("firstCheck.title")}>
    <h2 className="card-title">{t("firstCheck.title")}</h2>
    {project.no_site ? <>
      <p className="mt-2 text-sm text-gray-700">{t("firstCheck.noSite")}</p>
    </> : <>
      <p className="mt-2 text-sm text-gray-700">{t("firstCheck.description")}</p>
      <p className={`mt-3 text-sm ${failed ? "text-amber-700" : "text-gray-700"}`} role="status">{t(`firstCheck.${status}`)}</p>
      {job?.error && <p className="mt-2 text-sm text-red-700 break-words">{job.error}</p>}
      {startError && <div className="alert alert-error mt-3" role="alert"><p>{t("firstCheck.startFailed")}</p><p className="break-words">{startError}</p></div>}
      {loadError && <div className="alert alert-error mt-3" role="alert"><p>{t("firstCheck.loadFailed")}</p><p className="break-words">{loadError}</p><button className="btn btn-secondary mt-2" onClick={() => setRefresh(n => n + 1)}>{t("common.refresh")}</button></div>}
      {ready && <p className="mt-2 text-sm text-gray-700">{tn("firstCheck.pages", audit.page_count)}{audit.run_at ? ` · ${date(audit.run_at * 1000)}` : ""}</p>}
      <div className="mt-3 flex flex-wrap gap-2">
        {ready && <Link className="btn btn-primary" to={`${prefix}/audit`}>{t("firstCheck.review")}</Link>}
        {canEdit && !loading && (!ready || failed || startError) && <button className={ready ? "btn btn-secondary" : "btn btn-primary"} disabled={starting || running || !!loadError} onClick={start}>{t(starting ? "firstCheck.starting" : failed || startError ? "firstCheck.retry" : "firstCheck.start")}</button>}
        {(job || running) && <Link className="btn btn-secondary" to={`${prefix}/settings/schedule`}>{t("firstCheck.runs")}</Link>}
      </div>
      {running && !active && <p className="hint mt-2">{t("firstCheck.busy")}</p>}
    </>}
    <SetupProgress slug={project.slug} hasAudit={!!ready} noSite={project.no_site} />

  </section>;
}
