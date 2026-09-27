// SPDX-License-Identifier: AGPL-3.0-or-later

import { useI18n } from "../../i18n";
import { HelpTip } from "../../components/HelpTip";
import type { KeywordRow } from "../../api";

export function PositionPill({ position, band }: { position: number; band?: string }) {
  const kind = band || (position > 0 && position <= 3 ? "top3" : position <= 10 ? "top10" : position <= 20 ? "top20" : "deep");
  const text = position > 0 ? position.toFixed(1) : "—";
  const tone = kind === "top3" ? "bg-emerald-100 text-emerald-800" : kind === "top10" ? "bg-amber-100 text-amber-800" : kind === "top20" ? "bg-blue-100 text-blue-800" : "bg-gray-100 text-gray-600";
  return <span className={`inline-flex min-w-12 justify-center rounded-md px-2 py-0.5 text-xs font-semibold ${tone}`}>{text}</span>;
}

export function KeywordTable({ rows, onSave, saved }: {
  rows: KeywordRow[];
  onSave?: (query: string) => void;
  saved?: Set<string>;
}) {
  const { t, num } = useI18n();
  if (!rows.length) return <p className="hint">{t("search.noQueries")}</p>;
  return (
    <div className="table-container"><table className="table">
      <thead>
        <tr>
          <th>{t("search.query")}</th>
          <th>{t("search.positionCol")}</th>
          <th>{t("search.clicks")}</th>
          <th>{t("search.impressions")}</th>
          <th>{t("search.ctr")}</th>
          {onSave && <th><HelpTip id="saveKeyword" /></th>}
        </tr>
      </thead>
      <tbody>
        {rows.map((r) => (
          <tr key={r.query}>
            <td>{r.query}</td>
            <td><PositionPill position={r.position} band={r.band} /></td>
            <td>{num(Math.round(r.clicks))}</td>
            <td>{num(Math.round(r.impressions))}</td>
            <td>{(r.ctr * 100).toFixed(2)}%</td>
            {onSave && (
              <td>
                <button className="btn btn-secondary btn-sm" onClick={() => onSave(r.query)}>
                  {saved?.has(r.query) ? t("search.saved") : t("search.save")}
                </button>
              </td>
            )}
          </tr>
        ))}
      </tbody>
    </table></div>
  );
}

export function downloadCsv(name: string, header: string[], rows: string[][]) {
  const esc = (s: string) => `"${s.replace(/"/g, '""')}"`;
  const body = [header.map(esc).join(","), ...rows.map((r) => r.map(esc).join(","))].join("\n");
  const a = document.createElement("a");
  a.href = URL.createObjectURL(new Blob([body], { type: "text/csv" }));
  a.download = name;
  a.click();
}
