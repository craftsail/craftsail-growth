# AGENTS.md

Instructions for coding agents working in this repository.

## UI text: English, Chinese, Portuguese

Never hardcode UI text. Add the key to `web/src/i18n/locales/en.ts`, `zh.ts` and `pt.ts` in the same change and render it with `t()` from `useI18n()`. Use `tn()` for counts (`_one` / `_other` keys). Page names in help prose come from `n("nav.*")` so a rename reaches the help center. Check a page with `?lang=zh` and `?lang=pt`.

## Navigation

Sidebar sections are flat titles: Overview; AI visibility; Improve; Project; Workspace. Nothing in the sidebar folds. Sub-views of one page are tabs under the page title. Add both in `web/src/app/nav.ts`, and add a redirect there when an address changes.

## UI colors

The palette is defined once in `web/src/styles.css` (`@theme`). Use these Tailwind names. Do not add hex colors or other hues for UI chrome.

### Primary: teal

| Token | Hex | Use |
|---|---|---|
| `primary-50` | `#f0fdfa` | active nav background, selected rows, info alerts |
| `primary-100` | `#ccfbf1` | primary badges, stat icon backgrounds |
| `primary-200` | `#99f6e4` | active chip border |
| `primary-300` | `#5eead4` | |
| `primary-400` | `#2dd4bf` | progress bar end |
| `primary-500` | `#14b8a6` | primary button gradient start, focus ring (`/30`), toggles on |
| `primary-600` | `#0d9488` | primary button gradient end, active nav text, links |
| `primary-700` | `#0f766e` | badge text, hover on links |
| `primary-800` | `#115e59` | |
| `primary-900` | `#134e4a` | text selection |
| `primary-950` | `#042f2e` | |

Primary buttons use `bg-gradient-to-r from-primary-500 to-primary-600` with `shadow-primary-500/25`, and `from-primary-600 to-primary-700` on hover.

### Neutrals: Tailwind gray

| Use | Class |
|---|---|
| Page background | `bg-gray-50` |
| Cards, sidebar, inputs | `bg-white` |
| Main text | `text-gray-900` |
| Secondary text | `text-gray-500`, body copy `text-gray-700` |
| Placeholder, muted icons | `text-gray-400` |
| Borders | `border-gray-200`; card borders and dividers `border-gray-100` |
| Hover background | `bg-gray-50` or `bg-gray-100` |
| Table header | `bg-gray-50`, `text-gray-600` |

Use `gray`, not `slate` or `zinc`. Do not use black (`bg-gray-900`, `bg-black`) for buttons, toggles or tabs; the only dark surface is code blocks (`bg-gray-900 text-gray-100`).

### Status colors

| Meaning | Background | Text |
|---|---|---|
| Success | `bg-emerald-100` (alerts `bg-emerald-50`) | `text-emerald-700` |
| Warning | `bg-amber-100` (alerts `bg-amber-50`) | `text-amber-700` |
| Danger | `bg-red-100` (alerts `bg-red-50`) | `text-red-700` |
| Neutral | `bg-gray-100` | `text-gray-700` |

Status colors mean state only. Always pair them with a word or icon, never color alone.

### Charts

Chart series colors identify an engine or brand and are not part of this palette. Numbers and labels in charts use the gray text colors above, never a series or status color.

## Backend

Every source file starts with `// SPDX-License-Identifier: AGPL-3.0-or-later` (`scripts/license-headers.sh --fix`). The database is SQLite by default and MySQL 8 optionally; raw SQL, upserts and date handling must work on both, with a test on SQLite (see `internal/invoker/sqlite_test.go`). Dependencies point one way: `cli → api → service → repo → model`.
