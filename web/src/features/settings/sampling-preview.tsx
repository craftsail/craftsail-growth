// SPDX-License-Identifier: AGPL-3.0-or-later

import {useEffect,useState} from "react";
import {samplingPreview,startJob,type SamplingPreview as Preview} from "../../api";
import {useI18n} from "../../i18n";
import {useAccess} from "../../app/access";

export function SamplingPreview({slug,onStarted}:{slug:string;onStarted:()=>void}) {
 const {t,tn,num}=useI18n(),{canEdit}=useAccess();
 const [platforms,setPlatforms]=useState(""),[repeat,setRepeat]=useState(1),[limit,setLimit]=useState(0),[data,setData]=useState<Preview|null>(null),[error,setError]=useState(""),[busy,setBusy]=useState(false);
 useEffect(()=>{let alive=true;setData(null);const timer=setTimeout(()=>{samplingPreview(slug,platforms,repeat,limit).then(r=>{if(alive){setData(r);setError("")}}).catch(e=>{if(alive)setError(e.message)})},200);return()=>{alive=false;clearTimeout(timer)}},[slug,platforms,repeat,limit]);
 return <section className="card space-y-3 p-5"><h2 className="card-title">{t("sampling.preview")}</h2>
  <div className="grid gap-3 sm:grid-cols-3"><label className="sm:col-span-3">{t("sampling.platforms")}<input name="sample_platforms" className="input mt-1 w-full" value={platforms} onChange={e=>setPlatforms(e.target.value.replace(/\s/g,""))}/></label>
   <label>{t("sampling.repeat")}<input name="sample_repeat" className="input mt-1 w-full" type="number" min={1} max={10} value={repeat} onChange={e=>setRepeat(Number(e.target.value))}/></label>
   <label>{t("sampling.limit")}<input name="sample_limit" className="input mt-1 w-full" type="number" min={0} max={1000} value={limit} onChange={e=>setLimit(Number(e.target.value))}/></label>
  </div>
  {data&&<><p className="font-medium">{tn("sampling.calls",data.calls)} · {tn("sampling.questions",data.questions)}</p><p className="text-sm text-gray-500">{t("sampling.tokens",{n:num(data.est_tokens)})} · {t("sampling.version")}: {data.revision.slice(0,12)}</p><p className="text-sm text-gray-500">{t("sampling.ready",{engines:data.available.join(", ")||"—"})}</p></>}
  <p className="text-xs text-gray-500">{t("sampling.previewNote")}</p>{error&&<p role="alert" className="text-sm text-red-700">{error}</p>}
  {canEdit&&<button disabled={busy||!data?.calls} className="btn btn-primary" onClick={async()=>{setBusy(true);setError("");try{await startJob(slug,"sample",{platforms,repeat,limit});onStarted()}catch(e){setError((e as Error).message)}finally{setBusy(false)}}}>{t("sampling.start")}</button>}
 </section>;
}
