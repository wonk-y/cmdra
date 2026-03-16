# Python Test Commands

From the repository root:

```bash
export PYTHONPATH="$PWD/python"
export ANSIBLE_LOCAL_TEMP="${TMPDIR:-/tmp}/cmdagent-ansible-local"
export ANSIBLE_REMOTE_TEMP="${TMPDIR:-/tmp}/cmdagent-ansible-remote"
mkdir -p "$ANSIBLE_LOCAL_TEMP" "$ANSIBLE_REMOTE_TEMP"

.venv/bin/pytest -q python/tests
```

Run the optional performance check:

```bash
export CMDAGENT_RUN_PERF=1
.venv/bin/pytest -q python/tests/test_performance_checks.py
```
