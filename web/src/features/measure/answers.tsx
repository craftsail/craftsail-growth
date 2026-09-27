// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useMemo, useState } from "react";
import { Link, useOutletContext, useParams } from "react-router-dom";
import { getSampleSheet, getSamples, importSamples, overrideSample, type Project, type SampleRow } from "../../api";
import { EmptyState } from "../../components/EmptyState";
import { useI18n } from "../../i18n";
import { HelpTip } from "../../components/HelpTip";
import { useAccess } from "../../app/access";

const accessOf = (mode: string) => (!mode || mode === "api" ? "API" : "Web");

function escapeRe(s: string) {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

// Highlight brand and competitor names so a reviewer can check the reading.
function Highlighted({ text, brand, others }: { text: string; brand: string[]; others: string[] }) {
  const names = [...brand, ...others].filter((n) => n.trim().length > 1);
  if (!names.length) return <>{text}</>;
  const re = new RegExp(`(${names.map(escapeRe).join("|")})`, "gi");
  const brandSet = new Set(brand.map((b) => b.toLowerCase()));
  return (
    <>
      {text.split(re).map((part, i) =>
        names.some((n) => n.toLowerCase() === part.toLowerCase())
          ? <mark key={i} className={brandSet.has(part.toLowerCase()) ? "bg-blue-100 text-gray-900" : "bg-gray-200 text-gray-900"}>{part}</mark>
          : <span key={i}>{part}</span>,
      )}
    </>
  );
}

export function Answers() {
  const { slug = "" } = useParams();
  const { t } = useI18n();
  const { canEdit } = useAccess();
  const ctx = useOutletContext<{ project?: Project } | undefined>();
  const project = ctx?.project;
  const [rows, setRows] = useState<SampleRow[] | null>(null);
  const [sel, setSel] = useState<number>(0);
  const [engine, setEngine] = useState("");
  const [access, setAccess] = useState("");
  const [review, setReview] = useState(false);
  const [err, setErr] = useState("");
  const [sheet, setSheet] = useState("");
  const [busy, setBusy] = useState("");

  function load() {
    getSamples(slug).then((d) => setRows(d.items || [])).catch((e: Error) => setErr(e.message));
  }
  useEffect(load, [slug]);

  const engines = useMemo(() => [...new Set((rows || []).map((r) => r.platform_name || r.platform))].sort(), [rows]);
  const shown = (rows || []).filter((r) =>
    (!engine || (r.platform_name || r.platform) === engine) && (!access || accessOf(r.sample_mode) === access) && (!review || r.needs_review));
  const current = shown.find((r) => r.id === sel) || shown[0];
  const brandNames = [project?.name || "", ...(project?.brand?.aliases || [])].filter(Boolean);

  async function correct(row: SampleRow, body: { mentioned?: boolean; negative?: boolean }) {
    setErr("");
    try {
      const updated = await overrideSample(slug, row.id, body);
      setRows((prev) => (prev || []).map((r) => (r.id === row.id ? { ...r, ...updated } : r)));
    } catch (e) {
      setErr((e as Error).message);
    }
  }

  async function run(kind: "sheet" | "import") {
    setBusy(kind);
    setErr("");
    try {
      if (kind === "sheet") setSheet((await getSampleSheet(slug)).markdown || "");
      else {
        await importSamples(slug, sheet);
        setSheet("");
        load();
      }
    } catch (e) {
      setErr((e as Error).message);
    } finally {
      setBusy("");
    }
  }

  const ghost = "btn btn-secondary";
  return (
    <section className="space-y-4">
      <p className="page-description">{t("measure.answers.description")}</p>
      <div className="flex flex-wrap items-center gap-2">
        <select aria-label={t("measure.answers.engine")} className="input w-auto py-1.5" value={engine} onChange={(e) => setEngine(e.target.value)}>
          <option value="">{t("measure.answers.allEngines")}</option>
          {engines.map((e) => <option key={e} value={e}>{e}</option>)}
        </select>
        <select aria-label={t("measure.answers.access")} className="input w-auto py-1.5" value={access} onChange={(e) => setAccess(e.target.value)}>
          <option value="">{t("measure.answers.apiAndWeb")}</option>
          <option value="API">API</option>
          <option value="Web">Web</option>
        </select>
        <label className="inline-flex items-center gap-1.5 text-sm text-gray-700"><input type="checkbox" checked={review} onChange={(e) => setReview(e.target.checked)} />{t("measure.answers.needsReviewOnly")}</label>
        <span className="flex-1" />
        <button type="button" className={ghost} disabled={!!busy} onClick={() => run("sheet")}>{busy === "sheet" ? t("measure.answers.exporting") : t("measure.answers.exportSheet")}</button>
        <HelpTip id="exportSheet" />
      </div>
      {err && <p className="text-sm text-red-700">{err}</p>}
      {sheet && (
        <div className="space-y-2">
          <textarea className="input h-64 font-mono text-xs" value={sheet} onChange={(e) => setSheet(e.target.value)} aria-label={t("measure.answers.sheet")} />
          {canEdit && <button type="button" className="btn btn-primary" disabled={!!busy} onClick={() => run("import")}>{busy === "import" ? t("measure.answers.importing") : t("measure.answers.importSheet")}</button>}{canEdit && <HelpTip id="importSheet" />}
        </div>
      )}
      {rows && shown.length === 0 && <EmptyState title={t("measure.answers.empty")}>{t("measure.answers.emptyHint")}</EmptyState>}
      {shown.length > 0 && (
        <div className="grid min-h-[520px] overflow-hidden rounded-2xl border border-gray-200 bg-white lg:grid-cols-[340px_minmax(0,1fr)]">
          <ul className="max-h-[70vh] overflow-auto border-b border-gray-200 lg:border-b-0 lg:border-r">
            {shown.map((r) => (
              <li key={r.id}>
                <button type="button" className={`w-full px-3 py-2 text-left text-sm ${current?.id === r.id ? "bg-gray-100" : "hover:bg-gray-50"}`} onClick={() => setSel(r.id)}>
                  <div className="truncate text-gray-900">{r.question_text || r.qid}</div>
                  <div className="text-xs text-gray-500">
                    {r.platform_name || r.platform} · {accessOf(r.sample_mode)} · {(r.sampled_on || "").slice(0, 10)} · {!r.ok ? t("measure.answers.failed") : r.mentioned ? t("measure.answers.mentioned") : t("measure.answers.notMentioned")}
                    {r.brand_in_question ? ` · ${t("measure.answers.branded")}` : ""}{r.needs_review ? ` · ${t("measure.answers.needsReview")}` : ""}{r.manual_override ? ` · ${t("measure.answers.corrected")}` : ""}
                  </div>
                </button>
              </li>
            ))}
          </ul>
          {current && (
            <article className="space-y-4 overflow-auto p-5 text-sm">
              <header>
                <h2 className="card-title">{current.question_text}</h2>
                <p className="mt-1 text-xs text-gray-500">
                  {current.platform_name || current.platform} · {accessOf(current.sample_mode)}{current.env && current.env !== "api" ? ` · ${current.env}` : ""} · {(current.sampled_on || "").slice(0, 10)} ·{" "}
                  <Link to={`/p/${slug}/ai/prompts/${current.qid}`}>{t("measure.answers.question", { id: current.qid })}</Link>
                </p>
              </header>
              {!current.ok ? <p className="text-red-700">{t("measure.answers.failedLine", { error: current.error || t("measure.answers.noAnswer") })}</p> : (
                <>
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="text-gray-700">{current.mentioned ? t("measure.answers.readMentioning") : t("measure.answers.readNotMentioning")}{current.rank ? t("measure.answers.position", { n: current.rank }) : ""}{current.negative ? t("measure.answers.negative") : ""}.</span>
                    {canEdit && <button type="button" className={ghost} onClick={() => correct(current, { mentioned: !current.mentioned })}>{current.mentioned ? t("measure.answers.markNotMentioned") : t("measure.answers.markMentioned")}</button>}
                    {canEdit && <button type="button" className={ghost} onClick={() => correct(current, { negative: !current.negative })}>{current.negative ? t("measure.answers.markNotNegative") : t("measure.answers.markNegative")}</button>}{canEdit && <HelpTip id="correct" />}
                  </div>
                  <div className="whitespace-pre-wrap rounded-xl border border-gray-100 bg-gray-50 p-4 leading-6 text-gray-800">
                    <Highlighted text={current.answer} brand={brandNames} others={current.competitors_mentioned || []} />
                  </div>
                  <div>
                    <h3 className="stat-label mb-1">{t("measure.answers.citations")}</h3>
                    {(current.cited?.length ?? 0) === 0 ? <p className="text-gray-500">{t("common.none")}</p> : (
                      <ol className="list-decimal space-y-0.5 pl-5">
                        {current.cited!.map((c, i) => <li key={i} className="break-all"><a href={c.url} target="_blank" rel="noreferrer">{c.title || c.url}</a></li>)}
                      </ol>
                    )}
                  </div>
                  <div>
                    <h3 className="stat-label mb-1">{t("measure.answers.searches")}</h3>
                    {(() => {
                      const q = (current.web_queries || []).filter(Boolean);
                      if (q.length === 0) return <p className="text-gray-500">{t("measure.answers.notReported")}</p>;
                      if (q.every((x) => x === "unavailable")) return <p className="text-gray-500">{t("measure.answers.hiddenQueries")}</p>;
                      return <ul className="list-disc pl-5">{q.filter((x) => x !== "unavailable").map((x) => <li key={x}>{x}</li>)}</ul>;
                    })()}
                  </div>
                </>
              )}
            </article>
          )}
        </div>
      )}
    </section>
  );
}
