// SPDX-License-Identifier: AGPL-3.0-or-later

import { useI18n, type Key } from "../../i18n";

const LEVELS = ["standard", "vendor", "experiment", "observational", "heuristic"];

// The label carries the meaning; the neutral outline never relies on color.
export function EvidenceBadge({ level, refs }: { level?: string; refs?: string[] }) {
  const { t } = useI18n();
  if (!level) return null;
  const known = LEVELS.includes(level);
  const title = [known ? t(`evidence.help.${level}` as Key) : "", refs && refs.length ? t("evidence.sources", { list: refs.join(", ") }) : ""].filter(Boolean).join(" ");
  return (
    <span className="whitespace-nowrap rounded border border-gray-300 px-1.5 py-0.5 text-[11px] font-medium text-gray-600" title={title}>
      {known ? t(`evidence.${level}` as Key) : level}
    </span>
  );
}
