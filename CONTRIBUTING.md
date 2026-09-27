# Contributing

Thanks for helping. Bug reports, fixes, translations and new engine
integrations are all welcome.

## Before you start

- For anything larger than a small fix, open an issue first so we can agree
  on the approach.
- Security problems go to [SECURITY.md](SECURITY.md), not public issues.

## Development setup

Requires Go 1.23+ and Node.js 20+.

```sh
make build                    # dashboard + binary
./craftsail-growth server     # API and dashboard on 127.0.0.1:8765 (SQLite by default)
cd web && npm install && npm run dev   # dashboard with hot reload
make demo                     # a populated dashboard to work against, no API keys needed
```

`make demo` writes to `./demo` (ignored by git) and creates the user
`demo` / `quillpad-demo-2026`.

### Screenshots

The README images come from the demo. After a visible UI change, start
`make demo` (and `make demo DEMO_LANG=zh` for the Chinese set), then run
`scripts/demo/screenshots.mjs` as described at the top of that file and
compress the result with `pngquant`.

## Checks

CI runs these; please run them before opening a PR:

```sh
go vet ./... && go test ./...   # add -short to skip the 30-second demo run
cd web && npx tsc --noEmit && npm run check:cjk && npm run build
scripts/license-headers.sh    # every source file starts with the SPDX line
```

Go tests run on in-memory SQLite. Code that must behave the same on MySQL
and SQLite (raw SQL, upserts, date handling) needs a test in
`internal/invoker/sqlite_test.go` or next to the repository code.

## Conventions

- UI text, navigation and colors follow [AGENTS.md](AGENTS.md): every string
  goes into `web/src/i18n/locales/en.ts`, `zh.ts` and `pt.ts` in the same
  change.
- New source files start with `// SPDX-License-Identifier: AGPL-3.0-or-later`
  (`scripts/license-headers.sh --fix` adds it).
- Dependencies point one way: `cli → api → service → repo → model`. `jobs`
  and `pipeline` orchestrate other services; other services must not import
  them.
- Keep PRs focused, and say in the description how you tested the change.

## Sign-off

Sign off every commit to certify the
[Developer Certificate of Origin](https://developercertificate.org/):

```sh
git commit -s -m "fix: ..."
```

By contributing, you agree that your contribution is licensed under the
AGPL-3.0-or-later, the license of this project.
