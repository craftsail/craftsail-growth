// SPDX-License-Identifier: AGPL-3.0-or-later

// Regenerates the README screenshots from a running demo (make demo).
// Needs Google Chrome and puppeteer-core, installed outside the repo:
//   npm i --prefix /tmp/shots puppeteer-core
//   NODE_PATH=/tmp/shots/node_modules node scripts/demo/screenshots.mjs
// Env: BASE (default http://127.0.0.1:8765), LANG_UI (en|zh|pt), OUT, PAGES.
import { createRequire } from "node:module";
import { join } from "node:path";

const require = createRequire(join(process.env.NODE_PATH || ".", "x.js"));
const puppeteer = require("puppeteer-core");

const base = process.env.BASE || "http://127.0.0.1:8765";
const lang = process.env.LANG_UI || "en";
const out = process.env.OUT || "docs/images";
const chrome = process.env.CHROME || "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome";
const pages = (process.env.PAGES || "overview,ai/visibility,ai/share-of-voice,ai/citations,opportunities,audit").split(",");

const browser = await puppeteer.launch({
  executablePath: chrome,
  headless: "new",
  defaultViewport: { width: 1440, height: 900, deviceScaleFactor: 2 },
});
const page = await browser.newPage();
await page.goto(base + "/", { waitUntil: "networkidle0", timeout: 20000 });
const status = await page.evaluate(async () => {
  const r = await fetch("/api/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username: "demo", password: "quillpad-demo-2026" }),
  });
  return r.status;
});
if (status !== 200) throw new Error(`login failed: ${status}`);
for (const p of pages) {
  await page.goto(`${base}/p/quillpad/${p}?lang=${lang}`, { waitUntil: "networkidle0", timeout: 20000 });
  await new Promise((r) => setTimeout(r, 1800)); // let charts finish animating
  const file = `${out}/${lang}-${p.replace(/^ai\//, "").replace(/\//g, "-")}.png`;
  await page.screenshot({ path: file });
  console.log(file);
}
// Chrome on macOS sometimes never answers close(); do not hang on it.
await Promise.race([browser.close(), new Promise((r) => setTimeout(r, 3000))]);
process.exit(0);
