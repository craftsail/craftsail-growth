// SPDX-License-Identifier: AGPL-3.0-or-later

import { useState } from "react";
import { useSearchParams } from "react-router-dom";
import { IconCheck, IconExternalLink, IconWorldSearch } from "@tabler/icons-react";
import { verifyKey, type KeyItem } from "../../api";
import { useKeys } from "./useKeys";
import { PROVIDER_COLOR, PROVIDER_KEY_URL, PROVIDER_LABEL, PROVIDER_LOGO, PROVIDER_ORDER } from "./provider-meta";
import { useI18n } from "../../i18n";
import { HelpTip } from "../../components/HelpTip";

// Model providers: a list on the left (most used engines first) and the
// selected engine's details and connect form on the right.
export function ModelProviders() {
  const { t } = useI18n();
  const keys = useKeys();
  const [params, setParams] = useSearchParams();
  const providers = keys.items
    .filter((k) => k.code !== "google" && !k.manual)
    .sort((a, b) => rank(a.code) - rank(b.code));
  const connected = providers.filter((k) => k.ready).length;
  const selected = params.get("connect") || providers[0]?.code || "";
  const current = providers.find((k) => k.code === selected) || providers[0];
  const select = (code: string) => {
    keys.setErr(""); keys.setOk("");
    setParams({ connect: code }, { replace: true });
  };

  return (
    <section>
      <p className="page-description">{t("providers.description")}</p>
      <div className="card grid min-h-[560px] overflow-hidden lg:grid-cols-[260px_minmax(0,1fr)]">
        <aside className="border-b border-gray-100 bg-gray-50/60 p-3 lg:border-b-0 lg:border-r">
          <div className="flex items-center justify-between px-3 pb-2 pt-1">
            <span className="text-xs font-semibold uppercase tracking-wider text-gray-400">{t("providers.engines")}</span>
            <span className={"badge " + (connected > 0 ? "badge-success" : "badge-gray")}>{connected}/{providers.length}</span>
          </div>
          {providers.map((k) => (
            <button key={k.code} type="button" className={"subnav-link mb-0.5 " + (current?.code === k.code ? "subnav-link-active" : "")} onClick={() => select(k.code)}>
              <ProviderLogo k={k} size={28} />
              <span className="min-w-0 flex-1 truncate">{labelOf(k)}</span>
              {k.ready
                ? <span className="h-2 w-2 shrink-0 rounded-full bg-emerald-500" title={t("common.connected")} />
                : <span className="h-2 w-2 shrink-0 rounded-full bg-gray-300" title={t("common.notConnected")} />}
            </button>
          ))}
        </aside>
        <div className="min-w-0 p-6 sm:p-8">
          {current && (
            <>
              <div className="mb-5 flex flex-wrap items-center gap-3">
                <ProviderLogo k={current} size={44} />
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <h2 className="section-title">{labelOf(current)}</h2>
                    {current.ready
                      ? <span className="badge badge-success"><IconCheck size={12} stroke={2.2} /> {t("common.connected")}</span>
                      : <span className="badge badge-gray">{t("common.notConnected")}</span>}
                    {current.search && <span className="badge badge-primary"><IconWorldSearch size={12} /> {t("providers.webSearch")}</span>}
                  </div>
                  <p className="mt-1 text-sm text-gray-500">
                    {current.search ? t("providers.searches") : t("providers.memory")}
                  </p>
                </div>
              </div>
              <dl className="mb-6 grid grid-cols-1 gap-3 rounded-xl border border-gray-100 bg-gray-50 p-4 text-sm sm:grid-cols-3">
                <div><dt className="text-xs text-gray-500">{t("providers.model")}</dt><dd className="truncate font-mono text-gray-800">{current.model}{current.custom_model ? "" : t("providers.default")}</dd></div>
                <div><dt className="text-xs text-gray-500">{t("providers.key")}</dt><dd className="font-mono text-gray-800">{current.ready ? `····${current.key_tail || ""}` : "—"}</dd></div>
                <div><dt className="text-xs text-gray-500">{t("providers.endpoint")}</dt><dd className="truncate font-mono text-gray-800">{current.custom_base ? current.base : t("providers.official")}</dd></div>
              </dl>
              {keys.ok && <div className="alert alert-success mt-0">{keys.ok}</div>}
              <ProviderForm key={current.code} k={current} keys={keys} onDone={() => undefined} />
            </>
          )}
        </div>
      </div>
    </section>
  );
}

function rank(code: string) {
  const i = PROVIDER_ORDER.indexOf(code);
  return i < 0 ? 99 : i;
}

function labelOf(k: KeyItem) {
  return PROVIDER_LABEL[k.code] || k.name;
}

function ProviderLogo({ k, size = 40 }: { k: KeyItem; size?: number }) {
  const src = PROVIDER_LOGO[k.code];
  const box = { width: size, height: size };
  if (src) {
    return (
      <span className="flex shrink-0 items-center justify-center rounded-xl border border-gray-100 bg-white" style={box}>
        <img src={src} alt="" style={{ width: size * 0.62, height: size * 0.62 }} />
      </span>
    );
  }
  return (
    <span className="flex shrink-0 items-center justify-center rounded-xl text-sm font-bold text-white" style={{ ...box, background: PROVIDER_COLOR[k.code] || "#6b7280" }}>
      {k.name.slice(0, 1).toUpperCase()}
    </span>
  );
}

function ProviderForm({ k, keys, onDone }: { k: KeyItem; keys: ReturnType<typeof useKeys>; onDone: () => void }) {
  const { t } = useI18n();
  const env = k.key_env || "";
  const baseEnv = k.base_env || "";
  const modelEnv = k.model_env || "";
  const official = k.official_base || "";
  const [key, setKey] = useState("");
  const [model, setModel] = useState(k.custom_model ? k.model || "" : "");
  const [base, setBase] = useState(k.custom_base ? k.base || "" : "");
  const [advanced, setAdvanced] = useState(!!(k.custom_base || k.custom_model));
  const [testing, setTesting] = useState(false);
  const [test, setTest] = useState<{ ok: boolean; msg: string } | null>(null);

  async function runTest() {
    setTesting(true); setTest(null);
    try {
      await verifyKey({ code: k.code, key: key.trim() || undefined, base: base.trim() || undefined });
      setTest({ ok: true, msg: t("providers.keyWorks") });
    } catch (e) {
      setTest({ ok: false, msg: (e as Error).message });
    } finally {
      setTesting(false);
    }
  }

  async function save() {
    const u: Record<string, string> = {};
    if (key.trim()) u[env] = key.trim();
    if (baseEnv) u[baseEnv] = base.trim() === official ? "" : base.trim();
    if (modelEnv) u[modelEnv] = model.trim();
    if (!k.ready && !key.trim()) {
      keys.setErr(t("providers.pasteFirst"));
      return;
    }
    await keys.save(u, t("providers.saved", { name: labelOf(k) }));
    onDone();
  }

  async function disconnect() {
    if (!window.confirm(t("providers.removeConfirm", { name: labelOf(k) }))) return;
    await keys.save({ [env]: "" }, t("providers.disconnected", { name: labelOf(k) }));
    onDone();
  }

  return (
    <div>
      <ol className="mb-5 space-y-1.5 text-sm text-gray-600">
        <li>{t("providers.step1", { name: labelOf(k) })}</li>
        <li>{t("providers.step2")}</li>
        <li>{t("providers.step3", { name: labelOf(k) })}</li>
      </ol>
      {PROVIDER_KEY_URL[k.code] && (
        <a className="btn btn-secondary btn-sm mb-5" href={PROVIDER_KEY_URL[k.code]} target="_blank" rel="noreferrer">
          {t("providers.openConsole", { name: labelOf(k) })} <IconExternalLink size={14} />
        </a>
      )}
      <div className="field">
        <label htmlFor="provider-key">{t("providers.apiKey")}</label>
        <input id="provider-key" className="input font-mono" type="password" autoComplete="off" value={key} onChange={(e) => setKey(e.target.value)}
          placeholder={k.ready ? t("providers.savedKey", { tail: k.key_tail || "" }) : t("providers.pasteKey")} />
        <div className="input-hint">{t("providers.envVar")} <code>{env}</code></div>
      </div>
      <div className="mb-4 flex items-center gap-3">
        <button type="button" className="btn btn-secondary btn-sm" disabled={testing} onClick={runTest}>{testing ? t("common.testing") : t("providers.testConnection")}</button>
        <HelpTip id="testConnection" />
        {test && <span className={"text-sm " + (test.ok ? "text-emerald-700" : "text-red-700")}>{test.msg}</span>}
      </div>

      <button type="button" className="mb-3 text-sm font-medium text-primary-600 hover:text-primary-700" onClick={() => setAdvanced((v) => !v)} aria-expanded={advanced}>
        {advanced ? t("providers.hideAdvanced") : t("providers.advanced")}
      </button>
      <HelpTip id="advanced" />
      {advanced && (
        <div className="mb-4 rounded-xl border border-gray-100 bg-gray-50 p-4">
          {modelEnv && (
            <div className="field">
              <label htmlFor="provider-model">{t("providers.model")}</label>
              <input id="provider-model" className="input font-mono" value={model} onChange={(e) => setModel(e.target.value)} placeholder={k.model} />
              <div className="input-hint">{t("providers.modelHint")}</div>
            </div>
          )}
          {baseEnv && (
            <div className="field mb-0">
              <label htmlFor="provider-base">{t("providers.apiEndpoint")}</label>
              <input id="provider-base" className="input font-mono" value={base} onChange={(e) => setBase(e.target.value)} placeholder={official} />
              <div className="input-hint">{t("providers.endpointHint")}</div>
            </div>
          )}
        </div>
      )}

      {keys.err && <div className="alert alert-error">{keys.err}</div>}
      <div className="flex flex-wrap items-center gap-2 border-t border-gray-100 pt-4">
        <button type="button" className="btn btn-primary" disabled={keys.busy} onClick={save}>{k.ready ? t("providers.saveChanges") : t("providers.saveConnect")}</button>
        <HelpTip id="saveConnect" />
        {k.ready && <button type="button" className="btn btn-ghost text-red-600 hover:bg-red-50 hover:text-red-700" disabled={keys.busy} onClick={disconnect}>{t("providers.disconnect")}</button>}
      </div>
    </div>
  );
}
