// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useState } from "react";
import { useParams, useSearchParams } from "react-router-dom";
import { IconBrandGoogle, IconChartBar, IconCheck, IconCopy, IconExternalLink, IconSearch } from "@tabler/icons-react";
import {
  disconnectGoogle, getGoogleStatus, getWebstats, runWebstats, saveGoogleChoice, verifyKey,
  type GoogleStatus, type SourceView,
} from "../../api";
import { useKeys } from "./useKeys";
import { useI18n, type Key } from "../../i18n";
import { HelpTip } from "../../components/HelpTip";


// Google Search Console and GA4. Step 1 connects a Google account; step 2
// picks the property for each source and shows its import status.
export function GoogleData() {
  const { slug = "" } = useParams();
  const { t } = useI18n();
  const [params] = useSearchParams();
  const keys = useKeys();
  const [st, setSt] = useState<GoogleStatus | null>(null);
  const [sources, setSources] = useState<SourceView[]>([]);
  const [picking, setPicking] = useState(params.get("pick") === "1");
  const [gsc, setGsc] = useState("");
  const [ga, setGa] = useState("");
  const [err, setErr] = useState(params.get("google") === "need-client" ? t("google.needClient") : "");
  const [ok, setOk] = useState("");
  const [busy, setBusy] = useState(false);

  function load(pick: boolean) {
    if (!slug) return;
    getGoogleStatus(slug, pick).then((d) => {
      setSt(d);
      const sugG = (d.gsc_choices || []).find((c) => c.suggested && c.selectable);
      const sugA = (d.ga_choices || []).find((c) => c.suggested);
      setGsc(d.gsc_site || sugG?.site_url || "");
      setGa(d.ga4_property || sugA?.id || "");
      if (d.list_error) setErr(d.list_error);
    }).catch((e: Error) => setErr(e.message));
    getWebstats(slug).then((s) => setSources(s.sources || [])).catch(() => setSources([]));
  }
  useEffect(() => { load(picking); }, [slug, picking]);

  async function saveProperties() {
    setBusy(true); setErr(""); setOk("");
    try {
      await saveGoogleChoice(slug, { gsc_site: gsc, ga4_property: ga });
      setOk(t("google.propertiesSaved"));
      setPicking(false);
      load(false);
    } catch (e) {
      setErr((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  async function sync() {
    setBusy(true); setErr(""); setOk("");
    try {
      const r = await runWebstats(slug);
      setOk(r.from && r.to ? t("google.syncedRange", { from: r.from, to: r.to }) : t("google.synced"));
      load(false);
    } catch (e) {
      setErr((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  const serviceAccount = keys.items.find((k) => k.code === "google");
  const saReady = !!serviceAccount?.ready;
  const connected = !!st?.connected || saReady;
  const gscSource = sources.find((s) => s.source === "gsc");
  const gaSource = sources.find((s) => s.source === "ga4");
  const connectURL = `/api/google/connect?slug=${encodeURIComponent(slug)}`;

  return (
    <section className="space-y-6">
      <p className="page-description">{t("google.description")}</p>
      {err && <div className="alert alert-error">{err}</div>}
      {ok && <div className="alert alert-success">{ok}</div>}

      <div className="card">
        <div className="card-header">
          <div className="flex items-center gap-3">
            <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-gray-100 text-gray-700"><IconBrandGoogle size={20} /></span>
            <div>
              <h2 className="card-title">{t("google.step1")}</h2>
              <p className="text-sm text-gray-500">
                {st?.connected ? t("google.connectedAs", { email: st.email || t("google.aGoogleAccount") }) : saReady ? t("google.usingSA") : t("common.notConnected")}
              </p>
            </div>
          </div>
          {connected && !st?.needs_reconnect && <span className="badge badge-success"><IconCheck size={12} stroke={2.2} /> {t("common.connected")}</span>}
          {st?.needs_reconnect && <span className="badge badge-danger">{t("google.needsSignIn")}</span>}
        </div>
        <div className="card-body">
          {st && !st.oauth_configured && !saReady && (
            <OAuthSetup redirect={st.redirect_uri} keys={keys} onSaved={() => load(false)} />
          )}
          {st?.oauth_configured && (!st.connected || st.needs_reconnect) && (
            <div>
              {st.needs_reconnect && <div className="alert alert-error mt-0">{t("google.reauthNote")}</div>}
              <p className="mb-3 text-sm text-gray-600">{t("google.asks")}</p>
              <ul className="mb-5 space-y-1.5 text-sm text-gray-700">
                <li className="flex items-center gap-2"><IconCheck size={16} className="text-primary-600" /> {t("google.scopeGsc")}</li>
                <li className="flex items-center gap-2"><IconCheck size={16} className="text-primary-600" /> {t("google.scopeGa")}</li>
                <li className="flex items-center gap-2"><IconCheck size={16} className="text-primary-600" /> {t("google.scopeEmail")}</li>
              </ul>
              <a className="btn btn-primary" href={connectURL}><IconBrandGoogle size={16} /> {st.needs_reconnect ? t("google.reconnect") : t("google.connect")}</a>
              <HelpTip id="googleConnect" />
            </div>
          )}
          {st?.connected && !st.needs_reconnect && (
            <div className="flex flex-wrap gap-2">
              <a className="btn btn-secondary btn-sm" href={connectURL}>{t("google.switch")}</a>
              <button type="button" className="btn btn-ghost btn-sm text-red-600 hover:bg-red-50" onClick={() => {
                if (window.confirm(t("google.disconnectConfirm"))) disconnectGoogle().then(() => load(false));
              }}>{t("google.disconnect")}</button>
              <HelpTip id="googleDisconnect" />
            </div>
          )}
          {!st?.connected && saReady && <p className="text-sm text-gray-600">{t("google.saNote")}</p>}
        </div>
      </div>

      <div>
        <div className="mb-3 flex items-center justify-between gap-3">
          <h2 className="section-title">{t("google.step2")}</h2>
          {connected && (
            <div className="flex gap-2">
              {!picking && <button type="button" className="btn btn-secondary btn-sm" onClick={() => setPicking(true)}>{t("google.change")}</button>}
              <button type="button" className="btn btn-primary btn-sm" disabled={busy} onClick={sync}>{busy ? t("google.working") : t("google.syncNow")}</button>
              <HelpTip id="googleSyncNow" />
            </div>
          )}
        </div>
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <SourceCard
            icon={<IconSearch size={20} />}
            title="Search Console"
            what={t("google.gscWhat")}
            property={st?.gsc_site}
            source={gscSource}
            disabled={!connected}
          >
            {picking && connected && (
              <select className="input" value={gsc} onChange={(e) => setGsc(e.target.value)} aria-label={t("google.gscProperty")}>
                <option value="">{t("google.choose")}</option>
                {(st?.gsc_choices || []).map((c) => (
                  <option key={c.site_url} value={c.site_url} disabled={!c.selectable}>{c.site_url}{c.selectable ? "" : t("google.notVerified")}</option>
                ))}
              </select>
            )}
          </SourceCard>
          <SourceCard
            icon={<IconChartBar size={20} />}
            title="Google Analytics 4"
            what={t("google.gaWhat")}
            property={st?.ga4_property}
            source={gaSource}
            disabled={!connected}
          >
            {picking && connected && (
              <select className="input" value={ga} onChange={(e) => setGa(e.target.value)} aria-label={t("google.gaProperty")}>
                <option value="">{t("google.choose")}</option>
                {(st?.ga_choices || []).map((c) => (
                  <option key={c.id} value={c.id}>{c.name} · {c.id}</option>
                ))}
              </select>
            )}
          </SourceCard>
        </div>
        {picking && connected && (
          <div className="mt-4 flex gap-2">
            <button type="button" className="btn btn-primary" disabled={busy} onClick={saveProperties}>{t("google.saveProperties")}</button>
            <HelpTip id="saveProperties" />
            <button type="button" className="btn btn-secondary" onClick={() => setPicking(false)}>{t("common.cancel")}</button>
          </div>
        )}
      </div>

      <details className="card">
        <summary className="cursor-pointer px-5 py-4 text-sm font-medium text-gray-700">{t("google.advanced")}</summary>
        <div className="border-t border-gray-100 p-5">
          <Advanced st={st} keys={keys} onSaved={() => load(false)} />
        </div>
      </details>
    </section>
  );
}

const STATE_BADGE: Record<string, string> = {
  ready: "badge-success", waiting: "badge-primary", choose_property: "badge-warning", not_connected: "badge-gray",
  needs_reauth: "badge-danger", rate_limited: "badge-warning", config_invalid: "badge-danger", failed: "badge-danger",
};

function SourceCard({ icon, title, what, property, source, disabled, children }: {
  icon: React.ReactNode; title: string; what: string; property?: string; source?: SourceView; disabled: boolean; children?: React.ReactNode;
}) {
  const { t } = useI18n();
  const state = disabled ? "not_connected" : source?.state || (property ? "waiting" : "choose_property");
  const cls = STATE_BADGE[state] || "badge-gray";
  const label = STATE_BADGE[state] ? t(`search.states.${state}` as Key) : source?.label || state;
  return (
    <div className={"card flex flex-col p-5 " + (disabled ? "opacity-70" : "")}>
      <div className="flex items-center gap-3">
        <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary-50 text-primary-600">{icon}</span>
        <h3 className="card-title min-w-0 flex-1">{title}</h3>
        <span className={"badge " + cls}>{label}</span>
      </div>
      <p className="mt-3 text-sm text-gray-500">{what}</p>
      <dl className="mt-3 space-y-1 text-sm">
        <div className="flex gap-2"><dt className="w-36 shrink-0 text-gray-500">{t("google.property")}</dt><dd className="min-w-0 truncate font-mono text-gray-800">{property || "—"}</dd></div>
        {source?.through && <div className="flex gap-2"><dt className="w-36 shrink-0 text-gray-500">{t("google.importedThrough")}</dt><dd className="text-gray-800">{source.through}</dd></div>}
      </dl>
      {source?.detail && state !== "ready" && <p className="mt-2 text-xs text-gray-500">{source.detail}</p>}
      {children && <div className="mt-4">{children}</div>}
    </div>
  );
}

function CopyField({ value }: { value: string }) {
  const { t } = useI18n();
  const [copied, setCopied] = useState(false);
  return (
    <div className="flex items-center gap-2">
      <code className="min-w-0 flex-1 truncate rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 text-[13px] text-gray-800">{value}</code>
      <button type="button" className="btn btn-secondary btn-sm" onClick={() => {
        navigator.clipboard?.writeText(value).then(() => { setCopied(true); setTimeout(() => setCopied(false), 1500); });
      }}>{copied ? <IconCheck size={14} /> : <IconCopy size={14} />} {copied ? t("google.copied") : t("google.copy")}</button>
    </div>
  );
}

function OAuthSetup({ redirect, keys, onSaved }: { redirect: string; keys: ReturnType<typeof useKeys>; onSaved: () => void }) {
  const { t } = useI18n();
  const [id, setId] = useState("");
  const [secret, setSecret] = useState("");
  return (
    <div>
      <p className="mb-4 text-sm text-gray-600">{t("google.setupIntro")}</p>
      <ol className="mb-5 space-y-3 text-sm text-gray-700">
        <li><span className="font-medium">1.</span> {t("google.setup1")}
          <a className="ml-1 inline-flex items-center gap-1" href="https://console.cloud.google.com/apis/library" target="_blank" rel="noreferrer">{t("google.openLibrary")} <IconExternalLink size={13} /></a></li>
        <li><span className="font-medium">2.</span> {t("google.setup2")}
          <div className="mt-2"><CopyField value={redirect} /></div>
          <a className="mt-1 inline-flex items-center gap-1" href="https://console.cloud.google.com/apis/credentials" target="_blank" rel="noreferrer">{t("google.openCredentials")} <IconExternalLink size={13} /></a></li>
        <li><span className="font-medium">3.</span> {t("google.setup3")}</li>
      </ol>
      <div className="field">
        <label htmlFor="g-client-id">{t("google.clientId")}</label>
        <input id="g-client-id" className="input font-mono" value={id} onChange={(e) => setId(e.target.value)} placeholder="xxxx.apps.googleusercontent.com" />
      </div>
      <div className="field">
        <label htmlFor="g-client-secret">{t("google.clientSecret")}</label>
        <input id="g-client-secret" className="input font-mono" type="password" autoComplete="off" value={secret} onChange={(e) => setSecret(e.target.value)} />
      </div>
      {keys.err && <div className="alert alert-error">{keys.err}</div>}
      <div className="flex flex-wrap gap-2">
        <button type="button" className="btn btn-primary" disabled={keys.busy || !id.trim() || !secret.trim()} onClick={async () => {
          await keys.save({ GOOGLE_OAUTH_CLIENT_ID: id.trim(), GOOGLE_OAUTH_CLIENT_SECRET: secret.trim() }, t("google.clientSaved"));
          onSaved();
        }}>{t("google.saveClient")}</button>
      </div>
      <p className="mt-4 text-xs text-gray-500">{t("google.testingNote")}</p>
    </div>
  );
}

function Advanced({ st, keys, onSaved }: { st: GoogleStatus | null; keys: ReturnType<typeof useKeys>; onSaved: () => void }) {
  const { t } = useI18n();
  const sa = keys.items.find((k) => k.code === "google");
  const [json, setJson] = useState("");
  const [proxy, setProxy] = useState("");
  const [test, setTest] = useState<{ ok: boolean; msg: string } | null>(null);
  return (
    <div className="space-y-6">
      {st?.oauth_configured && (
        <div>
          <h3 className="card-title mb-1">{t("google.clientTitle")}</h3>
          <p className="mb-3 text-sm text-gray-500">{t("google.clientSavedNote")}</p>
          <CopyField value={st.redirect_uri} />
        </div>
      )}
      <div>
        <h3 className="card-title mb-1">{t("google.saTitle")}</h3>
        <p className="mb-3 text-sm text-gray-500">{t("google.saHint")}</p>
        <textarea className="input min-h-36 font-mono text-xs" spellCheck={false} autoComplete="off" value={json} onChange={(e) => setJson(e.target.value)}
          placeholder={sa?.ready ? t("google.saSaved", { tail: sa.key_tail || "" }) : '{"type":"service_account","client_email":"…","private_key":"…"}'} />
        <div className="mt-2 flex flex-wrap items-center gap-2">
          <button type="button" className="btn btn-secondary btn-sm" onClick={async () => {
            setTest(null);
            try { await verifyKey({ code: "google", key: json.trim() || undefined }); setTest({ ok: true, msg: t("google.saWorks") }); }
            catch (e) { setTest({ ok: false, msg: (e as Error).message }); }
          }}>{t("common.test")}</button>
          <button type="button" className="btn btn-primary btn-sm" disabled={keys.busy || !json.trim()} onClick={async () => { await keys.save({ GOOGLE_SA_JSON: json.trim() }, t("google.saSavedMsg")); setJson(""); onSaved(); }}>{t("common.save")}</button>
          {test && <span className={"text-sm " + (test.ok ? "text-emerald-700" : "text-red-700")}>{test.msg}</span>}
        </div>
      </div>
      <div>
        <h3 className="card-title mb-1">{t("google.proxyTitle")}</h3>
        <p className="mb-3 text-sm text-gray-500">{t("google.proxyHint")}{sa?.proxy_set ? t("google.proxySet") : ""}</p>
        <div className="flex gap-2">
          <input className="input font-mono" value={proxy} onChange={(e) => setProxy(e.target.value)} placeholder="http://127.0.0.1:7890" />
          <button type="button" className="btn btn-secondary" disabled={keys.busy} onClick={() => keys.save({ GOOGLE_HTTP_PROXY: proxy.trim() }, proxy.trim() ? t("google.proxySaved") : t("google.proxyRemoved"))}>{t("common.save")}</button>
        </div>
      </div>
      {keys.ok && <div className="alert alert-success">{keys.ok}</div>}
    </div>
  );
}
