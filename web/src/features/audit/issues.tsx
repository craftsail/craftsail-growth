// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useMemo, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { getAuditIssues, type AuditIssueRow, type Reference } from "../../api";
import { EmptyState } from "../../components/EmptyState";
import { EvidenceBadge } from "../../components/metrics/EvidenceBadge";
import { useI18n, type Key } from "../../i18n";
import { ruleText } from "../../i18n/rules";

type Group = { code: string; first: AuditIssueRow; rows: AuditIssueRow[] };

const SEV_ORDER: Record<string, number> = { critical: 0, warning: 1, info: 2 };
const SURFACE_WORD: Record<string, string> = { seo: "SEO", geo: "GEO", both: "SEO + GEO" };

function Select({ value, onChange, options, label }: { value: string; onChange: (v: string) => void; options: [string, string][]; label: string }) {
  return (
    <select aria-label={label} className="input w-auto py-1.5" value={value} onChange={(e) => onChange(e.target.value)}>
      {options.map(([v, l]) => <option key={v} value={v}>{l}</option>)}
    </select>
  );
}

export function AuditIssues() {
  const { slug = "" } = useParams();
  const { t, locale } = useI18n();
  const [params, setParams] = useSearchParams();
  const [rows, setRows] = useState<AuditIssueRow[] | null>(null);
  const [refs, setRefs] = useState<Record<string, Reference>>({});
  const [open, setOpen] = useState("");
  const severity = params.get("severity") || "";
  const layer = params.get("layer") || "";
  const surface = params.get("surface") || "";

  useEffect(() => {
    getAuditIssues(slug).then((d) => { setRows(d.items || []); setRefs(d.references || {}); }).catch(() => setRows([]));
  }, [slug]);

  const groups = useMemo(() => {
    const by = new Map<string, Group>();
    for (const r of rows || []) {
      if ((severity && r.severity !== severity) || (layer && r.layer !== layer) || (surface && r.surface !== surface)) continue;
      const g = by.get(r.code) || { code: r.code, first: r, rows: [] };
      g.rows.push(r);
      by.set(r.code, g);
    }
    return [...by.values()].sort((a, b) =>
      (SEV_ORDER[a.first.severity] - SEV_ORDER[b.first.severity]) || Number(a.first.blocked) - Number(b.first.blocked) || b.rows.length - a.rows.length);
  }, [rows, severity, layer, surface]);

  function set(key: string, v: string) {
    const next = new URLSearchParams(params);
    if (v) next.set(key, v); else next.delete(key);
    setParams(next, { replace: true });
  }

  return (
    <section className="space-y-4">
      <p className="page-description">{t("audit.issues.description")}</p>
      <div className="flex flex-wrap gap-2">
        <Select label={t("audit.issues.severity")} value={severity} onChange={(v) => set("severity", v)} options={[["", t("audit.issues.allSeverities")], ["critical", t("severity.critical")], ["warning", t("severity.warning")], ["info", t("severity.info")]]} />
        <Select label={t("audit.issues.layer")} value={layer} onChange={(v) => set("layer", v)} options={[["", t("audit.issues.allLayers")], ["access", t("layer.names.access")], ["discover", t("layer.names.discover")], ["understand", t("layer.names.understand")], ["cite", t("layer.names.cite")]]} />
        <Select label={t("audit.issues.appliesTo")} value={surface} onChange={(v) => set("surface", v)} options={[["", t("audit.issues.seoAndGeo")], ["seo", t("audit.issues.seoOnly")], ["geo", t("audit.issues.geoOnly")], ["both", t("audit.issues.both")]]} />
      </div>
      {rows === null && <p className="text-sm text-gray-500">{t("common.loading")}</p>}
      {rows && groups.length === 0 && <EmptyState title={t("audit.issues.noFindings")}>{t("audit.issues.noFindingsHint")}</EmptyState>}
      {groups.length > 0 && (
        <div className="table-container"><table className="table">
          <thead>
            <tr className="bg-gray-50 text-left text-xs text-gray-500">
              <th className="px-3 py-2">{t("audit.issues.rule")}</th><th className="px-3 py-2">{t("audit.issues.severity")}</th><th className="px-3 py-2">{t("audit.issues.layer")}</th>
              <th className="px-3 py-2">{t("audit.issues.appliesTo")}</th><th className="px-3 py-2">{t("audit.issues.evidence")}</th><th className="px-3 py-2 text-right">{t("audit.issues.found")}</th>
            </tr>
          </thead>
          <tbody>
            {groups.map((g) => {
              const r = g.first;
              const expanded = open === g.code;
              return (
                <FragmentRow key={g.code}>
                  <tr className={`cursor-pointer border-t border-gray-100 ${r.blocked ? "text-gray-400" : "text-gray-800"}`} onClick={() => setOpen(expanded ? "" : g.code)}>
                    <td className="px-3 py-2"><span className="font-medium">{ruleText(locale, g.code, "title", r.title)}</span> <code className="text-xs text-gray-500">{g.code}</code>{r.blocked && <span className="ml-1 text-xs">{t("audit.issues.blocked")}</span>}</td>
                    <td className="px-3 py-2">{t(`severity.${r.severity}` as Key)}</td>
                    <td className="px-3 py-2">{t(`layer.names.${r.layer}` as Key)}</td>
                    <td className="px-3 py-2">{SURFACE_WORD[r.surface]}</td>
                    <td className="px-3 py-2"><EvidenceBadge level={r.evidence} refs={r.refs} /></td>
                    <td className="px-3 py-2 text-right tabular-nums">{g.rows.length}</td>
                  </tr>
                  {expanded && (
                    <tr className="border-t border-gray-100 bg-gray-50/50">
                      <td colSpan={6} className="space-y-2 px-3 py-3 text-gray-700">
                        {r.blocked && <p>{t("audit.issues.blockedNote")}</p>}
                        <p><span className="font-medium text-gray-900">{t("audit.issues.why")}. </span>{ruleText(locale, g.code, "why", r.why)}</p>
                        <p><span className="font-medium text-gray-900">{t("audit.issues.how")}. </span>{ruleText(locale, g.code, "fix", r.fix)}</p>
                        {r.refs.length > 0 && (
                          <p className="text-xs text-gray-600">{t("audit.issues.sources")}: {r.refs.map((k, i) => (
                            <span key={k}>{i > 0 && "; "}{refs[k]?.url ? <a href={refs[k].url} target="_blank" rel="noreferrer">{refs[k].cite}</a> : k}</span>
                          ))}</p>
                        )}
                        {g.rows.some((x) => x.url) && (
                          <ul className="list-disc pl-5 text-xs text-gray-600">
                            {g.rows.filter((x) => x.url).slice(0, 30).map((x) => <li key={x.id} className="break-all">{x.url}</li>)}
                          </ul>
                        )}
                        {!r.blocked && r.severity !== "info" && (
                          <Link to={`/p/${slug}/opportunities?key=${encodeURIComponent("audit:" + g.code)}`} >{t("audit.issues.openInPlan")}</Link>
                        )}
                      </td>
                    </tr>
                  )}
                </FragmentRow>
              );
            })}
          </tbody>
        </table></div>
      )}
    </section>
  );
}

function FragmentRow({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}
