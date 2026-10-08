// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useState } from "react";
import { confirmProgress, getProgress, type ProjectProgress } from "../../api";
import { useAccess } from "../../app/access";
import { useI18n } from "../../i18n";

export function ReviewConfirmation({ slug, kind, revision, dirty = false, auditID }: { slug: string; kind: "brand" | "questions" | "audit_helpful"; revision?: string; dirty?: boolean; auditID?: number }) {
  const { t, date } = useI18n();
  const { canEdit } = useAccess();
  const [progress, setProgress] = useState<ProjectProgress | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [refresh, setRefresh] = useState(0);
  useEffect(() => {
    let disposed = false; setProgress(null); setError("");
    getProgress(slug).then(p => { if (!disposed) setProgress(p); }).catch(e => { if (!disposed) setError(e.message); });
    return () => { disposed = true; };
  }, [slug, revision, auditID, refresh]);
  const helpful = kind === "audit_helpful";
  const matches = helpful || (kind === "brand" ? progress?.brand_current_revision : progress?.questions_current_revision) === revision;
  const confirmed = !dirty && matches && (helpful ? !!progress?.first_value_at : kind === "brand" ? progress?.brand_confirmed : progress?.questions_confirmed);
  const ready = helpful ? !!auditID : !!revision && matches && !dirty && (kind === "brand" ? progress?.brand_ready : progress?.questions_ready);
  async function confirm() {
    setBusy(true); setError("");
    try { setProgress(await confirmProgress(slug, { kind, revision, audit_id: auditID })); }
    catch (e) { setError((e as Error).message); }
    finally { setBusy(false); }
  }
  return <div className="card p-5 space-y-2">
    <h2 className="card-title">{t(`progress.${kind}Title`)}</h2>
    <p className="text-sm text-gray-700">{t(helpful ? "progress.helpfulHint" : "progress.reviewHint")}</p>
    {confirmed ? <p className="text-sm text-emerald-700" role="status">{helpful ? t("progress.valueRecorded", { date: date(progress!.first_value_at! * 1000) }) : t("progress.confirmed")}</p> : <>
      {dirty && <p className="text-sm text-amber-700">{t("progress.unsaved")}</p>}
      {!dirty && !matches && progress && <p className="text-sm text-amber-700">{t("progress.reload")}</p>}
      {!dirty && matches && !ready && progress && <p className="text-sm text-gray-500">{t(helpful ? "progress.noAudit" : kind === "brand" ? "progress.brandRequired" : "progress.questionsRequired")}</p>}
      {canEdit && <button className="btn btn-primary" disabled={!ready || busy || !progress} onClick={confirm}>{t(busy ? "common.saving" : helpful ? "progress.helpful" : "progress.confirm")}</button>}
    </>}
    {error && <p role="alert" className="text-sm text-red-700 break-words">{error}<button className="btn btn-secondary ml-2" onClick={() => setRefresh(n => n + 1)}>{t("common.refresh")}</button></p>}
  </div>;
}
