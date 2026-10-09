// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { patchProject, type SearchMode, type SearchObservation } from "../../api";
import { useAccess } from "../../app/access";
import { useI18n } from "../../i18n";

export function SearchObservationCard({ slug, value, onSaved }: { slug: string; value: SearchObservation; onSaved: () => void }) {
  const { t, tn } = useI18n();
  const { isAdmin } = useAccess();
  const [mode, setMode] = useState<SearchMode>(value.configured);
  const [minimum, setMinimum] = useState(String(value.min_impressions));
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);
  useEffect(() => { setMode(value.configured); setMinimum(String(value.min_impressions)); }, [value.configured, value.min_impressions]);

  async function save(event: React.FormEvent) {
    event.preventDefault();
    setSaving(true); setError(""); setSaved(false);
    try {
      await patchProject(slug, { search_mode: mode, search_min_impressions: Number(minimum) });
      setSaved(true); onSaved();
    } catch (e) { setError((e as Error).message); }
    finally { setSaving(false); }
  }

  return <div className="card mb-6 p-5">
    <h2 className="text-lg font-semibold text-gray-900">{t("search.observation.title")}</h2>
    <p className="mt-2 text-sm text-gray-700">{t(`search.observation.${value.mode}`)} · {t(`search.observation.${value.reason}`)}</p>
    <p className="hint mt-2">{t("search.observation.explanation")}</p>
    <p className="text-sm text-gray-700 mt-3">{value.coverage.from} – {value.coverage.through} · {t("search.observation.coverage", { current: value.coverage.covered_days, previous: value.previous_coverage.covered_days })}</p>
    <p className="text-sm text-gray-700 mt-1">{value.pages_with_impressions == null ? t("search.observation.pagesUnknown") : tn("search.observation.pages", value.pages_with_impressions)}</p>
    <div className="overflow-x-auto mt-3">
      <table className="table">
        <thead><tr><th>{t("search.observation.week")}</th><th>{t("search.clicks")}</th><th>{t("search.impressions")}</th><th>{t("search.coverage")}</th></tr></thead>
        <tbody>{(value.weeks || []).map(week => <tr key={week.from}>
          <td className="whitespace-nowrap">{week.from} – {week.through}</td><td>{week.clicks ?? "—"}</td><td>{week.impressions ?? "—"}</td><td>{week.covered_days}/7</td>
        </tr>)}</tbody>
      </table>
    </div>
    <p className="hint mt-2">{t("search.observation.unknown")}</p>
    <div className="flex flex-wrap gap-4 mt-3 text-sm text-primary-600">
      <Link to={`/p/${slug}/search/keywords`}>{t("search.observation.queries")}</Link>
      <Link to={`/p/${slug}/search/pages`}>{t("search.observation.pagesLink")}</Link>
    </div>
    {isAdmin && <form onSubmit={save} className="mt-4 border-t border-gray-100 pt-4">
      <div className="flex flex-wrap items-end gap-3">
        <label className="text-sm text-gray-700">{t("search.observation.mode")}<select className="input mt-1 block" value={mode} onChange={e => { setMode(e.target.value as SearchMode); setSaved(false); }}>
          {(["auto", "new_site", "established"] as const).map(item => <option key={item} value={item}>{t(`search.observation.${item}`)}</option>)}
        </select></label>
        <label className="text-sm text-gray-700">{t("search.observation.minimum")}<input className="input mt-1 block" type="number" required min={100} max={1000000} step={1} value={minimum} onChange={e => { setMinimum(e.target.value); setSaved(false); }} /></label>
        <button type="submit" className="btn btn-primary" disabled={saving}>{t(saving ? "common.saving" : "common.save")}</button>
      </div>
      <p className="hint mt-2">{t("search.observation.threshold")}</p>
      {saved && <p className="mt-2 text-sm text-emerald-700" role="status">{t("search.observation.saved")}</p>}
      {error && <p className="alert alert-error mt-2" role="alert">{error}</p>}
    </form>}
  </div>;
}
