// SPDX-License-Identifier: AGPL-3.0-or-later

import { useState } from "react";
import { EmptyState } from "../../components/EmptyState";
import { EvidenceBadge } from "../../components/metrics/EvidenceBadge";
import { BLOCKS, SCORE_PARTS } from "../../labels";
import { AuditActions, useAudit } from "./shared";
import { useI18n } from "../../i18n";

// Content readiness per page. The score thresholds are correlations from the
// citation dataset, so the column is labeled observational.
export function AuditPages() {
  const a = useAudit();
  const { t } = useI18n();
  const [open, setOpen] = useState("");
  const pages = [...(a.rep?.pages || [])].sort((x, y) => x.score - y.score);
  return (
    <section className="space-y-4">
      <p className="page-description">{t("audit.pages.description")}</p>
      <AuditActions a={a} />
      {a.loaded && !a.rep && <EmptyState title={t("audit.notAudited")}>{t("audit.notAuditedHint")}</EmptyState>}
      {pages.length > 0 && (
        <div className="table-container"><table className="table">
          <thead>
            <tr className="bg-gray-50 text-left text-xs text-gray-500">
              <th className="px-3 py-2">{t("audit.pages.page")}</th>
              <th className="px-3 py-2">{t("audit.pages.grade")}</th>
              <th className="px-3 py-2"><span className="inline-flex items-center gap-1">{t("audit.pages.score")} <EvidenceBadge level="observational" /></span></th>
              <th className="px-3 py-2 text-right">{t("audit.pages.words")}</th>
              <th className="px-3 py-2 text-right">{t("audit.pages.findings")}</th>
            </tr>
          </thead>
          <tbody>
            {pages.map((p) => {
              const expanded = open === p.url;
              return (
                <PageRows key={p.url}>
                  <tr className="cursor-pointer border-t border-gray-100" onClick={() => setOpen(expanded ? "" : p.url)}>
                    <td className="px-3 py-2"><div className="font-medium text-gray-900">{p.title || p.url}</div><div className="break-all text-xs text-gray-500">{p.url}</div></td>
                    <td className="px-3 py-2">{p.grade}</td>
                    <td className="px-3 py-2 tabular-nums">{p.score.toFixed(0)}</td>
                    <td className="px-3 py-2 text-right tabular-nums">{p.word_count}</td>
                    <td className="px-3 py-2 text-right tabular-nums">{p.issue_codes?.length ?? 0}</td>
                  </tr>
                  {expanded && (
                    <tr className="border-t border-gray-100 bg-gray-50/50">
                      <td colSpan={5} className="space-y-2 px-3 py-3 text-xs text-gray-700">
                        <div className="flex flex-wrap gap-x-4 gap-y-1">
                          {SCORE_PARTS.map((d) => <span key={d.key}>{t(d.label)} {Math.round(Number(p.dimensions?.[d.key] ?? 0))}/{d.max}</span>)}
                        </div>
                        <div>{t("audit.pages.missingBlocks", { list: BLOCKS.filter((b) => !p.blocks?.[b.key]).map((b) => t(b.label)).join(", ") || t("common.none") })}</div>
                        {(p.issue_codes?.length ?? 0) > 0 && <div>{t("audit.pages.findingsList", { list: p.issue_codes!.join(", ") })}</div>}
                        <div>{t("audit.pages.jsonld", { list: p.jsonld_types?.join(", ") || t("common.none") })}</div>
                      </td>
                    </tr>
                  )}
                </PageRows>
              );
            })}
          </tbody>
        </table></div>
      )}
    </section>
  );
}

function PageRows({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}
