// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Locale } from "..";
import { zh } from "./zh";
import { pt } from "./pt";

type RuleText = { title: string; why: string; fix: string };

// Audit rule texts in other languages. English comes from the backend
// registry (internal/service/audit/issues.go), passed in as the fallback.
const RULES: Partial<Record<Locale, Record<string, RuleText>>> = { zh, pt };

export function ruleText(locale: Locale, code: string, field: keyof RuleText, fallback: string): string {
  return RULES[locale]?.[code]?.[field] || fallback;
}
