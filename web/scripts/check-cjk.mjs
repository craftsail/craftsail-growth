// SPDX-License-Identifier: AGPL-3.0-or-later

// Fails when UI source contains Chinese characters outside the Chinese
// catalog and labels.ts. Interface text belongs in src/i18n/locales/*.ts;
// stored Chinese data keys are mapped in labels.ts.
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, relative } from "node:path";

const root = new URL("../src/", import.meta.url).pathname;
const allow = new Set(["labels.ts", "i18n/locales/zh.ts", "i18n/rules/zh.ts", "i18n/index.tsx", "features/help/content/zh.tsx"]);
const bad = [];
function walk(dir) {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) walk(p);
    else if (/\.(tsx?|css)$/.test(name) && !allow.has(relative(root, p))) {
      readFileSync(p, "utf8").split("\n").forEach((line, i) => {
        if (/[一-鿿]/.test(line)) bad.push(`${relative(root, p)}:${i + 1}: ${line.trim().slice(0, 80)}`);
      });
    }
  }
}
walk(root);
if (bad.length) {
  console.error("Chinese text outside the i18n catalogs:\n" + bad.join("\n"));
  process.exit(1);
}
console.log("ok: no Chinese text outside the i18n catalogs");
