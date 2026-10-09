// SPDX-License-Identifier: AGPL-3.0-or-later

import { Logo } from "../../components/brand/Logo";
import { Link } from "react-router-dom";
import type { Project } from "../../api";
import { CreateProjectForm } from "../settings/projects";
import { useI18n } from "../../i18n";
import { LanguagePicker } from "../../components/LanguagePicker";

// Onboard is shown after sign-in whenever no project exists yet, including
// right after the account is created on a fresh install.
export function Onboard({ onCreated }: { onCreated: (p: Project) => void }) {
  const { t } = useI18n();
  return (
    <div className="mesh-gradient flex min-h-screen items-start justify-center bg-gray-50 px-4">
      <div className="card mt-[8vh] w-full max-w-lg p-8 shadow-glass">
        <Logo />
        <h1 className="mt-6 text-2xl font-bold tracking-tight text-gray-900">{t("onboard.title")}</h1>
        <p className="mb-6 mt-1 text-sm leading-6 text-gray-500">
          {t("onboard.intro")}
        </p>
        <CreateProjectForm onCreated={onCreated} />
        <p className="mt-6 border-t border-gray-100 pt-4 text-sm text-gray-500">
          <strong className="text-gray-700">{t("onboard.keyFirst")}</strong> {t("onboard.keyWhy")}{" "}
          <Link to="/settings">{t("onboard.addKeys")}</Link>{" · "}<Link to="/settings/system">{t("nav.system")}</Link>
        </p>
        <LanguagePicker />
      </div>
    </div>
  );
}
