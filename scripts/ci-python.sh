#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

export PYTHONPATH="$repo_root/sdk/python"
export ANSIBLE_LOCAL_TEMP="${TMPDIR:-/tmp}/cmdagent-ansible-local"
export ANSIBLE_REMOTE_TEMP="${TMPDIR:-/tmp}/cmdagent-ansible-remote"
mkdir -p "$ANSIBLE_LOCAL_TEMP" "$ANSIBLE_REMOTE_TEMP"

.venv/bin/python -m py_compile sdk/python/cmdagent_client/*.py sdk/python/cmdagent_client/ansible_plugins/connection/*.py sdk/python/cmdagent_client/gen/agent/v1/*.py
.venv/bin/pytest -q sdk/python/tests
