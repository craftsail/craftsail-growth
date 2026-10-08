// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useMemo, useState } from "react";
import { useParams, useSearchParams } from "react-router-dom";
import { request } from "../../api";

export type Point = { date: string; value: number };
export type Leader = { name: string; mentions: number; share: number; prompts: number; is_brand: boolean };
export type PromptChart = { id: string; text: string; group: string; runs: number; failed: number; visibility: number; x: number; n: number; series: Array<Record<string, number | string>> };
export type Word = { word: string; count: number; share: number };
export type Opt = { id: string; label: string };
export type Gap = { qid: string; text: string; domains: string[] };
type Ref = { qid?: string; text?: string; url?: string; domain?: string; title?: string };
export type Opp = {
  category: string;
  title: string;
  why: string;
  qid?: string;
  urls?: string[];
  difficulty?: string;
  prompts?: Ref[];
  own?: Ref[];
  rivals?: Ref[];
};
export type Chip = { name: string; self: boolean };
export type Cite = { url: string; domain: string; title: string; category: string; page_type: string };
export type Run = {
  id: number;
  platform: string;
  platform_name: string;
  access: string;
  at: number;
  ok: boolean;
  error?: string;
  queries: string[];
  unknown: boolean;
  mentioned: boolean;
  chips: Chip[];
  answer: string;
  raw?: Record<string, unknown>;
  citations: Cite[];
};
export type PromptHead = {
  id: string;
  text: string;
  group: string;
  active: boolean;
  next: string;
  runs: number;
  visibility: number | null;
  series: Array<Record<string, number | string>>;
};

export type MeasureView = {
 languages?: string[]; regions?: string[]; revisions?: string[]; mixed_versions?: boolean;
  brand: string;
  range: string;
  visibility: number | null;
  access: "api" | "web";
  accesses: Array<"api" | "web">;
  visibility_x: number;
  visibility_n: number;
  visibility_ci: { lo: number; hi: number } | null;
  low_sample: boolean;
  recognition: number | null;
  recognition_n: number;
  own_cited: number | null;
  visibility_series: Point[];
  share: number | null;
  share_series: Point[];
  prompts: number;
  runs: number;
  failed: number;
  fail_reason: string;
  citations: number;
  citation_share: number | null;
  unique_domains: number;
  leaders: Leader[];
  prompt_charts: PromptChart[];
  fanout_total: number;
  fanout_unknown: number;
  fanout_known: number;
  fanout_avg: number;
  words: Word[];
  added: Word[];
  preserved: Word[];
  dropped: Word[];
  category_series: Array<Record<string, number | string>>;
  page_type_series: Array<Record<string, number | string>>;
  category_keys: string[];
  page_type_keys: string[];
  opportunities: Opp[];
  gaps: Gap[];
  competitors: string[];
  engines: Opt[];
  tags: Opt[];
  cadence: string;
  updated_at: number;
  truncated: boolean;
  prompt?: PromptHead;
  run_list?: Run[];
  notes: {
    visibility: string;
    share: string;
    citation_share: string;
    fanout: string;
    stack: string;
  };
};

export const RANGES = ["7d", "30d", "90d", "180d", "365d", "all"] as const;

export function pctText(value: number | null | undefined) {
  if (value == null || Number.isNaN(value)) return "—";
  return `${Math.round(value)}%`;
}

export function shareText(value: number) {
  const rounded = Math.round(value * 10) / 10;
  return Number.isInteger(rounded) ? `${rounded}%` : `${rounded.toFixed(1)}%`;
}

// fmtTime formats a unix time for the interface language; 0 means none yet.
export function fmtTime(unix: number, intl?: string) {
  if (!unix) return "";
  return new Date(unix * 1000).toLocaleString(intl, { hour12: false });
}

export function useMeasure(qid = "") {
  const { slug = "" } = useParams();
  const [sp, setSp] = useSearchParams();
  const range = sp.get("range") || "30d";
  const platform = sp.get("platform") || "";
  const access = sp.get("access") || "";
 const language=sp.get("language")||"",region=sp.get("region")||"",revision=sp.get("revision")||"";
  const tags = (sp.get("tags") || "").split(",").map((item) => item.trim()).filter(Boolean);
  const sort = sp.get("sort") || "asc";
  const q = sp.get("q") || "";
  const tab = sp.get("tab") || "mentions";
  const query = useMemo(() => {
    const params = new URLSearchParams();
    params.set("range", range);
    if (platform) params.set("platform", platform);
    if (access) params.set("access", access);
 if(language)params.set("language",language);if(region)params.set("region",region);if(revision)params.set("revision",revision);
    if (tags.length) params.set("tags", tags.join(","));
    if (sort === "desc") params.set("sort", sort);
    if (qid) params.set("qid", qid);
    return params.toString();
  }, [range, platform, access, language,region,revision,tags.join(","), sort, qid]);
  const [data, setData] = useState<MeasureView | null>(null);
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!slug) return;
    let stop = false;
    setLoading(true);
    request<MeasureView>(`/api/projects/${slug}/measure?${query}`)
      .then((view) => {
        if (!stop) {
          setData(view);
          setErr("");
        }
      })
      .catch((e: Error) => {
        if (!stop) setErr(e.message);
      })
      .finally(() => {
        if (!stop) setLoading(false);
      });
    return () => {
      stop = true;
    };
  }, [slug, query]);

  function setFilter(key: string, value: string) {
    const next = new URLSearchParams(sp);
    if (!value) next.delete(key);
    else next.set(key, value);
    setSp(next, { replace: true });
  }

  return { slug, data, err, loading, range, platform, access,language,region,revision,tags, sort, q, tab, setFilter, search: sp.toString() };
}

const BRAND_COLOR = "#2563eb";
const OTHERS_COLOR = "#cbd5e1";
const COMPETITOR_PALETTE = ["#10b981", "#f59e0b", "#8b5cf6", "#ec4899", "#14b8a6", "#f97316"];

export function shareColors(entries: { name: string; isBrand: boolean; mentions: number }[], topN = 6) {
  const map = new Map<string, string>();
  let index = 0;
  for (const entry of entries) {
    if (entry.mentions <= 0) continue;
    if (entry.isBrand) map.set(entry.name, BRAND_COLOR);
    else if (index < topN) map.set(entry.name, COMPETITOR_PALETTE[index++ % COMPETITOR_PALETTE.length]);
    else map.set(entry.name, OTHERS_COLOR);
  }
  return map;
}

export function withDay<T extends Record<string, unknown>>(rows: T[]) {
  return rows.map((row) => ({ ...row, day: String(row.day || row.date || "") }));
}

export function keepSearch(search: string, extra?: Record<string, string>) {
  const params = new URLSearchParams(search);
  if (extra) {
    for (const [key, value] of Object.entries(extra)) {
      if (value) params.set(key, value);
    }
  }
  const text = params.toString();
  return text ? `?${text}` : "";
}
