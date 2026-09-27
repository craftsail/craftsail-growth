// SPDX-License-Identifier: AGPL-3.0-or-later

import { FormEvent, useState } from "react";
import { changeOwnPassword } from "../../api";
import { Logo } from "../../components/brand/Logo";
import { LanguagePicker } from "../../components/LanguagePicker";
import { useI18n } from "../../i18n";

// ChangeDefaultPassword blocks the app for an account still on the public
// default password; the API refuses everything else until it is replaced.
export function ChangeDefaultPassword({ onOk, onLogout }: { onOk: () => void; onLogout: () => void }) {
  const { t } = useI18n();
  const [password, setPassword] = useState("");
  const [password2, setPassword2] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setErr("");
    if (password !== password2) {
      setErr(t("auth.mismatch"));
      return;
    }
    setBusy(true);
    try {
      await changeOwnPassword("", password);
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
        <h1 className="mt-4 page-title">{t("auth.changeDefaultTitle")}</h1>
        <p className="hint">{t("auth.changeDefaultHint")}</p>
        <form onSubmit={onSubmit}>
          <div className="field">
            <label>{t("auth.newPassword")}</label>
            <input className="input" type="password" autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)} />
          </div>
          <div className="field">
            <label>{t("auth.confirmPassword")}</label>
            <input className="input" type="password" autoComplete="new-password" value={password2} onChange={(e) => setPassword2(e.target.value)} />
          </div>
          {err && <div className="alert alert-error">{err}</div>}
          <div className="flex items-center gap-3">
            <button className="btn btn-primary" type="submit" disabled={busy || password.length < 12}>{busy ? t("auth.submitting") : t("auth.saveContinue")}</button>
            <button className="btn btn-secondary" type="button" onClick={onLogout}>{t("shell.logout")}</button>
          </div>
        </form>
        <LanguagePicker />
      </div>
    </div>
  );
}
