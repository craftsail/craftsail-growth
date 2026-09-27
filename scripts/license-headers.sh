#!/usr/bin/env bash
# Checks that every source file starts with the SPDX license line.
#   scripts/license-headers.sh        list files without it, exit 1 if any
#   scripts/license-headers.sh --fix  add it where it is missing
set -euo pipefail

cd "$(dirname "$0")/.."
id="SPDX-License-Identifier: AGPL-3.0-or-later"
fix=0
[ "${1:-}" = "--fix" ] && fix=1

files() {
	git ls-files --cached --others --exclude-standard -- \
		'*.go' 'web/*.ts' 'web/*.tsx' 'web/*.mjs' 'web/*.css'
}

missing=0
while IFS= read -r f; do
	[ -f "$f" ] || continue
	head -n 1 "$f" | grep -q "$id" && continue
	if [ "$fix" = 1 ]; then
		case "$f" in
		*.css) line="/* $id */" ;;
		*) line="// $id" ;;
		esac
		# The blank line keeps the header from becoming a Go package comment.
		{ printf '%s\n\n' "$line"; cat "$f"; } >"$f.tmp" && mv "$f.tmp" "$f"
	else
		echo "missing license header: $f"
		missing=1
	fi
done < <(files)
exit "$missing"
