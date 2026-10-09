// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { applySystemUpdate, getSystemVersion, restartSystem, type SystemVersion } from "../../api";
import { useI18n, type Key } from "../../i18n";

const errorKeys: Record<string, Key> = Object.fromEntries([
  "busy", "unsupported", "pending", "target", "download", "checksum", "archive", "install",
  "restartUnavailable", "jobsRunning", "failed", "checkFailed", "noRelease", "noAsset",
].map(k => [`update.${k}`, `update.${k}`])) as Record<string, Key>;
const errorKey = (error: unknown): Key => errorKeys[error instanceof Error ? error.message : String(error)] || "update.failed";

export function VersionBadge({ collapsed }: { collapsed: boolean }) {
  const { t } = useI18n();
  const { slug } = useParams();
  const [state, setState] = useState<SystemVersion>();
  useEffect(() => {
    let active = true;
    const check = () => { getSystemVersion().then(v => { if (active) setState(v); }).catch(() => {}); };
    check();
    const timer = window.setInterval(check, 20 * 60 * 1000);
    return () => { active = false; clearInterval(timer); };
  }, []);
  return <Link to={`/p/${slug}/settings/system`} title={t("nav.system")} className="mb-2 block rounded-lg px-3 py-2 text-xs text-gray-500 hover:bg-gray-100">
    {collapsed ? <span aria-label={t("nav.system")}>↑</span> : <>{t("nav.system")}{state && <span className="ml-2">{state.current.version}</span>}
      {state?.has_update && !state.warning && <span className="mt-1 block text-primary-700">{t("update.available")}</span>}</>}
  </Link>;
}

export function SystemUpdates() {
  const { t } = useI18n();
  const [state, setState] = useState<SystemVersion>();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<Key>();
  const [message, setMessage] = useState<Key>();
  const [target, setTarget] = useState("backup");
  useEffect(() => { let active = true; getSystemVersion().then(v => { if (active) setState(v); }).catch(e => { if (active) setError(errorKey(e)); }); return () => { active = false; }; }, []);
  async function check() {
    setBusy(true); setError(undefined); setMessage(undefined);
    try { setState(await getSystemVersion(true)); } catch (e) { setError(errorKey(e)); } finally { setBusy(false); }
  }
  async function apply(rollback: boolean) {
    if (!state) return;
    const version = rollback ? target : state.latest?.tag_name;
    if (!version || !window.confirm(t(rollback ? "update.confirmRollback" : "update.confirmUpdate", { version: version === "backup" ? state.backup_version || "" : version }))) return;
    setBusy(true); setError(undefined); setMessage("update.installing");
    try {
      await applySystemUpdate(version, rollback);
      setState(await getSystemVersion()); setMessage("update.installed");
    } catch (e) { setError(errorKey(e)); setMessage(undefined); } finally { setBusy(false); }
  }
  async function restart() {
    if (!state || !window.confirm(t("update.confirmRestart"))) return;
    setBusy(true); setError(undefined); setMessage("update.restarting");
    try {
      await restartSystem();
      const deadline = Date.now() + 120_000;
      while (Date.now() < deadline) {
        await new Promise(resolve => setTimeout(resolve, 2000));
        try {
          const next = await getSystemVersion();
          if (next.instance !== state.instance) { window.location.reload(); return; }
        } catch { /* The server is expected to be temporarily unavailable. */ }
      }
      setMessage("update.restartTimeout");
    } catch (e) { setError(errorKey(e)); setMessage(undefined); } finally { setBusy(false); }
  }
  const rollbackAvailable = state && (target === "backup" ? !!state.backup_version : state.rollbacks.includes(target));
  return <div className="max-w-3xl space-y-5">
    <p className="page-description">{t("update.intro")}</p>
    {error && <p role="alert" className="alert alert-error">{t(error)}</p>}
    {message && <p role="status" className="alert alert-info">{t(message)}</p>}
    <section className="card space-y-4 p-5">
      <h2 className="text-lg font-semibold text-gray-900">{t("update.versions")}</h2>
      {state ? <>
        <dl className="grid grid-cols-1 gap-3 text-sm sm:grid-cols-2">
          <div><dt className="text-gray-500">{t("update.current")}</dt><dd className="mt-1 font-medium text-gray-900">{state.current.version} <span className="font-normal text-gray-500">({t(state.current.build_type === "release" ? "update.releaseBuild" : "update.sourceBuild")})</span></dd></div>
          <div><dt className="text-gray-500">{t("update.latest")}</dt><dd className="mt-1 font-medium text-gray-900">{state.latest?.tag_name || t("update.unknown")}</dd></div>
        </dl>
        {state.checked_at && <p className="text-xs text-gray-500">{t(state.cached ? "update.cached" : "update.checked", { time: new Date(state.checked_at).toLocaleString() })}</p>}
        {state.warning && <p role="alert" className="alert alert-warning">{t(errorKey(state.warning))}</p>}
        {!state.can_update && <p className="alert alert-warning">{t("update.unsupported")}</p>}
        {state.pending_version ? <p className="alert alert-info">{t("update.pendingVersion", { version: state.pending_version })}</p> : !state.warning && state.latest && state.current.build_type === "release" && <p className="text-sm text-gray-700">{t(state.has_update ? "update.available" : "update.upToDate")}</p>}
      </> : <p className="text-sm text-gray-500">{t("common.loading")}</p>}
      <div className="flex flex-wrap gap-3">
        <button className="btn btn-secondary" disabled={busy} onClick={check}>{t("update.check")}</button>
        <button className="btn btn-primary" disabled={busy || !state?.can_update || !state.has_update || !!state.pending_version || !!state.warning} onClick={() => apply(false)}>{t("update.apply")}</button>
      </div>
      {state?.latest && <details className="border-t border-gray-100 pt-3"><summary className="cursor-pointer text-sm text-primary-700">{t("update.notes")}</summary><pre className="mt-3 whitespace-pre-wrap break-words font-sans text-sm text-gray-700">{state.latest.body || t("update.noNotes")}</pre></details>}
    </section>
    <section className="card space-y-4 p-5">
      <h2 className="text-lg font-semibold text-gray-900">{t("update.rollback")}</h2>
      <p className="text-sm text-gray-500">{t("update.rollbackHelp")}</p>
      <label className="block text-sm text-gray-700" htmlFor="rollback-version">{t("update.targetLabel")}</label>
      <div className="flex flex-wrap gap-3">
        <select id="rollback-version" className="rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900" value={target} onChange={e => setTarget(e.target.value)} disabled={busy}>
          <option value="backup">{state?.backup_version ? t("update.localBackup", { version: state.backup_version }) : t("update.noBackup")}</option>
          {state?.rollbacks.map(v => <option key={v} value={v}>{v}</option>)}
        </select>
        <button className="btn btn-secondary" disabled={busy || !state?.can_update || !rollbackAvailable} onClick={() => apply(true)}>{t("update.rollback")}</button>
      </div>
    </section>
    <section className="card space-y-4 p-5">
      <h2 className="text-lg font-semibold text-gray-900">{t("update.restart")}</h2>
      <p className="text-sm text-gray-500">{t(state?.can_restart ? "update.restartHelp" : "update.restartUnavailable")}</p>
      <button className="btn btn-primary" disabled={busy || !state?.can_restart} onClick={restart}>{t("update.restart")}</button>
    </section>
    <p className="text-sm leading-6 text-gray-500">{t("update.containerHelp")}</p>
  </div>;
}
