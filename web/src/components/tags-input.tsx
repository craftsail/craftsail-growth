// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useMemo, useRef, useState } from "react";
import { useI18n } from "../i18n";

export function TagsInput({
  value,
  onChange,
  options = [],
  placeholder,
  searchPlaceholder,
  disabled = false,
}: {
  value: string[];
  onChange: (next: string[]) => void;
  options?: string[];
  placeholder?: string;
  searchPlaceholder?: string;
  disabled?: boolean;
}) {  const { t } = useI18n();

  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const root = useRef<HTMLDivElement>(null);
  const selected = useMemo(() => new Set(value), [value]);
  const q = query.trim().toLowerCase();
  const filtered = options.filter((item) => !q || item.includes(q));
  const canCreate = !disabled && q.length > 0 && !selected.has(q) && !options.includes(q);

  useEffect(() => {
    function close(event: MouseEvent) {
      if (!root.current?.contains(event.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", close);
    return () => document.removeEventListener("mousedown", close);
  }, []);

  function add(raw: string) {
    const tag = raw.trim().toLowerCase();
    if (!tag || selected.has(tag)) return;
    onChange([...value, tag]);
    setQuery("");
  }

  return (
    <div ref={root} className="relative w-full">
      <div
        role="combobox"
        aria-expanded={open}
        tabIndex={disabled ? -1 : 0}
        onClick={() => !disabled && setOpen(true)}
        className={`flex min-h-9 w-full flex-wrap items-center gap-1 rounded-xl border border-gray-200 bg-white px-2.5 py-1.5 text-sm ${disabled ? "cursor-not-allowed bg-gray-50 opacity-70" : "cursor-text"}`}
      >
        {value.length === 0 && <span className="text-xs text-gray-400">{placeholder ?? t("tags.add")}</span>}
        {value.map((tag) => (
          <span key={tag} className="inline-flex max-w-full items-center gap-1 rounded-md bg-primary-50 px-1.5 py-0.5 text-xs font-medium text-primary-700">
            <span className="truncate">{tag}</span>
            {!disabled && (
              <button type="button" aria-label={`Remove ${tag}`} className="rounded-sm px-0.5 text-gray-400 hover:bg-gray-200 hover:text-gray-700" onClick={(e) => { e.stopPropagation(); onChange(value.filter((item) => item !== tag)); }}>
                ×
              </button>
            )}
          </span>
        ))}
      </div>
      {open && !disabled && (
        <div className="absolute left-0 z-40 mt-2 w-full min-w-56 overflow-hidden rounded-xl border border-gray-200 bg-white shadow-lg">
          <input
            autoFocus
            className="h-9 w-full border-b border-gray-100 px-3 text-sm outline-none"
            placeholder={searchPlaceholder ?? t("tags.search")}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && q) {
                e.preventDefault();
                add(q);
              }
              if (e.key === "Backspace" && query === "" && value.length > 0) onChange(value.slice(0, -1));
            }}
          />
          <div className="max-h-56 overflow-auto py-1">
            {canCreate && (
              <button type="button" className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm hover:bg-gray-50" onClick={() => add(q)}>
                <span className="text-gray-400">+</span>
                <span>{t("tags.create", { q })}</span>
              </button>
            )}
            {filtered.map((tag) => (
              <button key={tag} type="button" className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm hover:bg-gray-50" onClick={() => { if (selected.has(tag)) onChange(value.filter((item) => item !== tag)); else add(tag); }}>
                <span className={`w-4 text-gray-700 ${selected.has(tag) ? "" : "opacity-0"}`}>✓</span>
                <span className="truncate">{tag}</span>
              </button>
            ))}
            {!q && options.length === 0 && <p className="px-3 py-3 text-center text-xs text-gray-400">{t("tags.typeToCreate")}</p>}
            {q && !canCreate && filtered.length === 0 && <p className="px-3 py-4 text-center text-sm text-gray-400">{t("tags.noMatch")}</p>}
          </div>
        </div>
      )}
    </div>
  );
}
