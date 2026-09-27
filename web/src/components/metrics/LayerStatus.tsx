// SPDX-License-Identifier: AGPL-3.0-or-later

import { useI18n, type Key } from "../../i18n";

// Status uses the reserved status colors and always pairs them with an icon
// and a word, so the state never depends on color alone.
const STATUS: Record<string, { icon: string; word: Key; color: string }> = {
  ok: { icon: "✓", word: "layer.ok", color: "#0ca30c" },
  warn: { icon: "!", word: "layer.warn", color: "#b7791f" },
  fail: { icon: "✕", word: "layer.fail", color: "#d03b3b" },
  blocked: { icon: "⏸", word: "layer.blocked", color: "#64748b" },
};

export function LayerStatus({ status, blocked }: { status: string; blocked?: boolean }) {
  const { t } = useI18n();
  const s = STATUS[blocked && status !== "fail" ? "blocked" : status] || STATUS.ok;
  return (
    <span className="inline-flex items-center gap-1 text-xs font-medium text-gray-700">
      <span aria-hidden className="grid h-4 w-4 place-items-center rounded-full text-[10px] text-white" style={{ background: s.color }}>{s.icon}</span>
      {t(s.word)}
    </span>
  );
}
