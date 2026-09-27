// SPDX-License-Identifier: AGPL-3.0-or-later

import { Logo } from "../../components/brand/Logo";
import { FormEvent, useState } from "react";
import { login, setupAccount } from "../../api";
import { useI18n } from "../../i18n";
import { LanguagePicker } from "../../components/LanguagePicker";

export function Login({ onOk, setup }: { onOk: () => void; setup?: boolean }) {
  const { t } = useI18n();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [password2, setPassword2] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr("");
    try {
      if (setup) {
        if (password !== password2) {
          throw new Error(t("auth.mismatch"));
        }
        await setupAccount(username, password);
      } else {
        await login(username, password);
      }
      onOk();
    } catch (ex) {
      setErr((ex as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mesh-gradient flex min-h-screen items-start justify-center bg-gray-50">
      <div className="card mt-[8vh] w-[min(460px,calc(100%-32px))] p-8 shadow-glass">
        <Logo />
        <h1 className="mt-4 page-title">{setup ? t("auth.setupTitle") : t("auth.loginTitle")}</h1>
        {setup && <p className="hint">{t("auth.setupHint")}</p>}
        <form onSubmit={onSubmit}>
          <div className="field">
            <label>{t("auth.account")}</label>
            <input className="input" autoComplete="username" value={username} onChange={(e) => setUsername(e.target.value)} />
          </div>
          <div className="field">
            <label>{t("auth.password")}</label>
            <input className="input" type="password" autoComplete={setup ? "new-password" : "current-password"} value={password} onChange={(e) => setPassword(e.target.value)} />
          </div>
          {setup && (
            <div className="field">
              <label>{t("auth.confirmPassword")}</label>
              <input className="input" type="password" autoComplete="new-password" value={password2} onChange={(e) => setPassword2(e.target.value)} />
            </div>
          )}
          {err && <div className="alert alert-error">{err}</div>}
          <button className="btn btn-primary" type="submit" disabled={busy}>{busy ? t("auth.submitting") : setup ? t("auth.saveContinue") : t("auth.login")}</button>
        </form>
        <LanguagePicker />
      </div>
    </div>
  );
}
