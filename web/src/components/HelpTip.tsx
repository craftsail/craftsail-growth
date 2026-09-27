// SPDX-License-Identifier: AGPL-3.0-or-later

import { useNavigate, useParams } from "react-router-dom";
import { IconHelpCircle } from "@tabler/icons-react";
import { TIPS, type TipId } from "../app/tips";
import { useI18n, type Key } from "../i18n";

// HelpTip is the "?" next to a button: hover or focus shows what the button
// does; clicking opens that button's entry in the help center.
export function HelpTip({ id }: { id: TipId }) {
  const { t } = useI18n();
  const { slug } = useParams();
  const nav = useNavigate();
  const tip = TIPS[id];
  const text = t(`tips.${id}` as Key);
  const to = `${slug ? `/p/${slug}` : ""}/help#${tip.topic}/${id}`;
  return (
    <span className="group relative -ml-1 inline-flex align-middle">
      <button type="button" className="rounded-full p-0.5 text-gray-400 hover:text-primary-600 focus:text-primary-600 focus:outline-none" aria-label={`${text} ${t("tips.more")}`} onClick={() => nav(to)}>
        <IconHelpCircle size={16} stroke={1.8} />
      </button>
      <span role="tooltip" className="pointer-events-none absolute bottom-full left-1/2 z-50 mb-2 w-64 -translate-x-1/2 rounded-lg border border-gray-200 bg-white px-3 py-2 text-left text-xs font-normal leading-5 text-gray-700 opacity-0 shadow-lg transition-opacity group-hover:opacity-100 group-focus-within:opacity-100">
        {text}
        <span className="mt-1 block text-primary-600">{t("tips.more")}</span>
      </span>
    </span>
  );
}
