// SPDX-License-Identifier: AGPL-3.0-or-later

import { useId, useState } from "react";
import { createPortal } from "react-dom";
import { useNavigate, useParams } from "react-router-dom";
import { IconHelpCircle } from "@tabler/icons-react";
import { TIPS, type TipId } from "../app/tips";
import { useI18n, type Key } from "../i18n";

// HelpTip is the "?" next to a button: hover or focus shows what the button
// does; clicking opens that button's entry in the help center.
export function HelpTip({ id }: { id: TipId }) {
  const { t } = useI18n();
 const tooltipId=useId();
 const [anchor,setAnchor]=useState<{left:number;bottom:number}|null>(null);
 const show=(element:HTMLElement)=>{const r=element.getBoundingClientRect();setAnchor({left:Math.max(16,Math.min(r.left+r.width/2-128,window.innerWidth-272)),bottom:window.innerHeight-r.top+8})};
  const { slug } = useParams();
  const nav = useNavigate();
  const tip = TIPS[id];
  const text = t(`tips.${id}` as Key);
  const to = `${slug ? `/p/${slug}` : ""}/help#${tip.topic}/${id}`;
  return (
    <span className="group relative -ml-1 inline-flex align-middle">
      <button type="button" className="rounded-full p-0.5 text-gray-400 hover:text-primary-600 focus:text-primary-600 focus:outline-none" onMouseEnter={e=>show(e.currentTarget)} onMouseLeave={()=>setAnchor(null)} onFocus={e=>show(e.currentTarget)} onBlur={()=>setAnchor(null)} aria-describedby={anchor?tooltipId:undefined} aria-label={`${text} ${t("tips.more")}`} onClick={() => nav(to)}>
        <IconHelpCircle size={16} stroke={1.8} />
      </button>
      {anchor && createPortal(<span id={tooltipId} role="tooltip" style={anchor} className="pointer-events-none fixed z-50 w-64 max-w-[calc(100vw-32px)] rounded-lg border border-gray-200 bg-white px-3 py-2 text-left text-xs font-normal leading-5 text-gray-700 shadow-lg">
        {text}
        <span className="mt-1 block text-primary-600">{t("tips.more")}</span>
      </span>,document.body)}
    </span>
  );
}
