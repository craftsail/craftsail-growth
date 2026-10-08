// SPDX-License-Identifier: AGPL-3.0-or-later

import type { CTRReference } from "../../api";
import { useI18n, type Key } from "../../i18n";

export function CTRReferenceCard({value: v}: {value: CTRReference}) {
 const {t,tn,num}=useI18n();
 const reason=["quality","insufficient","brand","rank"].includes(v.reason)?v.reason:"insufficient";
 return <aside className="my-3 rounded-lg border border-gray-200 bg-white p-3 text-sm">
  <h3 className="font-medium text-gray-900">{t("prioritization.ctrTitle")}</h3>
  <p className="my-1 text-gray-700">{v.ctr != null ? t("prioritization.ctrReady",{band:v.band,scope:t(v.scope==="segment"?"prioritization.segment":"prioritization.site"),from:v.from,through:v.through,version:v.version,ctr:num(v.ctr,{style:"percent",maximumFractionDigits:2})}) : t("prioritization.ctrMissing",{reason:t(`prioritization.${reason}` as Key)})}</p>
  {v.ctr != null && <p>{tn("prioritization.queries",v.queries)} · {num(v.impressions)}</p>}
  <p className="mt-1 text-xs text-gray-500">{t("prioritization.ctrNote")}</p>
 </aside>;
}
