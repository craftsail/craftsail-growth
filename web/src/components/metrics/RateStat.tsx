// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ReactNode } from "react";
import { useI18n } from "../../i18n";

type Interval = { lo: number; hi: number } | null | undefined;

// A rate on the 0-100 scale with its 95% interval and sample size. Numbers use
// text ink, never a status or series color; "not measured" is text, not 0%.
export function RateStat({
  label,
  help,
  value,
  ci,
  n,
  lowSample,
  suffix = "%",
  footer,
}: {
  label: string;
  help?: string;
  value: number | null | undefined;
  ci?: Interval;
  n?: number;
  lowSample?: boolean;
  suffix?: string;
  footer?: ReactNode;
}) {
  const { t } = useI18n();
  const measured = value !== null && value !== undefined && !Number.isNaN(value);
  return (
    <div className="card p-5">
      <div className="flex items-center gap-1 text-xs font-semibold text-gray-500">
        <span>{label}</span>
        {help && <span className="cursor-help rounded-full border border-gray-300 px-1 text-[10px] leading-4 text-gray-400" title={help} aria-label={help}>?</span>}
      </div>
      <div className="stat-value mt-1 tabular-nums">
        {measured ? `${value!.toFixed(0)}${suffix}` : <span className="text-base font-normal text-gray-400">{t("common.notMeasured")}</span>}
      </div>
      {measured && (ci || n !== undefined) && (
        <p className="mt-1 text-xs text-gray-500">
          {ci && <>{t("measure.ci", { lo: ci.lo.toFixed(0), hi: ci.hi.toFixed(0) })}</>}
          {ci && n !== undefined && " · "}
          {n !== undefined && <>n = {n}</>}
          {lowSample && <span className="ml-1.5 rounded border border-gray-300 px-1 text-[11px] text-gray-600">{t("measure.smallSample")}</span>}
        </p>
      )}
      {footer && <div className="mt-2 text-xs text-gray-500">{footer}</div>}
    </div>
  );
}
