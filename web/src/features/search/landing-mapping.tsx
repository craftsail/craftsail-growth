// SPDX-License-Identifier: AGPL-3.0-or-later

import {useEffect,useState} from "react";
import {Link} from "react-router-dom";
import {getPageMapping,type PageMapping} from "../../api";
import {useI18n} from "../../i18n";
export function LandingMapping({slug,landing,from,through,onClose}:{slug:string;landing:string;from:string;through:string;onClose:()=>void}){
 const {t,tn}=useI18n();const [data,setData]=useState<PageMapping|null>(null),[error,setError]=useState("");
 useEffect(()=>{let stale=false;setData(null);setError("");getPageMapping(slug,new URLSearchParams({value:landing,from,through})).then(d=>{if(!stale)setData(d)}).catch((e:Error)=>{if(!stale)setError(e.message)});return()=>{stale=true}},[slug,landing,from,through]);
 return <div className="card my-5 p-5"><div className="flex justify-between gap-4"><h3 className="break-all font-medium text-gray-900">{landing}</h3><button className="btn btn-secondary" onClick={onClose}>{t("common.close")}</button></div><p className="hint my-3">{t("gaSegments.mappingNote")}</p>{error&&<p role="alert" className="alert alert-error">{error}</p>}{!data&&!error&&<p>{t("common.loading")}</p>}{data&&<><p className="text-sm text-gray-700">{t(`gaSegments.${data.state}`)}</p><p className="my-2 text-sm text-gray-500">{t("gaSegments.hosts")}: {(data.hosts||[]).map(h=>h.hostname).join(" · ")||"—"}</p><p className="hint">{t("nav.google")} · {from} – {through} · {tn("search.imports.days",data.ga_coverage.covered_days,{total:data.ga_coverage.total_days})} / {tn("search.imports.days",data.gsc_coverage.covered_days,{total:data.gsc_coverage.total_days})}</p>{data.urls.map(url=><Link key={url} className="mt-2 block break-all text-sm text-primary-700 underline" to={`/p/${slug}/search/pages?${new URLSearchParams({value:url,from,through})}`}>{url}</Link>)}{!data.urls.length&&<p className="hint">{t("gaSegments.empty")}</p>}</>}</div>
}
