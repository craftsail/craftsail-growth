// SPDX-License-Identifier: AGPL-3.0-or-later

// Fails when a playbook step links to an address that is not a page or tab
// in the navigation catalog, or uses a label that address does not carry,
// so a renamed page or relabeled tab cannot leave a stale step behind.
import { readFileSync } from "node:fs";

const read = (p) => readFileSync(new URL(p, import.meta.url), "utf8");

// stripComments drops "//" line comments: a whole line whose trimmed start
// is "//", and a trailing " // ..." comment when it follows a "," "{" or
// "}". This is how this codebase writes comments in these two files; it is
// not a general-purpose JS comment stripper.
function stripComments(src) {
  return src
    .split("\n")
    .map((line) => {
      if (line.trim().startsWith("//")) return "";
      const m = line.match(/^(.*[,{}])\s*\/\/.*$/);
      return m ? m[1] : line;
    })
    .join("\n");
}

const nav = stripComments(read("../src/app/nav.ts"));
const catalog = stripComments(read("../src/features/playbook/catalog.ts"));

// navLabels maps each "to" address to every label used to reach it: an
// item's own label, and for pages with tabs, each tab's label.
const navLabels = new Map();
for (const m of nav.matchAll(/label:\s*"([^"]+)"[^{}]*?\bto:\s*"([^"]+)"/g)) {
  const [, label, to] = m;
  if (!navLabels.has(to)) navLabels.set(to, new Set());
  navLabels.get(to).add(label);
}

const steps = [...catalog.matchAll(/path:\s*"([^"]+)",\s*label:\s*"([^"]+)"/g)]
  .map((m) => ({ path: m[1], label: m[2] }));
const cadenceCount = [...catalog.matchAll(/\bcadence:\s*"/g)].length;

if (!steps.length) {
  console.error("check-playbook: no step pages found; did the catalog format change?");
  process.exit(1);
}

const problems = [];
if (steps.length !== cadenceCount) {
  problems.push(`matched ${steps.length} step pages but ${cadenceCount} "cadence:" entries; the step regex missed one`);
}
for (const { path, label } of steps) {
  const labels = navLabels.get(path);
  if (!labels) {
    problems.push(`step page "${path}" is not in NAV`);
  } else if (!labels.has(label)) {
    problems.push(`step page "${path}" uses label "${label}", which NAV does not use for that address (NAV has: ${[...labels].join(", ")})`);
  }
}

if (problems.length) {
  console.error("Playbook steps do not match NAV:\n" + problems.join("\n"));
  process.exit(1);
}
console.log(`ok: ${steps.length} playbook links, all in NAV with matching labels`);
