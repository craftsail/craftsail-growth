// SPDX-License-Identifier: AGPL-3.0-or-later

import { Link } from "react-router-dom";
import { ReviewConfirmation } from "../onboarding/review";
import { EmptyState } from "../../components/EmptyState";
import { LayerStatus } from "../../components/metrics/LayerStatus";
import { AuditActions, useAudit } from "./shared";
import { useI18n, type Key } from "../../i18n";
import { ruleText } from "../../i18n/rules";

const LAYERS = ["access", "discover", "understand", "cite"];

// Readiness shows the four layers in fix order. A failing layer blocks the
// ones after it: fixing downstream issues first changes nothing engines see.
export function Readiness() {
  const a = useAudit();
  const { t, locale } = useI18n();
  const layers = a.rep?.layers || [];
  const layerName = (key: string, fallback: string) => (LAYERS.includes(key) ? t(`layer.names.${key}` as Key) : fallback);
  return (
    <section className="space-y-4">
      <p className="page-description">{t("audit.readiness.description")}</p>
      <AuditActions a={a} />
      {a.loaded && !a.rep && <EmptyState title={t("audit.notAudited")}>{t("audit.notAuditedHint")}</EmptyState>}
      <ol className="space-y-3">
        {layers.map((l, i) => (
          <li key={l.key} className="card p-5">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <h2 className="card-title">{i + 1}. {layerName(l.key, l.name)}</h2>
              <LayerStatus status={l.status} blocked={Boolean(l.blocked_by)} />
            </div>
            <p className="mt-1 text-sm text-gray-600">{LAYERS.includes(l.key) ? t(`audit.readiness.questions.${l.key}` as Key) : l.question}</p>
            {l.blocked_by && l.status !== "fail" && (
              <p className="mt-2 text-sm text-gray-700">{t("audit.readiness.blockedBy", { layer: l.blocked_by_key ? layerName(l.blocked_by_key, l.blocked_by) : l.blocked_by })}</p>
            )}
            {l.items && l.items.length > 0 ? (
              <ul className="mt-2 list-disc space-y-0.5 pl-5 text-sm text-gray-700">
                {l.items.map((x) => <li key={x.code}>[{t(`severity.${x.severity}` as Key)}] {ruleText(locale, x.code, "title", x.title)} ({x.count})</li>)}
              </ul>
            ) : (l.issues?.length ?? 0) > 0 ? (
              <ul className="mt-2 list-disc space-y-0.5 pl-5 text-sm text-gray-700">
                {l.issues!.map((x) => <li key={x}>{x}</li>)}
              </ul>
            ) : <p className="mt-2 text-sm text-gray-500">{t("audit.readiness.noFindings")}</p>}
            <Link to={`/p/${a.slug}/audit/issues?layer=${l.key}`} className="mt-2 inline-block text-sm">{t("audit.readiness.seeIssues")}</Link>
          </li>
        ))}
      </ol>
      {a.rep?.id && !a.rep.no_site && a.rep.page_count > 0 && <ReviewConfirmation key={`${a.slug}/${a.rep.id}`} slug={a.slug} kind="audit_helpful" auditID={a.rep.id} />}
    </section>
  );
}
