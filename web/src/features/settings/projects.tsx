// SPDX-License-Identifier: AGPL-3.0-or-later

import { FormEvent, useEffect, useState } from "react";
import { useNavigate, useOutletContext } from "react-router-dom";
import { createProject, getProject, patchProject, startJob, type Project } from "../../api";
import { useI18n } from "../../i18n";
import { HelpTip } from "../../components/HelpTip";

type Ctx = { project?: Project; projects?: Project[]; onProject?: (p: Project) => void; onCreated?: (p: Project) => void } | undefined;

// CreateProjectForm is shared by first-run onboarding and the Projects page.
// The first technical check runs without AI or Google credentials.
export function CreateProjectForm({
  onCreated,
  submitLabel,
}: {
  onCreated: (p: Project) => void;
  submitLabel?: string;
}) {
  const { t } = useI18n();
  const nav = useNavigate();
  const [url, setUrl] = useState("");
  const [name, setName] = useState("");
  const [noSite, setNoSite] = useState(false);
  const [runNow, setRunNow] = useState(true);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr("");
    try {
      const p = await createProject({ url: noSite ? "" : url.trim(), name: name.trim(), no_site: noSite });
      let startError = "";
      if (runNow && !noSite) {
        try { await startJob(p.slug, "first-check"); }
        catch (ex) { startError = (ex as Error).message; }
      }
      // Creation already succeeded. A failed start must never invite another
      // create request; recovery happens against this same project.
      onCreated(p);
      nav(`/p/${p.slug}/overview`, { state: { firstCheckError: startError } });
    } catch (ex) {
      setErr((ex as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={submit}>
      <label className="checkbox-row">
        <input type="checkbox" checked={noSite} onChange={(e) => setNoSite(e.target.checked)} />
        {t("projects.noSite")}
      </label>
      {!noSite && (
        <div className="field">
          <label>{t("projects.url")}</label>
          <input className="input" value={url} onChange={(e) => setUrl(e.target.value)} placeholder="https://example.com" required={!noSite} />
        </div>
      )}
      <div className="field">
        <label>{t("projects.brandName")}</label>
        <input className="input" value={name} onChange={(e) => setName(e.target.value)} placeholder={noSite ? t("projects.nameRequired") : t("projects.nameOptional")} required={noSite} />
      </div>
      {!noSite && <label className="checkbox-row">
        <input type="checkbox" checked={runNow} onChange={(e) => setRunNow(e.target.checked)} />
        {t("projects.runNow")}
      </label>}
      {err && <div className="alert alert-error">{err}</div>}
      <button className="btn btn-primary" disabled={busy} type="submit">{busy ? t("projects.creating") : submitLabel || t("projects.create")}</button>
      <HelpTip id="createProject" />
    </form>
  );
}

export function Projects() {
  const { t } = useI18n();
  const ctx = useOutletContext<Ctx>();
  const nav = useNavigate();
  const [adding, setAdding] = useState(false);
  const [ok, setOk] = useState("");
  const [err, setErr] = useState("");
  const current = ctx?.project;
  const projects = ctx?.projects || [];

  return (
    <section className="space-y-6">
      <p className="page-description">{t("projects.description")}</p>

      <div className="card">
        <div className="card-header">
          <h2 className="card-title">{t("projects.all")}</h2>
          {!adding && <button type="button" className="btn btn-primary" onClick={() => setAdding(true)}>{t("projects.new")}</button>}
        </div>
        <div className="divide-y divide-gray-100">
          {projects.map((p) => (
            <div key={p.slug} className="flex items-center justify-between gap-3 px-5 py-4">
              <div className="min-w-0">
                <div className="truncate font-medium text-gray-900">{p.name}</div>
                <div className="truncate text-sm text-gray-500">{p.no_site ? t("common.noWebsite") : (p.site || p.slug)}</div>
              </div>
              {p.slug === current?.slug
                ? <span className="badge badge-primary">{t("common.current")}</span>
                : <button type="button" className="btn btn-secondary btn-sm" onClick={() => nav(`/p/${p.slug}/settings/projects`)}>{t("common.open")}</button>}
            </div>
          ))}
        </div>
        {adding && (
          <div className="card-footer">
            <div className="mb-4 flex items-center justify-between">
              <h3 className="card-title">{t("projects.new")}</h3>
              <button type="button" className="btn btn-ghost btn-sm" onClick={() => setAdding(false)}>{t("common.cancel")}</button>
            </div>
            <CreateProjectForm onCreated={(p) => { ctx?.onCreated?.(p); setAdding(false); }} />
          </div>
        )}
      </div>

      {current && (
        <div className="card card-body">
          {ok && <div className="alert alert-success mt-0">{ok}</div>}
          {err && <div className="alert alert-error mt-0">{err}</div>}
          <ProjectForm
            key={current.slug}
            slug={current.slug}
            seed={current}
            onSaved={(p) => { ctx?.onProject?.(p); setOk(t("projects.saved")); }}
            onError={setErr}
          />
        </div>
      )}
    </section>
  );
}

function ProjectForm({
  slug, seed, onSaved, onError,
}: {
  slug: string;
  seed?: Project;
  onSaved: (p: Project) => void;
  onError: (msg: string) => void;
}) {
  const { t } = useI18n();
  const [p, setP] = useState<Project | null>(seed || null);
  const [url, setUrl] = useState(seed?.site || "");
  const [name, setName] = useState(seed?.name || "");
  const [noSite, setNoSite] = useState(!!seed?.no_site);
  const [sampling,setSampling]=useState({sampling_language:seed?.sampling_language||"",site_language:seed?.site_language||"",report_language:seed?.report_language||"en",target_region:seed?.target_region||""});
  const [gscSite, setGscSite] = useState(seed?.gsc_site || "");
  const [ga4Property, setGa4Property] = useState(seed?.ga4_property || "");
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    getProject(slug).then((got) => {
      setP(got);
 setSampling({sampling_language:got.sampling_language||"",site_language:got.site_language||"",report_language:got.report_language||"en",target_region:got.target_region||""});
      setUrl(got.site || "");
      setName(got.name || "");
      setNoSite(!!got.no_site);
      setGscSite(got.gsc_site || "");
      setGa4Property(got.ga4_property || "");
    }).catch((e: Error) => onError(e.message));
  }, [slug]);
  async function save() {
    setBusy(true);
    onError("");
    try {
      const got = await patchProject(slug, {
        ...sampling, name, no_site: noSite, url: noSite ? "" : url,
        gsc_site: gscSite.trim(),
        ga4_property: ga4Property.trim(),
      });
      setP(got);
      onSaved(got);
    } catch (e) {
      onError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <div>
      <h2 className="section-title">{p?.name || slug}</h2>
      <p className="hint">{t("projects.recrawl")}</p>
      <div className="field">
        <label>{t("projects.brandName")}</label>
        <input className="input" value={name} onChange={(e) => setName(e.target.value)} />
      </div>
      <label className="checkbox-row">
        <input type="checkbox" checked={noSite} onChange={(e) => setNoSite(e.target.checked)} />
        {t("projects.noSite")}
      </label>
      {!noSite && (
        <div className="field">
          <label>{t("projects.url")}</label>
          <input className="input" value={url} onChange={(e) => setUrl(e.target.value)} placeholder="https://" />
        </div>
      )}
      <fieldset className="my-5 space-y-3 rounded-lg border border-gray-200 p-4"><legend className="px-2 font-medium">{t("sampling.title")}</legend><p className="text-sm text-gray-500">{t("sampling.note")}</p>
        {([['sampling_language','sampling.language'],['site_language','sampling.site'],['report_language','weeklyReport.language']] as const).map(([field,label])=><label className="flex flex-wrap items-center gap-3" key={field}>{t(label)}<select className="input" name={field} value={sampling[field]} onChange={e=>setSampling({...sampling,[field]:e.target.value})}>{field!=="report_language"&&<option value="">{t("sampling.unknown")}</option>}{([['en','english'],['zh','chinese'],['pt','portuguese']] as const).map(([v,k])=><option key={v} value={v}>{t(`weeklyReport.${k}`)}</option>)}</select></label>)}
        <label className="block">{t("sampling.region")}<input name="target_region" className="input mt-1 w-full" maxLength={64} value={sampling.target_region} onChange={e=>setSampling({...sampling,target_region:e.target.value})}/></label>
      </fieldset>
      <button className="btn btn-primary" disabled={busy} onClick={save}>{busy ? t("common.saving") : t("projects.save")}</button>
    </div>
  );
}
