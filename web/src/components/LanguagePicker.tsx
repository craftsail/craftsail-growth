// SPDX-License-Identifier: AGPL-3.0-or-later

import { IconLanguage } from "@tabler/icons-react";
import { LOCALES, useI18n, type Locale } from "../i18n";

// LanguagePicker is the language choice on pages outside the app shell
// (sign-in, first run). Inside the shell it lives in the user menu.
export function LanguagePicker() {
  const { t, locale, setLocale } = useI18n();
  return (
    <label className="mt-6 flex items-center justify-end gap-2 border-t border-gray-100 pt-4 text-xs text-gray-500">
      <IconLanguage size={14} aria-hidden="true" />
      <span className="sr-only">{t("lang.label")}</span>
      <select className="rounded-lg border border-gray-200 bg-white px-2 py-1 text-xs text-gray-700" value={locale} onChange={(e) => setLocale(e.target.value as Locale)} aria-label={t("lang.label")}>
        {LOCALES.map((l) => <option key={l.id} value={l.id}>{l.label}</option>)}
      </select>
    </label>
  );
}
