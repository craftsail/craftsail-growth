// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { getKeywords, getSavedKeywords, saveKeyword, type KeywordRow } from "../../api";
import { KeywordTable, downloadCsv } from "./keyword-table";
import { useI18n } from "../../i18n";
import { useAccess } from "../../app/access";

export function Keywords() {
  const { slug } = useParams();
  const { t } = useI18n();
  const { canEdit, isAdmin } = useAccess();
  const [rows, setRows] = useState<KeywordRow[]>([]);
  const [saved, setSaved] = useState<Set<string>>(new Set());
  const [q, setQ] = useState("");
  const [err, setErr] = useState("");

  function load() {
    if (!slug) return;
    getKeywords(slug).then((d) => setRows(d.items || [])).catch((e: Error) => setErr(e.message));
    getSavedKeywords(slug).then((d) => setSaved(new Set((d.items || []).map((x) => x.query)))).catch(() => undefined);
  }
  useEffect(load, [slug]);

  const [sort, setSort] = useState<"clicks" | "position" | "impressions">("clicks");
  const shown = rows
    .filter((r) => !q || r.query.toLowerCase().includes(q.toLowerCase()))
    .slice()
    .sort((a, b) => sort === "position" ? a.position - b.position : b[sort] - a[sort]);
  return (
    <section>
      <p className="page-description mb-5">{t("search.kwDescription")}</p>
      <div className="toolbar">
        {isAdmin && <Link className="btn btn-secondary" to={`/p/${slug}/settings/google`}>{t("search.sync")}</Link>}
        <button className="btn btn-secondary" onClick={() => downloadCsv("keywords.csv",
          ["query", "position", "clicks", "impressions", "ctr"],
          shown.map((r) => [r.query, r.position.toFixed(2), String(r.clicks), String(r.impressions), (r.ctr * 100).toFixed(3) + "%"]))}
        >{t("common.exportCsv")}</button>
        <input className="max-w-60 input" value={q} onChange={(e) => setQ(e.target.value)} placeholder={t("search.filterQueries")} />
        <select className="max-w-36 input" value={sort} onChange={(e) => setSort(e.target.value as typeof sort)}>
          <option value="clicks">{t("search.clicks")}</option>
          <option value="impressions">{t("search.impressions")}</option>
          <option value="position">{t("search.positionCol")}</option>
        </select>
      </div>
      {err && <div className="alert alert-error">{err}</div>}
      {!err && rows.length === 0 && (
        <p className="hint">{t("search.noKeywords")} {isAdmin && <Link to={`/p/${slug}/settings/google`}>{t("search.connectAndSync")}</Link>}</p>
      )}
      <KeywordTable rows={shown} saved={saved} onSave={canEdit ? async (query) => {
        if (!slug) return;
        await saveKeyword(slug, query);
        setSaved(new Set([...saved, query]));
      } : undefined} />
    </section>
  );
}
