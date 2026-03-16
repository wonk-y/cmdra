#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

if [ ! -x .venv/bin/python ]; then
  echo "missing .venv/bin/python" >&2
  exit 1
fi

.venv/bin/python -m build --no-isolation ./python --outdir dist/python

echo "python distributions written to dist/python"
