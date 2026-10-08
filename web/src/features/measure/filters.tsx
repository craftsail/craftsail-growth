// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useRef, useState, type ReactNode } from "react";
import { RANGES, type MeasureView } from "./data";
import { useI18n, type Key } from "../../i18n";

const trigger = "chip";

function Menu({ label, icon, active, badge, children }: { label: string; icon: string; active?: boolean; badge?: number; children: ReactNode }) {
  const [open, setOpen] = useState(false);
  const root = useRef<HTMLDivElement>(null);
  useEffect(() => {
    function close(event: MouseEvent) {
      if (!root.current?.contains(event.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", close);
    return () => document.removeEventListener("mousedown", close);
  }, []);
  return (
    <div ref={root} className="relative">
      <button type="button" className={`${trigger} ${active ? "chip-active" : ""}`} aria-expanded={open} onClick={() => setOpen((v) => !v)}>
        <span className="text-gray-400">{icon}</span>
        <span>{label}</span>
        {badge ? <span className="grid h-4 min-w-4 place-items-center rounded-full bg-primary-600 px-1 text-[10px] text-white">{badge}</span> : null}
        <span className="text-xs text-gray-400">⌄</span>
      </button>
      {open && <div className="absolute left-0 z-40 mt-2 w-56 overflow-hidden rounded-xl border border-gray-200 bg-white py-1 shadow-lg">{children}</div>}
    </div>
  );
}

export function Help({ text }: { text: string }) {
  if (!text) return null;
  return (
    <button type="button" className="inline-flex text-gray-400" title={text} aria-label={text}>?</button>
  );
}

export function FilterBar({
  data,
  range,
  platform,
  access = "",
 language="",region="",revision="",
  tags,
  sort,
  q,
  onChange,
  showSort,
  showSearch,
}: {
  data: MeasureView | null;
  range: string;
  platform: string;
  access?: string;
 language?: string;region?: string;revision?: string;
  tags: string[];
  sort: string;
  q: string;
  onChange: (key: string, value: string) => void;
  showSort?: boolean;
  showSearch?: boolean;
}) {
  const engines = data?.engines ?? [];
  const tagOptions = data?.tags ?? [];
  const { t } = useI18n();
  const rangeText = (id: string) => t(`measure.filters.ranges.${RANGES.includes(id as (typeof RANGES)[number]) ? id : "30d"}` as Key);
  const engineLabel = engines.find((item) => item.id === platform)?.label || t("measure.filters.allModels");
  function toggleTag(id: string) {
    const next = tags.includes(id) ? tags.filter((item) => item !== id) : [...tags, id];
    onChange("tags", next.join(","));
  }
  const currentAccess = access || data?.access || "api";
  return (
    <div className="flex flex-wrap items-center gap-2">
      {([{key:"language",value:language,options:data?.languages,label:"sampling.languageFilter"},{key:"region",value:region,options:data?.regions,label:"sampling.regionFilter"},{key:"revision",value:revision,options:data?.revisions,label:"sampling.versionFilter"}] as const).map(f=><label className="text-xs" key={f.key}>{t(f.label)}<select className="input ml-2 max-w-44 text-xs" aria-label={t(f.label)} value={f.value} onChange={e=>onChange(f.key,e.target.value)}><option value="">{t("sampling.all")}</option>{(f.options||[]).map(v=><option key={v} value={v}>{v==="unknown"?t("sampling.unknown"):f.key==="revision"?v.slice(0,12):v}</option>)}</select></label>)}
      {data?.mixed_versions&&<p className="w-full text-xs text-amber-700">{t("sampling.mixed")}</p>}
      {(data?.accesses?.length ?? 0) > 1 && (
        <div className="tabs" title={t("measure.filters.apiWebHint")}>
          {(["api", "web"] as const).map((id) => (
            <button
              key={id}
              type="button"
              className={`tab py-1 ${currentAccess === id ? "tab-active" : ""}`}
              onClick={() => onChange("access", id)}
            >
              {id === "api" ? "API" : "Web"}
            </button>
          ))}
        </div>
      )}
      <Menu icon="▣" label={engineLabel} active={Boolean(platform)}>
        <button type="button" className="flex w-full px-3 py-1.5 text-left text-sm hover:bg-gray-50" onClick={() => onChange("platform", "")}>{t("measure.filters.allModels")}</button>
        {engines.map((item) => (
          <button key={item.id} type="button" className={`flex w-full px-3 py-1.5 text-left text-sm hover:bg-gray-50 ${platform === item.id ? "bg-gray-100" : ""}`} onClick={() => onChange("platform", item.id)}>{item.label}</button>
        ))}
      </Menu>
      <Menu icon="#" label={t("measure.filters.tags")} active={tags.length > 0} badge={tags.length}>
        <div className="flex items-center justify-between border-b border-gray-100 px-3 py-2 text-sm">
          <span className="font-medium">{t("measure.filters.tags")}</span>
          {tags.length > 0 && <button type="button" className="text-xs text-gray-500" onClick={() => onChange("tags", "")}>{t("measure.filters.clear")}</button>}
        </div>
        <div className="max-h-64 overflow-auto">
          {tagOptions.length === 0 && <p className="px-3 py-4 text-center text-sm text-gray-400">{t("measure.filters.noTags")}</p>}
          {tagOptions.map((tag) => {
            const checked = tags.includes(tag.id);
            return (
              <button key={tag.id} type="button" className={`flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm ${checked ? "bg-gray-100" : "hover:bg-gray-50"}`} onClick={() => toggleTag(tag.id)}>
                <span className={`grid h-4 w-4 place-items-center rounded border text-[10px] ${checked ? "border-primary-600 bg-primary-600 text-white" : "border-gray-300"}`}>{checked ? "✓" : ""}</span>
                <span className="capitalize">{tag.label}</span>
              </button>
            );
          })}
        </div>
      </Menu>
      <Menu icon="◷" label={rangeText(range)} active={range !== "30d"}>
        {RANGES.map((id) => (
          <button key={id} type="button" className={`flex w-full px-3 py-1.5 text-left text-sm hover:bg-gray-50 ${range === id ? "bg-gray-100" : ""}`} onClick={() => onChange("range", id)}>{rangeText(id)}</button>
        ))}
      </Menu>
      {showSort && (
        <Menu icon="↕" label={sort === "desc" ? t("measure.filters.highLow") : t("measure.filters.lowHigh")}>
          <button type="button" className="flex w-full px-3 py-1.5 text-left text-sm hover:bg-gray-50" onClick={() => onChange("sort", "asc")}>{t("measure.filters.lowHigh")}</button>
          <button type="button" className="flex w-full px-3 py-1.5 text-left text-sm hover:bg-gray-50" onClick={() => onChange("sort", "desc")}>{t("measure.filters.highLow")}</button>
        </Menu>
      )}
      {showSearch && (
        <input
          className="input min-w-40 flex-1 py-1.5"
          placeholder={t("measure.filters.searchQuestions")}
          aria-label={t("measure.filters.searchQuestions")}
          value={q}
          onChange={(e) => onChange("q", e.target.value)}
        />
      )}
    </div>
  );
}
