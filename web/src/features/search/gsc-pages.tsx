// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { getGscPages, type GscPageRow } from "../../api";
import { PositionPill, downloadCsv } from "./keyword-table";
import { useI18n } from "../../i18n";
import { useAccess } from "../../app/access";

export function GscPages() {
  const { slug } = useParams();
  const { t, num } = useI18n();
  const { isAdmin } = useAccess();
  const [rows, setRows] = useState<GscPageRow[]>([]);
  const [err, setErr] = useState("");
  useEffect(() => {
    if (!slug) return;
    getGscPages(slug).then((d) => setRows(d.items || [])).catch((e: Error) => setErr(e.message));
  }, [slug]);
  return (
    <section>
      <p className="page-description mb-5">{t("search.pagesDescription")}</p>
      <div className="toolbar">
        <button className="btn btn-secondary" onClick={() => downloadCsv("pages.csv",
          ["url", "position", "clicks", "impressions", "ctr"],
          rows.map((r) => [r.url, r.position.toFixed(2), String(r.clicks), String(r.impressions), (r.ctr * 100).toFixed(3) + "%"]))}
        >{t("common.exportCsv")}</button>
      </div>
      {err && <div className="alert alert-error">{err}</div>}
      {!rows.length && !err && <p className="hint">{t("search.noPages")} {isAdmin && <Link to={`/p/${slug}/settings/google`}>{t("search.connectAndSync")}</Link>}</p>}
      {rows.length > 0 && (
        <div className="table-container"><table className="table">
          <thead><tr><th>{t("search.url")}</th><th>{t("search.positionCol")}</th><th>{t("search.clicks")}</th><th>{t("search.impressions")}</th><th>{t("search.ctr")}</th></tr></thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.url}>
                <td>{r.url}</td>
                <td><PositionPill position={r.position} band={r.band} /></td>
                <td>{num(Math.round(r.clicks))}</td>
                <td>{num(Math.round(r.impressions))}</td>
                <td>{(r.ctr * 100).toFixed(2)}%</td>
              </tr>
            ))}
          </tbody>
        </table></div>
      )}
    </section>
  );
}
