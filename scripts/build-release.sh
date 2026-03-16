#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

export GOCACHE="${GOCACHE:-$repo_root/.gocache}"

version="${VERSION:-$(cat VERSION)}"
commit="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)}"
date="${DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
out_dir="${OUT_DIR:-dist/release/$version}"
mkdir -p "$out_dir"

ldflags=(
  "-X" "cmdagent/internal/buildinfo.Version=$version"
  "-X" "cmdagent/internal/buildinfo.Commit=$commit"
  "-X" "cmdagent/internal/buildinfo.Date=$date"
)

build_one() {
  local goos="$1"
  local goarch="$2"
  local ext=""
  if [[ "$goos" == "windows" ]]; then
    ext=".exe"
  fi
  local target_dir="$out_dir/${goos}-${goarch}"
  mkdir -p "$target_dir"
  GOOS="$goos" GOARCH="$goarch" go build -ldflags "${ldflags[*]}" -o "$target_dir/cmdagentd$ext" ./cmd/cmdagentd
  GOOS="$goos" GOARCH="$goarch" go build -ldflags "${ldflags[*]}" -o "$target_dir/cmdagentctl$ext" ./cmd/cmdagentctl
  GOOS="$goos" GOARCH="$goarch" go build -ldflags "${ldflags[*]}" -o "$target_dir/cmdagentui$ext" ./cmd/cmdagentui
}

build_one linux amd64
build_one linux arm64
build_one darwin amd64
build_one darwin arm64
build_one windows amd64
build_one windows arm64

echo "release binaries written to $out_dir"
