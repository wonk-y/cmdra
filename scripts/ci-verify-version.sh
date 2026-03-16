#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

python3 - <<'PY'
from pathlib import Path
import tomllib

root = Path(".").resolve()
root_version = (root / "VERSION").read_text(encoding="utf-8").strip()
pyproject = tomllib.loads((root / "sdk/python/pyproject.toml").read_text(encoding="utf-8"))
python_version = pyproject["project"]["version"]

if root_version != python_version:
    raise SystemExit(
        f"version mismatch: VERSION={root_version!r} sdk/python/pyproject.toml={python_version!r}"
    )

print(f"version aligned: {root_version}")
PY
