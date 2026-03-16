#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

./scripts/gen-proto.sh
git diff --exit-code -- gen/agent/v1 sdk/python/cmdra_client/gen/agent/v1
