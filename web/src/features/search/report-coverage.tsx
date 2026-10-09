// SPDX-License-Identifier: AGPL-3.0-or-later

import { Link, useParams } from "react-router-dom";
import type { GrainCoverage } from "../../api";
import { useI18n } from "../../i18n";

export function ReportCoverage({ value, report }: { value?: GrainCoverage; report: "query" | "page" | "country" | "device" }) {
  const { t, tn } = useI18n();
  const { slug } = useParams();
  if (!value) return null;
  return <div className="mb-5 rounded-lg border border-gray-200 bg-gray-50 p-4 text-sm text-gray-700">
    <p>{t(report === "page" ? "search.grain.page" : report==="query" ? "search.grain.query" : "searchSegments.grain")}</p>
    <p className="mt-1">{value.from} – {value.through} · {tn("search.imports.days", value.covered_days, { total: value.total_days })}</p>
    {value.state === "covered" && (!value.quality?.known || value.quality.sampled || value.quality.thresholded || value.quality.other_row || value.quality.restricted) && <p className="mt-2 text-amber-700">{t("prioritization.quality")}</p>}
    {value.state !== "covered" && <p className="mt-2 text-amber-700">{t(value.state === "missing" ? "search.grain.missing" : "search.grain.partial")} <Link className="text-primary-700 underline" to={`/p/${slug}/search`}>{t("search.grain.viewSync")}</Link></p>}
  </div>;
}
