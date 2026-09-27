// SPDX-License-Identifier: AGPL-3.0-or-later

import { useI18n } from "../../i18n";
export function WordCloud({ terms }: { terms: { term: string; count: number }[] }) {
  const { t } = useI18n();
  const items = [...terms].sort((a, b) => b.count - a.count).slice(0, 48);
  if (items.length === 0) return <p className="py-6 text-center text-sm text-gray-500">{t("charts.noWords")}</p>;
  const counts = items.map((item) => item.count);
  const max = Math.max(...counts);
  const min = Math.min(...counts);
  const scale = (count: number) => (max === min ? 0.6 : (Math.sqrt(count) - Math.sqrt(min)) / (Math.sqrt(max) - Math.sqrt(min)));
  const colors = ["#7c3aed", "#4f46e5", "#2563eb", "#0891b2", "#0d9488", "#059669", "#d97706"];
  const ordered: typeof items = [];
  items.forEach((item, index) => {
    if (index % 2 === 0) ordered.push(item);
    else ordered.unshift(item);
  });
  return (
    <div className="flex flex-wrap items-center justify-center gap-x-3 gap-y-1 leading-tight">
      {ordered.map((item) => {
        const weight = scale(item.count);
        const color = colors[Math.min(colors.length - 1, Math.round((1 - weight) * (colors.length - 1)))];
        return (
          <span key={item.term} title={`${item.term} · ${item.count}`} className="font-semibold" style={{ fontSize: Math.round(13 + weight * 27), color }}>
            {item.term}
          </span>
        );
      })}
    </div>
  );
}

export function TermBars({ items }: { items: { label: string; count: number; suffix?: string }[] }) {
  const max = Math.max(1, ...items.map((item) => item.count));
  return (
    <ul className="space-y-3">
      {items.map((item) => (
        <li key={item.label}>
          <div className="mb-1 flex justify-between gap-3 text-sm">
            <span>{item.label}</span>
            <span className="text-gray-500">{item.suffix || item.count}</span>
          </div>
          <div className="h-2 rounded-full bg-gray-100">
            <div className="h-2 rounded-full bg-primary-700" style={{ width: `${Math.max(4, (item.count / max) * 100)}%` }} />
          </div>
        </li>
      ))}
    </ul>
  );
}
