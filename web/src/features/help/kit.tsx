// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import { IconAlertTriangle, IconBulb } from "@tabler/icons-react";
import type { Key } from "../../i18n";
import type { TipId } from "../../app/tips";

// Topic order is the reading order; the Next button follows it.
export const TOPIC_IDS = [
  "start", "setup", "prompts", "terms", "numbers", "pages", "opportunities", "audit",
  "manual", "providers", "google", "schedule", "access", "troubleshooting", "limits",
] as const;
export type TopicId = (typeof TOPIC_IDS)[number];
export type GroupId = "start" | "read" | "act" | "connect" | "help";

export type Topic = { group: GroupId; label: string; keys: string };

// Kit is what topic bodies use to link into the app. Page names come from
// the nav catalog, so the help text follows any rename.
export type Kit = {
  page: (path: string, label: Key) => ReactNode;
  topic: (id: TopicId, text: string) => ReactNode;
  n: (label: Key) => string;
};

// One HelpDoc per locale. Bodies are prose, so each locale writes its own
// instead of splitting sentences into catalog keys.
export type HelpDoc = {
  groups: Record<GroupId, string>;
  topics: Record<TopicId, Topic>;
  body: (k: Kit) => Record<TopicId, ReactNode>;
  // buttons adds detail to each button's tip, shown under "Buttons on this
  // page" in the button's topic. Every tip must have an entry.
  buttons: (k: Kit) => Record<TipId, ReactNode>;
};

export function makeKit(slug: string, go: (id: TopicId) => void, n: (label: Key) => string): Kit {
  const to = (path: string) => (slug ? `/p/${slug}/${path}` : "/");
  return {
    n,
    page: (path, label) => <Link to={to(path)}>{n(label)}</Link>,
    topic: (id, text) => <button type="button" className="text-primary-600 hover:underline" onClick={() => go(id)}>{text}</button>,
  };
}

export function Tip({ children }: { children: ReactNode }) {
  return <div className="flex gap-3 rounded-xl border border-primary-100 bg-primary-50 px-4 py-3 text-primary-900"><IconBulb size={18} className="mt-0.5 shrink-0 text-primary-600" /><div>{children}</div></div>;
}

export function Warn({ children }: { children: ReactNode }) {
  return <div className="flex gap-3 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-amber-900"><IconAlertTriangle size={18} className="mt-0.5 shrink-0 text-amber-600" /><div>{children}</div></div>;
}

export function Table({ head, rows }: { head: string[]; rows: ReactNode[][] }) {
  return (
    <div className="overflow-x-auto rounded-xl border border-gray-200">
      <table>
        <thead><tr>{head.map((h) => <th key={h}>{h}</th>)}</tr></thead>
        <tbody>{rows.map((r, i) => <tr key={i}>{r.map((c, j) => <td key={j}>{c}</td>)}</tr>)}</tbody>
      </table>
    </div>
  );
}


// Formula shows one formula block. Formulas are written once, in English
// notation, and shared by every language so they cannot drift apart.
export function Formula({ children }: { children: string }) {
  return <pre className="overflow-x-auto whitespace-pre rounded-xl border border-gray-200 bg-gray-50 px-4 py-3 font-mono text-[13px] leading-6 text-gray-800">{children}</pre>;
}

// FX holds every formula the help center shows. Each matches the code that
// computes the number (internal/service/metrics and the services it names);
// change both together.
export const FX = {
  visibility: "visibility = answers that mention the brand / successful answers\n             (unbranded questions only)",
  recognition: "recognition = answers that mention the brand / successful answers\n              (branded questions only)",
  top: "top 1 = answers where the brand is named first / successful unbranded answers\ntop 3 = answers where the brand is among the first three named / successful unbranded answers\n        (brand and competitors ordered by first appearance in the answer)",
  sov: "share of voice = brand mentions / (brand mentions + competitor mentions)\n                 (unbranded questions; the brand counts once per answer,\n                  each competitor named in an answer counts once)",
  ownCite: "answers citing your domain = unbranded answers with a link to your site\n                             / successful unbranded answers\ncitation share             = citations pointing to your site / all citations",
  wilson: "p = x / n        (x answers that mention you, n successful answers)\nz = 1.96         (95%)\ncenter = (p + z²/2n) / (1 + z²/n)\nhalf   = z · √( p(1−p)/n + z²/4n² ) / (1 + z²/n)\ninterval = center ± half\nsmall sample: n < 30",
  wilsonExample: "5 of 20  → 25%, interval 11.2% – 46.9%, small sample\n4 of 6   → 67%, interval 30.0% – 90.3%, small sample",
  change: "difference d = p_after − p_before\n95% interval of d: Newcombe hybrid score method (from both Wilson intervals)\nup    if the whole interval > 0\ndown  if the whole interval < 0\nwithin noise otherwise",
  changeExample: "18/30 → 6/30 : interval −58.6 to −15.3 points → down\n 6/20 → 5/20 : interval crosses 0            → within noise",
  stability: "share_d(x) = citations of domain x on day d / all citations on day d\ndistance(d1, d2) = 1 − Σ_x min(share_d1(x), share_d2(x))   (Bray–Curtis on shares)\nstability = 100 × (1 − average distance between consecutive days)\n< 40 wide open · 40–69 contested · ≥ 70 locked in",
  layers: "layer status = fail  if any critical finding in the layer\n               warn  if any warning (and no critical)\n               pass  otherwise\nlayers after the first failing layer are blocked",
  pageScore: "page score = crawlability (15) + length (15) + structure (20)\n           + extractable blocks (25) + authority (15) + relevance (10)   = 0–100\ngrade: A ≥ 80 · B ≥ 65 · C ≥ 45 · D otherwise\nsite average = mean score of reachable pages",
  pageParts: "crawlability  HTTP 200 +7 (other 2xx/3xx +3) · no noindex +3 · canonical +2 · ≥120 words +3\nlength        15 × factor: ≥1500 words 1.00 · ≥1000 0.85 · ≥600 0.60 · ≥300 0.35 · ≥120 0.15\nstructure     one H1 +4 (several +2) · H2 count ×6 · paragraphs ×5 · list density ×5\nblocks        definition +7 · ≥3 numbers with units +7 · comparison +6 · steps +5\nauthority     date +4 · author +2 · external links ×4 · schema types ×5\nrelevance     share of question keywords in title, H1 and H2, ×10",
  gap: "citation gap on a question = competitor sites are cited AND your site is not\n                             (over the answers in the chosen period)",
  striking: "striking distance : position 4–20 and impressions ≥ 20\nlow click-through : impressions ≥ 50, position ≤ 15 and expected CTR − actual CTR > 2 points\n   expected CTR by position: 1 → 28% · 2 → 15% · 3 → 11% · ≤5 → 7% · ≤10 → 3% · ≤20 → 1%\npage decay        : previous clicks ≥ 10 and change ≤ −25%\ncannibalization   : two or more of your pages get impressions for one query (top page ≥ 20)",
  verify: "audit action   passes when the rule no longer fires in the latest audit\ncitation action passes when visibility on that question is significantly up (Newcombe)\nvisibility drop passes when visibility is no longer significantly below the baseline\nverified → regressed when a later check fails",
  calls: "calls per day   = enabled questions × engines with a key × runs per day\ntokens per call ≈ 60 (question) + 900 (answer) = 960\ntokens per day  ≈ calls per day × 960",
  callsExample: "20 questions × 3 engines × 3 runs = 180 calls ≈ 172,800 tokens a day",
  search: "CTR              = clicks / impressions\naverage position = Σ(position × impressions) / Σ impressions\nsite totals      = Google's date-only totals (not the sum of query rows)",
};
