// SPDX-License-Identifier: AGPL-3.0-or-later

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { en, type Messages } from "./locales/en";
import { zh } from "./locales/zh";
import { pt } from "./locales/pt";

// Supported interface languages. English is the source catalog; the others
// must have exactly the same keys (TypeScript enforces this through Messages).
export const LOCALES = [
  { id: "en", label: "English", intl: "en-US" },
  { id: "zh", label: "中文", intl: "zh-CN" },
  { id: "pt", label: "Português", intl: "pt-BR" },
] as const;
export type Locale = (typeof LOCALES)[number]["id"];

const CATALOGS: Record<Locale, Messages> = { en, zh, pt };
const STORAGE = "lang";

// Key is every dotted path to a string in the English catalog.
type Leaves<T, P extends string = ""> = {
  [K in keyof T & string]: T[K] extends string ? `${P}${K}` : Leaves<T[K], `${P}${K}.`>;
}[keyof T & string];
export type Key = Leaves<Messages>;
export type Vars = Record<string, string | number>;

function detect(): Locale {
  const fromURL = new URLSearchParams(window.location.search).get("lang");
  if (fromURL && LOCALES.some((l) => l.id === fromURL)) {
    localStorage.setItem(STORAGE, fromURL);
    return fromURL as Locale;
  }
  const saved = localStorage.getItem(STORAGE);
  if (saved && LOCALES.some((l) => l.id === saved)) return saved as Locale;
  for (const lang of navigator.languages || [navigator.language]) {
    const l = lang.toLowerCase();
    if (l.startsWith("zh")) return "zh";
    if (l.startsWith("pt")) return "pt";
    if (l.startsWith("en")) return "en";
  }
  return "en";
}

function lookup(cat: Messages, key: string): string | undefined {
  let cur: unknown = cat;
  for (const part of key.split(".")) {
    if (cur && typeof cur === "object" && part in (cur as Record<string, unknown>)) cur = (cur as Record<string, unknown>)[part];
    else return undefined;
  }
  return typeof cur === "string" ? cur : undefined;
}

function fill(text: string, vars?: Vars): string {
  if (!vars) return text;
  return text.replace(/\{(\w+)\}/g, (m, k: string) => (k in vars ? String(vars[k]) : m));
}

type Ctx = {
  locale: Locale;
  intl: string;
  setLocale: (l: Locale) => void;
  t: (key: Key, vars?: Vars) => string;
  // tn picks `${key}_one` or `${key}_other` by the plural rules of the locale.
  tn: (key: string, n: number, vars?: Vars) => string;
  num: (n: number, opts?: Intl.NumberFormatOptions) => string;
  date: (d: Date | string | number, opts?: Intl.DateTimeFormatOptions) => string;
};

const I18nContext = createContext<Ctx | null>(null);

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(detect);
  const intl = LOCALES.find((l) => l.id === locale)!.intl;
  useEffect(() => {
    document.documentElement.lang = intl;
  }, [intl]);
  const setLocale = useCallback((l: Locale) => {
    localStorage.setItem(STORAGE, l);
    setLocaleState(l);
  }, []);
  const value = useMemo<Ctx>(() => {
    const cat = CATALOGS[locale];
    const t = (key: Key, vars?: Vars) => fill(lookup(cat, key) ?? lookup(en, key) ?? key, vars);
    const plural = new Intl.PluralRules(intl);
    const tn = (key: string, n: number, vars?: Vars) => {
      const form = plural.select(n) === "one" ? "one" : "other";
      const text = lookup(cat, `${key}_${form}`) ?? lookup(cat, `${key}_other`) ?? lookup(en, `${key}_${form}`) ?? key;
      return fill(text, { n, ...vars });
    };
    const nf = new Intl.NumberFormat(intl);
    return {
      locale, intl, setLocale, t, tn,
      num: (n, opts) => (opts ? new Intl.NumberFormat(intl, opts).format(n) : nf.format(n)),
      date: (d, opts) => new Intl.DateTimeFormat(intl, opts || { dateStyle: "medium" }).format(new Date(d)),
    };
  }, [locale, intl, setLocale]);
  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n(): Ctx {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useI18n must be used inside I18nProvider");
  return ctx;
}

