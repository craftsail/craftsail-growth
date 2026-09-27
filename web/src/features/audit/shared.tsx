// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { getAudit, runAudit, runningJob, startJob, waitJob, type AuditReport } from "../../api";
import { useI18n } from "../../i18n";
import { HelpTip } from "../../components/HelpTip";
import { useAccess } from "../../app/access";

export function useAudit() {
  const { slug = "" } = useParams();
  const [rep, setRep] = useState<AuditReport | null>(null);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState("");
  const [loaded, setLoaded] = useState(false);

  function load() {
    getAudit(slug)
      .then((r) => { setRep(r); setErr(""); })
      .catch(() => setRep(null))
      .finally(() => setLoaded(true));
  }
  useEffect(load, [slug]);

  // A crawl can take minutes, so it runs as a background job. Returning to
  // the page while it runs shows it as busy and refreshes when it ends.
  async function crawlJob(id: number) {
    const done = await waitJob(id);
    if (done.status !== "done") throw new Error(done.error || done.status);
  }
  useEffect(() => {
    if (!slug) return;
    runningJob(slug, ["crawl"]).then(async (j) => {
      if (!j) return;
      setBusy("crawl");
      try { await crawlJob(j.id); await runAudit(slug); load(); } catch (e) { setErr((e as Error).message); } finally { setBusy(""); }
    }).catch(() => undefined);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [slug]);

  async function run(kind: "crawl" | "audit") {
    setBusy(kind);
    setErr("");
    try {
      if (kind === "crawl") await crawlJob((await startJob(slug, "crawl")).job.id);
      await runAudit(slug);
      load();
    } catch (e) {
      setErr((e as Error).message);
    } finally {
      setBusy("");
    }
  }

  async function exportMD() {
    setErr("");
    try {
      const res = await fetch(`/api/projects/${slug}/audit.md`, { credentials: "include" });
      if (!res.ok) throw new Error(res.statusText);
      const blob = await res.blob();
      const a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = `${slug}-site-audit.md`;
      a.click();
      URL.revokeObjectURL(a.href);
    } catch (e) {
      setErr((e as Error).message);
    }
  }

  return { slug, rep, err, busy, loaded, run, exportMD, reload: load };
}

export function AuditActions({ a }: { a: ReturnType<typeof useAudit> }) {
  const { t } = useI18n();
  const { canEdit } = useAccess();
  const ghost = "btn btn-secondary";
  return (
    <div className="flex flex-wrap items-center gap-2">
      {canEdit && (
        <>
          <button type="button" className="btn btn-primary" disabled={!!a.busy} onClick={() => a.run("crawl")}>
            {a.busy === "crawl" ? t("audit.actions.crawling") : t("audit.actions.crawl")}
          </button>
          <HelpTip id="auditCrawl" />
          <button type="button" className={ghost} disabled={!!a.busy} onClick={() => a.run("audit")}>{a.busy === "audit" ? t("audit.actions.auditing") : t("audit.actions.again")}</button>
          <HelpTip id="auditAgain" />
        </>
      )}
      <button type="button" className={ghost} disabled={!a.rep || !!a.busy} onClick={a.exportMD} title={t("audit.actions.exportHint")}>{t("audit.actions.export")}</button>
      <HelpTip id="exportAI" />
      {a.err && <span className="text-sm text-red-700">{a.err}</span>}
    </div>
  );
}
