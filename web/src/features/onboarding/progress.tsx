// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { getProgress, type ProjectProgress } from "../../api";
import { useAccess } from "../../app/access";
import { useI18n } from "../../i18n";

export function SetupProgress({ slug, hasAudit, noSite }: { slug: string; hasAudit: boolean; noSite: boolean }) {
  const { t, date } = useI18n();
  const { isAdmin } = useAccess();
  const [progress, setProgress] = useState<ProjectProgress | null>(null);
  const [error, setError] = useState("");
  const [refresh, setRefresh] = useState(0);
  useEffect(() => {
    let disposed = false;setProgress(null);setError("");
    getProgress(slug).then(p => { if (!disposed) setProgress(p); }).catch(e => { if (!disposed) setError(e.message); });
    return () => { disposed = true; };
  }, [slug, refresh]);
  const next = !progress?.brand_confirmed ? "brand" : !progress.questions_confirmed ? "questions" : !progress.first_value_at && hasAudit ? "evidence" : "actions";
  const path = { brand: "settings/brand", questions: "settings/questions", evidence: "audit", actions: "opportunities" }[next];
  return <div className="mt-4 border-t border-gray-100 pt-4">
    <h3 className="text-sm font-semibold text-gray-900">{t("progress.title")}</h3>
    {error && <p className="mt-2 text-sm text-red-700" role="alert">{error}<button className="btn btn-secondary ml-2" onClick={() => setRefresh(n => n + 1)}>{t("common.refresh")}</button></p>}
    {!progress && !error && <p className="hint">{t("common.loading")}</p>}
    {progress && <>
      <ul className="mt-2 space-y-1 text-sm text-gray-700">
        {(["brand", "questions"] as const).map(step => <li key={step}>{t(`progress.${step}Title`)} · {t(progress[`${step}_confirmed`] ? "progress.confirmed" : progress[`${step}_confirmed_at`] ? "progress.changed" : "progress.pending")}</li>)}
        <li>{progress.first_value_at ? t("progress.valueRecorded", { date: date(progress.first_value_at * 1000) }) : t("progress.valuePending")}</li>
      </ul>
      <Link className="inline-block mt-3 text-sm" to={`/p/${slug}/${path}`}>{t(`progress.next_${next}`)}</Link>
      {progress.brand_confirmed && progress.questions_confirmed && <div className="mt-3 flex flex-wrap gap-4 text-sm">
        {isAdmin && <Link to={`/p/${slug}/settings/google`}>{t("firstCheck.google")}</Link>}
        <Link to={`/p/${slug}/settings/schedule`}>{t("firstCheck.ai")}</Link>
      </div>}
      {noSite && !progress.first_value_at && <p className="hint mt-2">{t("progress.noSite")}</p>}
    </>}
  </div>;
}
