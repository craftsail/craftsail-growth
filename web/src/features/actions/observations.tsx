// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useState } from "react";
import { evaluateObservation, listObservations, releaseTask, type OpportunityItem, type TaskObservation, type ObservationResult } from "../../api";
import { useAccess } from "../../app/access";
import { useI18n, type Key } from "../../i18n";

export function ObservationPanel({slug,item}:{slug:string;item:OpportunityItem}) {
 const {t,num,intl}=useI18n(),{canEdit}=useAccess();
 const [rows,setRows]=useState<TaskObservation[]>([]),[error,setError]=useState(""),[busy,setBusy]=useState(false),[notice,setNotice]=useState("");
 const [metric,setMetric]=useState(item.acceptance?.check?.startsWith("issue.absent:")?"technical":item.source==="search"?"clicks":"visibility");
 const code=item.task_code!;
 const load=()=>listObservations(slug,code).then(r=>setRows(r.items||[])).catch(e=>setError(e.message));
 useEffect(()=>{setRows([]);setError("");void load();},[slug,code]);
 const localNow=()=>{const d=new Date();return new Date(d.getTime()-d.getTimezoneOffset()*60000).toISOString().slice(0,16)};
 const reason=(r:string)=>t((`observation.${r==="window"?"windowReason":r==="baseline"?"baselineReason":r}`) as Key);
 const conclusion=(r:ObservationResult)=>t(`observation.${r.conclusion}` as Key);
 return <section className="space-y-3 rounded-lg border border-gray-200 bg-white p-3">
  <h3 className="font-semibold text-gray-900">{t("observation.title")}</h3>
  <p className="text-xs text-gray-500">{t("observation.note")}</p>
  {error&&<p role="alert" className="text-red-700">{error}</p>}{notice&&<p role="status" className="text-primary-700">{notice}</p>}
  {!rows.length&&<p className="text-sm text-gray-500">{t("observation.empty")}</p>}
  {rows.map(o=><article key={o.id} className="space-y-2 rounded-lg border border-gray-200 p-3">
   <h4 className="font-medium">{o.hypothesis}</h4>
   <p className="text-xs text-gray-500">{t(`observation.${o.metric}` as Key)} · {new Date(o.released_at*1000).toLocaleString(intl)} · {o.owner}</p>
   <p>{t("observation.baseline")}: {o.baseline_from} – {o.baseline_through} · {o.baseline.valid?num(o.baseline.value):t("observation.notMeasured")}</p>
   <p>{t("observation.followup")}: {o.followup_from} – {o.followup_through}</p>
   {o.guardrails&&<p>{t("observation.guardrails")}: {o.guardrails}</p>}{o.notes&&<p>{t("observation.notes")}: {o.notes}</p>}
   {(o.control_urls?.length||0)>0&&<p className="break-all">{t("observation.controls")}: {o.control_urls.join(", ")} · {o.control_baseline.valid?num(o.control_baseline.value):t("observation.notMeasured")}</p>}
   {(o.results||[]).map(r=><div key={r.id} className="border-t border-gray-100 pt-2">
    <p className="font-medium text-gray-900">{conclusion(r)}</p><p className="text-xs text-gray-500">{t("observation.version",{id:r.id,date:new Date(r.created_at*1000).toLocaleString(intl)})}</p>
    <p>{t("observation.reason",{reason:reason(r.reason)})}</p>
    {r.followup.valid&&<p>{t("observation.followup")}: {num(r.followup.value)}</p>}
    {r.control_followup?.valid&&<p>{t("observation.controls")}: {num(r.control_followup.value)}</p>}
    {r.notes&&<p>{r.notes}</p>}
   </div>)}
   {canEdit&&<form className="flex flex-wrap items-end gap-2" onSubmit={async e=>{e.preventDefault();const notes=String(new FormData(e.currentTarget).get("notes")||"");setBusy(true);setError("");try{const r=await evaluateObservation(slug,code,o.id,notes);setNotice(`${conclusion(r)}: ${reason(r.reason)}`);await load();}catch(e){setError((e as Error).message)}finally{setBusy(false)}}}>
    <label className="flex-1 text-xs">{t("observation.notes")}<input name="notes" className="input mt-1 w-full" maxLength={8000}/></label><button disabled={busy} className="btn btn-secondary btn-sm">{t("observation.evaluate")}</button>
   </form>}
  </article>)}
  {canEdit&&<details><summary className="cursor-pointer text-primary-700">{t("observation.new")}</summary>
   <form className="mt-3 grid gap-3 sm:grid-cols-2" onSubmit={async e=>{e.preventDefault();const form=e.currentTarget,f=new FormData(form),val=(k:string)=>String(f.get(k)||""),lines=(k:string)=>val(k).split(/\n/).map(v=>v.trim()).filter(Boolean);setBusy(true);setError("");setNotice("");try{await releaseTask(slug,code,{hypothesis:val("hypothesis"),metric,guardrails:val("guardrails"),urls:lines("urls"),qids:val("qids").split(",").map(s=>s.trim()).filter(Boolean),control_urls:lines("controls"),owner:val("owner"),effort_hours:Number(val("effort")),released_at:Math.floor(new Date(val("released")).getTime()/1000),wait_days:Number(val("wait")),window_days:Number(val("window")),engine:val("engine"),access:val("access"),notes:val("notes")});await load();setNotice(t("observation.saved"));form.closest("details")?.removeAttribute("open");}catch(e){setError((e as Error).message)}finally{setBusy(false)}}}>
    <label className="sm:col-span-2">{t("observation.hypothesis")}<input className="input mt-1 w-full" name="hypothesis" required maxLength={4000}/></label>
    <label>{t("observation.metric")}<select className="input mt-1 w-full" value={metric} onChange={e=>setMetric(e.target.value)}>{["clicks","impressions","visibility",...(item.acceptance?.check?.startsWith("issue.absent:")?["technical"]:[])].map(v=><option key={v} value={v}>{t(`observation.${v}` as Key)}</option>)}</select></label>
    <label>{t("observation.released")}<input className="input mt-1 w-full" name="released" type="datetime-local" defaultValue={localNow()} max={localNow()} required/></label>
    {metric==="visibility"?<>
     <label>{t("observation.qids")}<input className="input mt-1 w-full" name="qids" defaultValue={item.qid||""} required/></label>
     <label>{t("observation.engine")}<input className="input mt-1 w-full" name="engine" required/></label>
     <label>{t("observation.access")}<select className="input mt-1 w-full" name="access"><option value="api">{t("observation.api")}</option><option value="web">{t("observation.web")}</option></select></label>
    </>:<label className="sm:col-span-2">{t("observation.urls")}<textarea className="input mt-1 w-full" name="urls" defaultValue={(item.urls||[]).join("\n")} required rows={2}/></label>}
    {metric!=="visibility"&&<label className="sm:col-span-2">{t("observation.controls")}<textarea className="input mt-1 w-full" name="controls" rows={2}/></label>}
    {(["wait","window","effort"] as const).map(name=><label key={name}>{t(`observation.${name}`)}<input className="input mt-1 w-full" name={name} type="number" min={name==="window"?28:0} max={name==="effort"?10000:90} step={name==="effort"?0.25:1} defaultValue={name==="window"?28:0} required/></label>)}
    <label>{t("observation.owner")}<input className="input mt-1 w-full" name="owner" maxLength={128}/></label>
    <label className="sm:col-span-2">{t("observation.guardrails")}<input className="input mt-1 w-full" name="guardrails" maxLength={4000}/></label>
    <label className="sm:col-span-2">{t("observation.notes")}<textarea className="input mt-1 w-full" name="notes" maxLength={8000}/></label>
    <p className="text-xs text-gray-500 sm:col-span-2">{t("observation.rules")}</p>
    <button disabled={busy} className="btn btn-primary sm:col-span-2">{t("observation.record")}</button>
   </form>
  </details>}
 </section>;
}
