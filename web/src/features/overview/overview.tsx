// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useState } from "react";
import { Link, useOutletContext } from "react-router-dom";
import { getAudit, listOpportunities, listRuns, type AuditReport, type OpportunityItem, type Project, type SampleRunRow } from "../../api";
import { useI18n, type Key } from "../../i18n";
import { opportunityTitle } from "../actions/title";
import { AreaSeries } from "../../components/charts/series";
import { LayerStatus } from "../../components/metrics/LayerStatus";
import { RateStat } from "../../components/metrics/RateStat";
import { keepSearch, useMeasure } from "../measure/data";
import { FilterBar } from "../measure/filters";

const BRAND = "#2563eb";


// Overview answers one question: is the brand in AI answers, measured the
// same way as every other page. Google numbers live under Search.
export function Overview() {
  const m = useMeasure();
  const { slug, data } = m;
  const { t, locale, intl } = useI18n();
  const ctx = useOutletContext<{ project?: Project } | undefined>();
  const when = (unix: number | null | undefined) => (unix ? new Date(unix * 1000).toLocaleString(intl, { hour12: false }) : t("overview.notFinished"));
  const oppTitle = (o: OpportunityItem) => opportunityTitle(o, t, locale);
  const [audit, setAudit] = useState<AuditReport | null>(null);
  const [opps, setOpps] = useState<OpportunityItem[] | null>(null);
  const [runs, setRuns] = useState<SampleRunRow[]>([]);

  useEffect(() => {
    if (!slug) return;
    getAudit(slug).then(setAudit).catch(() => setAudit(null));
    listOpportunities(slug, { status: "new" }).then((d) => setOpps(d.items || [])).catch(() => setOpps([]));
    listRuns(slug).then((d) => setRuns(d.items || [])).catch(() => setRuns([]));
  }, [slug]);

  const layers = audit?.layers || [];
  const passed = layers.filter((l) => l.status === "ok" && !l.blocked_by).length;
  const lastRun = runs[0];
  const q = keepSearch(m.search);
  const points = (data?.visibility_series || []).map((p) => ({ day: p.date, value: p.value }));

  return (
    <div className="mx-auto flex w-full max-w-[1400px] flex-col gap-5">
      <FilterBar {...m} onChange={m.setFilter} />
      {m.err && <p className="text-sm text-red-700">{m.err}</p>}
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <RateStat
          label={`${t("overview.visibility")}${data?.access ? ` (${data.access === "web" ? "Web" : "API"})` : ""}`}
          help={t("notes.visibility")}
          value={data?.visibility}
          ci={data?.visibility_ci}
          n={data?.visibility_n}
          lowSample={data?.low_sample}
          footer={data?.recognition != null ? <>{t("measure.recognition", { v: data.recognition.toFixed(0), n: data.recognition_n ?? 0 })}</> : undefined}
        />
        <RateStat label={t("overview.sov")} help={t("notes.share")} value={data?.share} />
        <RateStat label={t("overview.ownCited")} help={t("notes.ownCited")} value={data?.own_cited} />
        <div className="card p-5">
          <div className="stat-label">{t("overview.readiness")}</div>
          {layers.length === 0 ? (
            <div className="mt-1 text-base text-gray-400">{t("overview.notAudited")}</div>
          ) : (
            <>
              <div className="stat-value mt-1 tabular-nums">{passed} / {layers.length}</div>
              <ul className="mt-2 space-y-1">
                {layers.map((l) => (
                  <li key={l.key} className="flex items-center justify-between gap-2 text-xs text-gray-600">
                    <span>{["access", "discover", "understand", "cite"].includes(l.key) ? t(`layer.names.${l.key}` as Key) : l.name}</span>
                    <LayerStatus status={l.status} blocked={Boolean(l.blocked_by)} />
                  </li>
                ))}
              </ul>
            </>
          )}
        </div>
      </div>

      <section className="card p-5">
        <div className="mb-2 flex items-center justify-between">
          <h2 className="card-title">{t("overview.visByDay")}</h2>
          <Link to={`/p/${slug}/ai/visibility${q}`} className="text-sm">{t("overview.byQuestion")}</Link>
        </div>
        <AreaSeries label={t("overview.visibility")} points={points} color={BRAND} height={200} empty={t("overview.lineEmpty")} />
      </section>

      <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
        <section className="card p-5">
          <h2 className="card-title mb-3">{t("overview.whoMentioned")}</h2>
          {(data?.leaders?.length ?? 0) === 0 ? (
            <p className="text-sm text-gray-500">{t("overview.noMentions")}</p>
          ) : (
            <table className="table">
              <thead><tr className="text-left text-xs text-gray-500"><th className="py-1">{t("measure.brand")}</th><th className="py-1 text-right">{t("measure.mentions")}</th><th className="py-1 text-right">{t("measure.share")}</th></tr></thead>
              <tbody>
                {data!.leaders.slice(0, 6).map((l) => (
                  <tr key={l.name} className="border-t border-gray-100">
                    <td className={`py-1.5 ${l.is_brand ? "font-semibold text-gray-900" : "text-gray-700"}`}>{l.name}{l.is_brand ? t("overview.you") : ""}</td>
                    <td className="py-1.5 text-right tabular-nums">{l.mentions}</td>
                    <td className="py-1.5 text-right tabular-nums">{l.share.toFixed(0)}%</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
        <section className="card p-5">
          <div className="mb-2 flex items-center justify-between">
            <h2 className="card-title">{t("overview.nextSteps")}</h2>
            <Link to={`/p/${slug}/opportunities`} className="text-sm">{t("overview.all")}</Link>
          </div>
          {opps === null ? <p className="text-sm text-gray-500">{t("common.loading")}</p> : opps.length === 0 ? (
            <p className="text-sm text-gray-500">{t("overview.noOpen")}</p>
          ) : (
            <ul className="space-y-1.5">
              {opps.slice(0, 5).map((o) => (
                <li key={o.key} className="flex items-start gap-2 text-sm">
                  <span className="mt-0.5 rounded border border-gray-300 px-1 text-[11px] font-semibold text-gray-700">{o.priority}</span>
                  <Link to={`/p/${slug}/opportunities?key=${encodeURIComponent(o.key)}`} className="text-gray-800">{oppTitle(o)}</Link>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>

      <p className="text-sm text-gray-500">
        {lastRun ? (
          <>{t("overview.lastRun", { when: when(lastRun.finished_at || lastRun.started_at), ok: lastRun.succeeded, planned: lastRun.planned, failed: lastRun.failed, outcome: lastRun.outcome || lastRun.status })}{" "}
            {lastRun.failed > 0 && <Link to={`/p/${slug}/settings/schedule`}>{t("overview.retryFailures")}</Link>}</>
        ) : t("overview.noRuns")}
        {" "}{t("overview.cadence", { n: ctx?.project?.monitor_runs_per_day || 3 })}
      </p>
    </div>
  );
}
