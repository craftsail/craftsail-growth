// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { getCompetitors, saveCompetitors, type Competitor } from "../../api";
import { useI18n } from "../../i18n";
import { HelpTip } from "../../components/HelpTip";
import { useAccess } from "../../app/access";

export function Competitors() {
  const { t } = useI18n();
  const { canEdit } = useAccess();
  const { slug } = useParams();
  const [items, setItems] = useState<Competitor[]>([]);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!slug) return;
    getCompetitors(slug)
      .then((d) => setItems(d.items || []))
      .catch((e: Error) => setErr(e.message));
  }, [slug]);

  function add() {
    setItems([...items, { name: "", aliases: [], confirmed: false }]);
  }

  async function save() {
    if (!slug) return;
    setBusy(true);
    setErr("");
    try {
      await saveCompetitors(slug, items.filter((c) => c.name.trim()));
    } catch (e) {
      setErr((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <section>
      <p className="page-description">{t("competitors.description")}</p>
      {canEdit && (
        <div className="mb-4 flex gap-2">
          <button className="btn btn-primary" onClick={add}>{t("competitors.add")}</button>
          <HelpTip id="addCompetitor" />
          <button className="btn btn-secondary" disabled={busy} onClick={save}>{busy ? t("common.saving") : t("common.save")}</button>
          <HelpTip id="saveCompetitors" />
        </div>
      )}
      {err && <div className="alert alert-error">{err}</div>}
      {items.length === 0 && <p className="hint">{t("competitors.empty")}</p>}
      <fieldset disabled={!canEdit} className="contents">
      <div className="flex flex-col gap-3">
        {items.map((c, i) => (
          <div className="card p-5" key={i}>
            <div className="grid grid-cols-[1fr_1fr_140px_40px] items-center gap-2">
              <input className="input" placeholder={t("competitors.name")} value={c.name} onChange={(e) => {
                const n = [...items]; n[i] = { ...c, name: e.target.value }; setItems(n);
              }} />
              <input className="input" placeholder={t("competitors.aliases")} value={(c.aliases || []).join(",")}
                onChange={(e) => {
                  const n = [...items];
                  n[i] = { ...c, aliases: e.target.value.split(",").map((s) => s.trim()).filter(Boolean) };
                  setItems(n);
                }} />
              <label className="text-[13px] text-gray-500">
                <input type="checkbox" checked={c.confirmed !== false}
                  onChange={(e) => {
                    const n = [...items]; n[i] = { ...c, confirmed: e.target.checked }; setItems(n);
                  }} /> {t("competitors.confirmed")}
              </label>
              <button className="btn btn-secondary" aria-label={t("competitors.remove")} onClick={() => setItems(items.filter((_, j) => j !== i))}>×</button>
            </div>
          </div>
        ))}
      </div>
      </fieldset>
    </section>
  );
}
