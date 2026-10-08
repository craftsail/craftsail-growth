// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import { questionLibraries, type QuestionLibrary, getQuestions, runningJob, saveQuestions, startJob, waitJob, type JobRow, type Question } from "../../api";
import { TagsInput } from "../../components/tags-input";
import { DEFAULT_GROUP, PROMPT_GROUPS } from "../../labels";
import { useI18n } from "../../i18n";
import { ReviewConfirmation } from "../onboarding/review";
import { HelpTip } from "../../components/HelpTip";
import { useAccess } from "../../app/access";

// Only used for prompts that are not saved yet; saved prompts use the
// server's system tag, which applies the single branded-prompt rule.
function mentionsBrand(text: string, names: string[], site: string) {
  const q = text.toLowerCase();
  if (names.some((n) => n && q.includes(n.toLowerCase()))) return true;
  const host = site.toLowerCase().replace(/^https?:\/\//, "").replace(/^www\./, "").split(/[/?]/)[0];
  if (!host) return false;
  return q.includes(host) || q.includes(host.split(".")[0]);
}

export function Questions() {
  const { t,tn,intl } = useI18n();
 const [libraries,setLibraries]=useState<QuestionLibrary[]>([]);
  const { canEdit } = useAccess();
  const { slug } = useParams();
  const [items, setItems] = useState<Question[]>([]);
  const [names, setNames] = useState<string[]>([]);
  const [site, setSite] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [note, setNote] = useState("");
  const [drafting, setDrafting] = useState(false);
  const [revision, setRevision] = useState("");
  const [dirty, setDirty] = useState(false);
  function editItems(next: Question[]) { setDirty(true); setItems(next); }

  function load() {
    if (!slug) return;
 questionLibraries(slug).then(r=>setLibraries(r.items||[])).catch(e=>setErr(e.message));
    getQuestions(slug)
      .then((d) => {
        setItems(d.items || []); setRevision(d.review_revision); setDirty(false);
        setNames([d.brand || "", ...(d.aliases || [])]);
        setSite(d.site || "");
      })
      .catch((e: Error) => setErr(e.message));
  }
  useEffect(load, [slug]);

  // Redraft asks a connected model to write a new prompt library (and
  // competitor list) from the site text. It replaces the current lists. It
  // runs as a background job because the model calls take minutes; coming
  // back to the page picks the running job up again.
  async function follow(job: JobRow) {
    setDrafting(true); setErr(""); setNote(t("questions.draftingBg"));
    try {
      const done = await waitJob(job.id);
      if (done.status !== "done") setErr(done.error || done.status);
      else if (/no model answered/.test(done.log || "")) setNote(t("questions.noModel"));
      else setNote("");
      load();
    } catch (e) {
      setErr((e as Error).message);
    } finally {
      setDrafting(false);
    }
  }
  useEffect(() => {
    if (!slug) return;
    runningJob(slug, ["bootstrap"]).then((j) => { if (j) follow(j); }).catch(() => undefined);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [slug]);

  async function redraft() {
    if (!slug) return;
    if (!window.confirm(t("questions.redraftConfirm"))) return;
    setErr("");
    try {
      const r = await startJob(slug, "bootstrap", { skip_llm: false });
      await follow(r.job);
    } catch (e) {
      setErr((e as Error).message);
    }
  }

  const [picked, setPicked] = useState<string[]>([]);
  const suggestions = useMemo(() => [...new Set(items.flatMap((item) => item.tags || []))].sort(), [items]);

  function add() {
    const n = items.length + 1;
    editItems([...items, { qid: `q${String(n).padStart(3, "0")}`, group: DEFAULT_GROUP, text: "", enabled: true, tags: [] }]);
  }

  function patch(i: number, next: Partial<Question>) {
    editItems(items.map((item, index) => (index === i ? { ...item, ...next } : item)));
  }

  async function save() {
    if (!slug) return;
    setBusy(true);
    setErr("");
    try {
      await saveQuestions(slug, items);
      const saved = await getQuestions(slug);
      setItems(saved.items || []); setRevision(saved.review_revision); setDirty(false);
    } catch (e) {
      setErr((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  const allPicked = items.length > 0 && picked.length === items.length;
  return (
    <section>
      <ReviewConfirmation key={slug} slug={slug || ""} kind="questions" revision={revision} dirty={dirty || busy || drafting} />
      <p className="page-description mt-4">{t("questions.description")}</p>
      {canEdit && <div className="mb-4 flex items-center gap-2">
        <button type="button" className="btn btn-primary" onClick={add}>{t("questions.add")}</button>
        <HelpTip id="addQuestion" />
        <button type="button" className="btn btn-secondary" disabled={busy} onClick={save}>{busy ? t("common.saving") : t("common.save")}</button>
        <HelpTip id="saveQuestions" />
        <button type="button" className="btn btn-secondary ml-auto" disabled={drafting} onClick={redraft} title={t("questions.redraftHint")}>{drafting ? t("questions.drafting") : t("questions.redraft")}</button>
        <HelpTip id="redraft" />
      </div>}
      <fieldset disabled={!canEdit} className="contents">
      {picked.length > 0 && (
        <div className="alert alert-info mb-3 mt-0 flex items-center justify-between">
          <span>{t("questions.selected", { n: picked.length })}</span>
          <div className="flex gap-2">
            <button type="button" className="btn btn-secondary btn-sm" onClick={() => editItems(items.map((item) => picked.includes(item.qid) ? { ...item, enabled: true } : item))}>{t("questions.enable")}</button>
            <button type="button" className="btn btn-secondary btn-sm" onClick={() => editItems(items.map((item) => picked.includes(item.qid) ? { ...item, enabled: false } : item))}>{t("questions.disable")}</button>
            <button type="button" className="btn btn-ghost btn-sm" onClick={() => setPicked([])}>{t("questions.clear")}</button>
          </div>
        </div>
      )}
      {err && <div className="alert alert-error">{err}</div>}
      {note && <div className="alert alert-info">{note}</div>}
      <div className="card px-5 py-3">
      <div className="hidden items-center gap-2 border-b border-gray-200 px-1 pb-2 text-sm text-gray-500 md:grid md:grid-cols-[2rem_minmax(0,1fr)_9rem_9.5rem_16rem_3rem]">
        <input type="checkbox" aria-label={t("questions.selectAll")} checked={allPicked} onChange={() => setPicked(allPicked ? [] : items.map((item) => item.qid))} />
        <span title={t("questions.questionHint")}>{t("questions.question")}</span>
        <span title={t("questions.groupHint")}>{t("questions.group")}</span>
        <span title={t("questions.systemHint")}>{t("questions.system")}</span>
        <span title={t("questions.tagsHint")}>{t("questions.tags")}</span>
        <span />
      </div>
      <div className="divide-y divide-gray-100">
        {items.map((q, i) => {
          const on = q.enabled !== false;
          return (
            <div key={q.qid + i} className={`grid items-center gap-2 py-2 md:grid-cols-[2rem_minmax(0,1fr)_9rem_9.5rem_16rem_3rem] ${on ? "" : "opacity-60"}`}>
              <input type="checkbox" aria-label={t("questions.select")} checked={picked.includes(q.qid)} onChange={() => setPicked(picked.includes(q.qid) ? picked.filter((id) => id !== q.qid) : [...picked, q.qid])} />
              <div className="space-y-1"><input className="input w-full" value={q.text} placeholder={t("questions.placeholder")} onChange={(e) => patch(i, { text: e.target.value })} />
              <select className="input text-xs" aria-label={t("sampling.languageFilter")} value={q.language||""} onChange={e=>patch(i,{language:e.target.value})}><option value="">{t("sampling.unknown")}</option>{([['en','english'],['zh','chinese'],['pt','portuguese']] as const).map(([v,k])=><option key={v} value={v}>{t(`weeklyReport.${k}`)}</option>)}</select></div>
              <select aria-label={t("questions.group")} className="input w-auto" value={q.group} onChange={(e) => patch(i, { group: e.target.value })}>
                {PROMPT_GROUPS.map((g) => <option key={g.key} value={g.key}>{t(g.label)}</option>)}
              </select>
              <TagsInput value={q.system_tags?.length ? q.system_tags : [mentionsBrand(q.text, names, site) ? t("questions.branded") : t("questions.unbranded")]} onChange={() => {}} disabled />
              <TagsInput value={q.tags || []} options={suggestions} onChange={(tags) => patch(i, { tags })} />
              <button type="button" role="switch" aria-checked={on} aria-label={on ? t("questions.disableOne") : t("questions.enableOne")} className={`relative h-5 w-9 rounded-full ${on ? "bg-primary-500" : "bg-gray-300"}`} onClick={() => patch(i, { enabled: !on })}>
                <span className={`absolute top-0.5 h-4 w-4 rounded-full bg-white transition-all ${on ? "left-4" : "left-0.5"}`} />
              </button>
            </div>
          );
        })}
      </div>
      </div>
      </fieldset>
      <details className="card mt-4 p-4"><summary className="cursor-pointer font-medium">{t("sampling.history")}</summary><p className="my-3 text-sm text-gray-500">{t("sampling.note")}</p><ol className="space-y-3">{libraries.map(lib=><li key={lib.id}><details><summary className="cursor-pointer break-all text-sm">{lib.revision.slice(0,12)} · {lib.language||t("sampling.unknown")} · {lib.region||t("sampling.unknown")} · {tn("sampling.questions",lib.questions.length)} · {new Date(lib.created_at*1000).toLocaleString(intl)}</summary><ul className="list-disc space-y-1 pl-5 text-sm text-gray-700">{lib.questions.map(q=><li key={q.qid}>{q.qid} · {q.text}</li>)}</ul></details></li>)}</ol></details>
    </section>
  );
}
