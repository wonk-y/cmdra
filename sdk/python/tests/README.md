# Python Test Commands

From the repository root:

```bash
export PYTHONPATH="$PWD/sdk/python"
export ANSIBLE_LOCAL_TEMP="${TMPDIR:-/tmp}/cmdagent-ansible-local"
export ANSIBLE_REMOTE_TEMP="${TMPDIR:-/tmp}/cmdagent-ansible-remote"
mkdir -p "$ANSIBLE_LOCAL_TEMP" "$ANSIBLE_REMOTE_TEMP"

.venv/bin/pytest -q ./sdk/python/tests
```

Run the optional performance check:

```bash
export CMDAGENT_RUN_PERF=1
.venv/bin/pytest -q ./sdk/python/tests/test_performance_checks.py
```

Run the same Python test entrypoint used by CI:

```bash
./scripts/ci-python.sh
```

The Python suite now includes history-management coverage for:

- deleting one finished execution from history
- clearing finished history while preserving running items
- cross-identity denial for history deletion
