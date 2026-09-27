// SPDX-License-Identifier: AGPL-3.0-or-later

import { useState, type FormEvent } from "react";
import { changeOwnPassword } from "../../api";
import { useI18n } from "../../i18n";
import { HelpTip } from "../../components/HelpTip";

// PasswordPage changes the signed-in user's own password. The server ends
// their other sessions and signs this browser in again.
export function PasswordPage() {
  const { t } = useI18n();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [again, setAgain] = useState("");
  const [err, setErr] = useState("");
  const [ok, setOk] = useState(false);
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setErr(""); setOk(false);
    if (next !== again) { setErr(t("access.mismatch")); return; }
    setBusy(true);
    try {
      await changeOwnPassword(current, next);
      setOk(true); setCurrent(""); setNext(""); setAgain("");
    } catch (e2) {
      setErr((e2 as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <section>
      <p className="page-description">{t("access.passwordHint")}</p>
      <form className="card max-w-md space-y-4 p-6" onSubmit={submit}>
        {err && <div className="alert alert-error">{err}</div>}
        {ok && <div className="alert alert-success">{t("access.passwordChanged")}</div>}
        <label className="block text-sm"><span className="mb-1 block font-medium text-gray-700">{t("access.current")}</span>
          <input className="input" type="password" value={current} onChange={(e) => setCurrent(e.target.value)} autoComplete="current-password" /></label>
        <label className="block text-sm"><span className="mb-1 block font-medium text-gray-700">{t("access.newPassword")}</span>
          <input className="input" type="password" value={next} onChange={(e) => setNext(e.target.value)} autoComplete="new-password" /></label>
        <label className="block text-sm"><span className="mb-1 block font-medium text-gray-700">{t("access.repeat")}</span>
          <input className="input" type="password" value={again} onChange={(e) => setAgain(e.target.value)} autoComplete="new-password" /></label>
        <button type="submit" className="btn btn-primary" disabled={busy || !current || next.length < 12}>{t("access.changePassword")}</button>
        <HelpTip id="changePassword" />
      </form>
    </section>
  );
}
