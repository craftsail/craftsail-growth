// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import { IconSearch } from "@tabler/icons-react";
import { useI18n, type Locale } from "../../i18n";
import { TOPIC_IDS, makeKit, type HelpDoc, type TopicId } from "./kit";
import { TIPS, TIP_IDS } from "../../app/tips";
import type { Key } from "../../i18n";
import { en } from "./content/en";
import { zh } from "./content/zh";
import { pt } from "./content/pt";

const DOCS: Record<Locale, HelpDoc> = { en, zh, pt };

// The hash is #<topic> or #<topic>/<button>, the second form from a "?" tip.
function topicFromHash(): TopicId {
  const id = window.location.hash.replace(/^#/, "").split("/")[0];
  return (TOPIC_IDS as readonly string[]).includes(id) ? (id as TopicId) : "start";
}

function anchorFromHash(): string {
  return window.location.hash.replace(/^#/, "").split("/")[1] || "";
}

const prose =
  "space-y-4 text-sm leading-6 text-gray-700 [&_h2]:m-0 [&_h2]:text-xl [&_h2]:font-semibold [&_h2]:text-gray-900 " +
  "[&_h3]:mb-1 [&_h3]:mt-6 [&_h3]:text-base [&_h3]:font-semibold [&_h3]:text-gray-900 [&_ul]:list-disc [&_ul]:space-y-1 [&_ul]:pl-5 " +
  "[&_ol]:list-decimal [&_ol]:space-y-1.5 [&_ol]:pl-5 [&_strong]:font-semibold [&_strong]:text-gray-900 " +
  "[&_table]:w-full [&_table]:text-sm [&_th]:border-b [&_th]:border-gray-200 [&_th]:bg-gray-50 [&_th]:px-3 [&_th]:py-2 [&_th]:text-left [&_th]:text-xs [&_th]:font-medium [&_th]:text-gray-600 " +
  "[&_td]:border-b [&_td]:border-gray-100 [&_td]:px-3 [&_td]:py-2 [&_td]:align-top";

export function Help() {
  const { slug = "" } = useParams();
  const { t, locale } = useI18n();
  const doc = DOCS[locale] || en;
  const [topic, setTopic] = useState<TopicId>(topicFromHash);
  const [q, setQ] = useState("");
  useEffect(() => {
    const onHash = () => setTopic(topicFromHash());
    window.addEventListener("hashchange", onHash);
    return () => window.removeEventListener("hashchange", onHash);
  }, []);

  const go = (id: TopicId) => { window.location.hash = id; setTopic(id); window.scrollTo({ top: 0 }); };
  const kit = useMemo(() => makeKit(slug, go, t), [slug, t]);
  const body = useMemo(() => doc.body(kit), [doc, kit]);
  const buttons = useMemo(() => doc.buttons(kit), [doc, kit]);
  const topicButtons = TIP_IDS.filter((id) => TIPS[id].topic === topic);
  // Scroll to the button a tip linked to and mark it for a moment.
  const [hit, setHit] = useState(anchorFromHash);
  useEffect(() => {
    const onHash = () => setHit(anchorFromHash());
    window.addEventListener("hashchange", onHash);
    return () => window.removeEventListener("hashchange", onHash);
  }, []);
  useEffect(() => {
    if (!hit) return;
    document.getElementById(`btn-${hit}`)?.scrollIntoView({ behavior: "smooth", block: "center" });
    const tm = window.setTimeout(() => setHit(""), 2500);
    return () => window.clearTimeout(tm);
  }, [hit, topic]);
  // Search matches the current language and English, so either works.
  const shown = useMemo(() => {
    const s = q.trim().toLowerCase();
    if (!s) return [...TOPIC_IDS];
    return TOPIC_IDS.filter((id) => {
      const a = doc.topics[id], b = en.topics[id];
      return [a.label, a.keys, b.label, b.keys].join(" ").toLowerCase().includes(s);
    });
  }, [q, doc]);
  const groups = [...new Set(shown.map((id) => doc.topics[id].group))];
  const next = TOPIC_IDS[TOPIC_IDS.indexOf(topic) + 1];

  return (
    <section className="grid gap-6 lg:grid-cols-[240px_minmax(0,1fr)]">
      <p className="page-description lg:col-span-2">{t("helpCenter.description")}</p>
      <nav className="card self-start p-3 lg:sticky lg:top-4" aria-label={t("helpCenter.topics")}>
        <div className="relative mb-3">
          <IconSearch size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          <input className="input py-1.5 pl-9" placeholder={t("helpCenter.search")} value={q} onChange={(e) => setQ(e.target.value)} aria-label={t("helpCenter.search")} />
        </div>
        {groups.map((g) => (
          <div key={g} className="mb-2">
            <div className="px-3 pb-1 pt-2 text-xs font-semibold uppercase tracking-wider text-gray-400">{doc.groups[g]}</div>
            {shown.filter((id) => doc.topics[id].group === g).map((id) => (
              <button key={id} type="button" className={"subnav-link" + (topic === id ? " subnav-link-active" : "")} onClick={() => go(id)}>{doc.topics[id].label}</button>
            ))}
          </div>
        ))}
        {shown.length === 0 && <p className="px-3 py-2 text-sm text-gray-500">{t("helpCenter.noMatch")}</p>}
      </nav>

      <article className={"card p-6 lg:p-8 " + prose}>
        {body[topic]}
        {topicButtons.length > 0 && (
          <>
            <h3>{t("tips.buttons")}</h3>
            <div className="space-y-3">
              {topicButtons.map((id) => (
                <div key={id} id={`btn-${id}`} className={"scroll-mt-24 rounded-xl border px-4 py-3 transition-colors " + (hit === id ? "border-primary-300 bg-primary-50" : "border-gray-200")}>
                  <div className="font-semibold text-gray-900">{t(TIPS[id].label as Key)}</div>
                  <p className="m-0 mt-1">{t(`tips.${id}` as Key)}</p>
                  <div className="mt-1 text-gray-600">{buttons[id]}</div>
                </div>
              ))}
            </div>
          </>
        )}
        {next && (
          <div className="mt-8 flex justify-end border-t border-gray-100 pt-4">
            <button type="button" className="btn btn-secondary" onClick={() => go(next)}>{t("helpCenter.next", { label: doc.topics[next].label })}</button>
          </div>
        )}
      </article>
    </section>
  );
}
