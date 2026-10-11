// SPDX-License-Identifier: AGPL-3.0-or-later

import { useSearchParams } from "react-router-dom";
import { useI18n } from "../../i18n";
import { GAExplorer } from "./ga-explore";

// Arrivals is one tab for Google Analytics visits: by channel or by landing
// page. The view lives in the URL so links and reloads keep it.
export function Arrivals() {
  const { t } = useI18n();
  const [params, setParams] = useSearchParams();
  const view = params.get("view") === "landing" ? "landing" : "channel";
  const choose = (v: "channel" | "landing") => {
    const next = new URLSearchParams();
    if (params.has("lang")) next.set("lang", params.get("lang")!);
    if (v === "landing") next.set("view", "landing");
    setParams(next);
  };
  return (
    <div className="flex flex-col gap-4">
      <div className="tabs" role="group" aria-label={t("nav.tabs.arrivals")}>
        {(["channel", "landing"] as const).map((v) => (
          <button key={v} type="button" aria-pressed={view === v} onClick={() => choose(v)}
            className={"tab " + (view === v ? "tab-active" : "")}>
            {t(v === "channel" ? "searchViews.channels" : "searchViews.landings")}
          </button>
        ))}
      </div>
      <GAExplorer key={view} report={view} />
    </div>
  );
}
