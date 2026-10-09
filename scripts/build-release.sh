#!/usr/bin/env bash
# Build the archives and the exact Linux binaries used by the public images.
set -euo pipefail
cd "$(dirname "$0")/.."
version="${1:?usage: build-release.sh v1.2.3 [output-directory]}"
version="${version#v}"
[[ "$version" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] || { echo 'A stable semantic version is required' >&2; exit 1; }
out="${2:-dist/release}"
mkdir -p "$out"
out="$(cd "$out" && pwd)"
# These build variables contain no shell syntax; the repository may be a fork.
repository="${GITHUB_REPOSITORY:-craftsail/craftsail-growth}"
[[ "$repository" =~ ^[a-zA-Z0-9_-]+/[a-zA-Z0-9_.-]+$ ]] || exit 1
commit="$(git rev-parse HEAD)"
date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
pkg=github.com/craftsail/craftsail-growth/internal/pkg/buildinfo
flags="-s -w -X $pkg.Version=$version -X $pkg.Commit=$commit -X $pkg.Date=$date -X $pkg.BuildType=release -X $pkg.Repository=$repository"
# npm ci and npm run build must have completed first.
test -f web/dist/index.html
mkdir -p internal/webembed/dist
find internal/webembed/dist -mindepth 1 ! -name .gitkeep -delete
cp -R web/dist/. internal/webembed/dist/
for os in linux darwin; do
  for arch in amd64 arm64; do
    dir="$out/bin/${os}_${arch}"
    mkdir -p "$dir"
    CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath -ldflags "$flags" -o "$dir/craftsail-growth" ./cmd/craftsail-growth
    tar -czf "$out/craftsail-growth_${version}_${os}_${arch}.tar.gz" -C "$dir" craftsail-growth
  done
done
(cd "$out" && shasum -a 256 craftsail-growth_"${version}"_*.tar.gz > checksums.txt)
