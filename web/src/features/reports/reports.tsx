// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { buildReport, getReport } from "../../api";
import { EmptyState } from "../../components/EmptyState";
import { useI18n } from "../../i18n";
import { HelpTip } from "../../components/HelpTip";
import { useAccess } from "../../app/access";

type ReportDoc = { language?: string; markdown: string; html: string; report_on?: string; on?: string };

function download(name: string, body: string, type: string) {
  const a = document.createElement("a");
  a.href = URL.createObjectURL(new Blob([body], { type }));
  a.download = name;
  a.click();
  URL.revokeObjectURL(a.href);
}

// The report prints the same numbers as the dashboard: visibility with its
// interval, audit findings with evidence levels, opportunities and actions.
export function Reports() {
  const { t } = useI18n();
 const [language,setLanguage]=useState("en");
  const { canEdit } = useAccess();
  const { slug = "" } = useParams();
  const [doc, setDoc] = useState<ReportDoc | null>(null);
  const [loaded, setLoaded] = useState(false);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  useEffect(() => {
    getReport(slug).then((d) => {setDoc(d as ReportDoc);setLanguage((d as ReportDoc).language || "en")}).catch(() => setDoc(null)).finally(() => setLoaded(true));
  }, [slug]);

  async function build() {
    setBusy(true);
    setErr("");
    try {
      setDoc(await buildReport(slug,language));
    } catch (e) {
      setErr((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  const day = (doc?.on || doc?.report_on || "").slice(0, 10);
  const ghost = "btn btn-secondary";
  return (
    <section className="space-y-4">
      <p className="page-description">{t("reports.description")}</p>
      <div className="flex flex-wrap items-center gap-2">
        {canEdit && <label className="flex items-center gap-2">{t("weeklyReport.language")}<select className="input" value={language} onChange={e=>setLanguage(e.target.value)}>{([['en','english'],['zh','chinese'],['pt','portuguese']] as const).map(([v,k])=><option value={v} key={v}>{t(`weeklyReport.${k}`)}</option>)}</select></label>}
        {canEdit && <button type="button" className="btn btn-primary" disabled={busy} onClick={build}>{busy ? t("reports.building") : t("reports.build")}</button>}{canEdit && <HelpTip id="buildReport" />}
        <button type="button" className={ghost} disabled={!doc} onClick={() => doc && download(`${slug}-report-${day}.html`, doc.html, "text/html")}>{t("reports.html")}</button>
        <button type="button" className={ghost} disabled={!doc} onClick={() => doc && download(`${slug}-report-${day}.md`, doc.markdown, "text/markdown")}>{t("reports.markdown")}</button>
        <HelpTip id="downloadReport" />
        {doc?.language && <span className="text-sm text-gray-500">{t("weeklyReport.currentLanguage",{language:doc.language})}</span>}
        {day && <span className="text-sm text-gray-500">{t("reports.latest", { day })}</span>}
        {err && <span className="text-sm text-red-700">{err}</span>}
      </div>
      {loaded && !doc && <EmptyState title={t("reports.empty")}>{t("reports.emptyHint")}</EmptyState>}
      {doc && <iframe title={t("reports.frame")} className="h-[75vh] w-full rounded-2xl border border-gray-200 bg-white" srcDoc={doc.html} sandbox="" />}
    </section>
  );
}
