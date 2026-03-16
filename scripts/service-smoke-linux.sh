#!/usr/bin/env bash
set -euo pipefail

if [[ "${EUID:-$(id -u)}" != 0 ]]; then
  echo "run as root" >&2
  exit 1
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

binary="${BINARY:-$repo_root/cmdrad}"
config="${CONFIG:-$repo_root/dev/cmdrad.json}"
name="${NAME:-cmdrad-smoke}"

"$binary" service install --name "$name" --binary "$binary" --config "$config"
"$binary" service start --name "$name"
"$binary" service status --name "$name"
"$binary" service stop --name "$name"
"$binary" service uninstall --name "$name"
