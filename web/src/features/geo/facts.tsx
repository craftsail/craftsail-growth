// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useState, type ReactNode } from "react";
import { useOutletContext, useParams } from "react-router-dom";
import { IconPlus, IconTrash } from "@tabler/icons-react";
import { getBrand, saveBrand, type BrandFacts, type KeyNumber, type Project } from "../../api";
import { useI18n } from "../../i18n";
import { ReviewConfirmation } from "../onboarding/review";
import { HelpTip } from "../../components/HelpTip";
import { useAccess } from "../../app/access";

const EMPTY: BrandFacts = {
  aliases: [], products: [], industry: "", target_users: "", definition: "",
  disambiguation: [], key_numbers: [], suitable: [], unsuitable: [],
};

// Brand: the facts engines should repeat about you. Saved as structured
// fields; the facts card (used for llms.txt and JSON-LD) is generated from them.
export function Facts() {
  const { t } = useI18n();
  const { canEdit } = useAccess();
  const { slug = "" } = useParams();
  const ctx = useOutletContext<{ onProject?: (p: Project) => void; project?: Project } | undefined>();
  const [name, setName] = useState("");
  const [site, setSite] = useState("");
  const [b, setB] = useState<BrandFacts>(EMPTY);
  const [card, setCard] = useState("");
  const [err, setErr] = useState("");
  const [ok, setOk] = useState("");
  const [busy, setBusy] = useState(false);
  const [revision, setRevision] = useState("");
  const [dirty, setDirty] = useState(false);

  useEffect(() => {
    if (!slug) return;
    setRevision("");
    getBrand(slug).then((d) => {
      setRevision(d.review_revision); setDirty(false);
      setName(d.name); setSite(d.site); setCard(d.facts_markdown || "");
      setB({ ...EMPTY, ...d.brand, key_numbers: d.brand.key_numbers || [] });
    }).catch((e: Error) => setErr(e.message));
  }, [slug]);

  const set = <K extends keyof BrandFacts>(k: K, v: BrandFacts[K]) => { setOk(""); setDirty(true); setB({ ...b, [k]: v }); };

  async function save() {
    setBusy(true); setErr(""); setOk("");
    try {
      const r = await saveBrand(slug, name, b);
      setCard(r.facts_markdown);
      const saved = await getBrand(slug);
      setName(saved.name); setB({ ...EMPTY, ...saved.brand, key_numbers: saved.brand.key_numbers || [] });
      setRevision(saved.review_revision); setDirty(false);
      if (ctx?.project && name.trim() && name.trim() !== ctx.project.name) ctx.onProject?.({ ...ctx.project, name: name.trim() });
      setOk(t("brand.saved"));
    } catch (e) {
      setErr((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  const missing = [!b.definition && t("brand.missingDefinition"), !b.industry && t("brand.missingCategory"), !b.target_users && t("brand.missingAudience")].filter(Boolean);

  return (
    <section className="space-y-6">
      <p className="page-description">{t("brand.description", { name: name || t("brand.theBrand") })}</p>
      {missing.length > 0 && <div className="alert alert-info mt-0">{t("brand.missing", { list: missing.join(", ") })}</div>}

      <fieldset disabled={!canEdit} className="contents">
      <Card title={t("brand.identity")}>
        <Field label={t("brand.name")} hint={t("brand.nameHint")}>
          <input className="input" value={name} onChange={(e) => { setOk(""); setDirty(true); setName(e.target.value); }} />
        </Field>
        <Field label={t("brand.aliases")} hint={t("brand.aliasesHint")}>
          <Lines value={b.aliases} onChange={(v) => set("aliases", v)} rows={3} placeholder={"acme\nAcme Inc"} />
        </Field>
        <Field label={t("brand.website")}>
          <input className="input bg-gray-50" value={site || t("common.noWebsite")} readOnly />
        </Field>
      </Card>

      <Card title={t("brand.whatItIs")}>
        <Field label={t("brand.definition")} hint={t("brand.definitionHint")}>
          <textarea className="input" rows={2} value={b.definition} onChange={(e) => set("definition", e.target.value)} placeholder={t("brand.definitionPlaceholder", { name: name || "Acme" })} />
        </Field>
        <div className="grid gap-4 md:grid-cols-2">
          <Field label={t("brand.category")} hint={t("brand.categoryHint")}>
            <input className="input" value={b.industry} onChange={(e) => set("industry", e.target.value)} />
          </Field>
          <Field label={t("brand.audience")} hint={t("brand.audienceHint")}>
            <input className="input" value={b.target_users} onChange={(e) => set("target_users", e.target.value)} />
          </Field>
        </div>
        <Field label={t("brand.products")} hint={t("brand.onePerLine")}>
          <Lines value={b.products} onChange={(v) => set("products", v)} rows={3} />
        </Field>
      </Card>

      <Card title={t("brand.numbers")} action={<button type="button" className="btn btn-secondary btn-sm" onClick={() => set("key_numbers", [...b.key_numbers, { fact: "", value: "", source: "" }])}><IconPlus size={14} /> {t("brand.addNumber")}</button>}>
        <p className="hint mb-3">{t("brand.numbersHint")}</p>
        {b.key_numbers.length === 0 && <p className="text-sm text-gray-400">{t("brand.noNumbers")}</p>}
        <div className="space-y-2">
          {b.key_numbers.map((n, i) => (
            <div key={i} className="grid grid-cols-[1fr_8rem_1fr_2.25rem] items-center gap-2">
              <input className="input" placeholder={t("brand.fact")} value={n.fact} onChange={(e) => set("key_numbers", upd(b.key_numbers, i, { fact: e.target.value }))} />
              <input className="input" placeholder={t("brand.value")} value={n.value} onChange={(e) => set("key_numbers", upd(b.key_numbers, i, { value: e.target.value }))} />
              <input className="input" placeholder={t("brand.source")} value={n.source} onChange={(e) => set("key_numbers", upd(b.key_numbers, i, { source: e.target.value }))} />
              <button type="button" className="btn btn-ghost btn-icon p-2" aria-label={t("brand.removeNumber")} onClick={() => set("key_numbers", b.key_numbers.filter((_, j) => j !== i))}><IconTrash size={16} /></button>
            </div>
          ))}
        </div>
      </Card>

      <Card title={t("brand.fit")}>
        <div className="grid gap-4 md:grid-cols-2">
          <Field label={t("brand.goodFit")} hint={t("brand.goodFitHint")}>
            <Lines value={b.suitable} onChange={(v) => set("suitable", v)} rows={4} />
          </Field>
          <Field label={t("brand.notFit")} hint={t("brand.notFitHint")}>
            <Lines value={b.unsuitable} onChange={(v) => set("unsuitable", v)} rows={4} />
          </Field>
        </div>
        <Field label={t("brand.confused")} hint={t("brand.confusedHint")}>
          <Lines value={b.disambiguation} onChange={(v) => set("disambiguation", v)} rows={3} />
        </Field>
      </Card>

      </fieldset>
      {err && <div className="alert alert-error">{err}</div>}
      {ok && <div className="alert alert-success">{ok}</div>}
      {canEdit && (
        <div className="flex gap-2">
          <button type="button" className="btn btn-primary" disabled={busy} onClick={save}>{busy ? t("common.saving") : t("brand.save")}</button>
          <HelpTip id="saveBrand" />
        </div>
      )}

      <ReviewConfirmation key={slug} slug={slug} kind="brand" revision={revision} dirty={dirty || busy} />
      <details className="card">
        <summary className="cursor-pointer px-5 py-4 text-sm font-medium text-gray-700">{t("brand.card")}</summary>
        <div className="border-t border-gray-100 p-5">
          <p className="hint mb-3">{t("brand.cardHint")}</p>
          <pre className="code-block max-h-96 whitespace-pre-wrap text-xs">{card || t("brand.cardEmpty")}</pre>
        </div>
      </details>
    </section>
  );
}

function upd(rows: KeyNumber[], i: number, patch: Partial<KeyNumber>) {
  return rows.map((r, j) => (j === i ? { ...r, ...patch } : r));
}

function Card({ title, action, children }: { title: string; action?: ReactNode; children: ReactNode }) {
  return (
    <div className="card">
      <div className="card-header"><h2 className="card-title">{title}</h2>{action}</div>
      <div className="card-body space-y-1">{children}</div>
    </div>
  );
}

function Field({ label, hint, children }: { label: string; hint?: string; children: ReactNode }) {
  return (
    <div className="field">
      <label>{label}</label>
      {children}
      {hint && <div className="input-hint">{hint}</div>}
    </div>
  );
}

// Lines edits a list as one item per line, keeping case and spaces.
function Lines({ value, onChange, rows, placeholder }: { value: string[]; onChange: (v: string[]) => void; rows: number; placeholder?: string }) {
  const [text, setText] = useState(value.join("\n"));
  useEffect(() => {
    if (text.split("\n").map((s) => s.trim()).filter(Boolean).join("\n") !== value.join("\n")) setText(value.join("\n"));
  }, [value]);
  return (
    <textarea className="input" rows={rows} value={text} placeholder={placeholder}
      onChange={(e) => { setText(e.target.value); onChange(e.target.value.split("\n").map((s) => s.trim()).filter(Boolean)); }} />
  );
}
