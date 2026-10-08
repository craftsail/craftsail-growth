// SPDX-License-Identifier: AGPL-3.0-or-later

import {Link} from "react-router-dom";
import type {Project} from "../../api";
import {useI18n,type Key} from "../../i18n";
export function DemoGuide({project}:{project:Project}){
 const {t}=useI18n();const n=t;const geo=project.demo_scenario==="geo";
 const steps:{id:"evidence"|"compare"|"action"|"release"|"review";page:Key;path:string}[]=[
  {id:"evidence",page:geo?"nav.audit":"nav.search",path:geo?"audit/issues":"search"},
  {id:"compare",page:geo?"nav.visibility":"nav.tabs.keywords",path:geo?"ai/visibility":"search/keywords"},
  {id:"action",page:"nav.actionPlan",path:"opportunities"},
  {id:"release",page:"nav.actionPlan",path:"opportunities?tab=progress"},
  {id:"review",page:"nav.reports",path:"reports"}
 ];
 return <details id="demo-guide" className="card p-5" open><summary className="cursor-pointer font-semibold text-gray-900">{t("demoGuide.title")}</summary><p className="hint my-3">{t("demoGuide.intro")}</p><ol className="grid gap-4 md:grid-cols-2">{steps.map(s=><li key={s.id} className="rounded-lg border border-gray-100 p-4"><h3 className="font-medium text-gray-900">{t(`demoGuide.${s.id}`)}</h3><p className="my-2 text-sm text-gray-700">{t(`demoGuide.${s.id}Text`,{page:n(s.page)})}</p><Link className="text-sm text-primary-700 underline" to={`/p/${project.slug}/${s.path}`}>{n(s.page)}</Link></li>)}</ol><p className="my-3 text-sm text-gray-700">{t("demoGuide.switch")}</p><div className="flex flex-wrap gap-4 text-sm"><Link className="text-primary-700 underline" to="/p/quillpad-new-site/overview">{t("demoGuide.new_site")}</Link><Link className="text-primary-700 underline" to="/p/quillpad-established/overview">{t("demoGuide.established")}</Link><Link className="text-primary-700 underline" to="/p/quillpad/overview">{t("demoGuide.geo")}</Link></div></details>
}
