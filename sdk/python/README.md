# Python SDK

Install the package in editable mode from the repository root:

```bash
.venv/bin/pip install -e ./sdk/python
```

Or with optional extras:

```bash
.venv/bin/pip install -e './sdk/python[ansible,robot,dev]'
```

## Proto Generation

From the repository root:

```bash
./scripts/gen-proto.sh
```

## Basic Usage

```python
from cmdagent_client import Client

client = Client(
    address="127.0.0.1:8443",
    ca_cert="dev/certs/ca.crt",
    client_cert="dev/certs/client-a.crt",
    client_key="dev/certs/client-a.key",
)
execution = client.start_argv("/bin/echo", ["hello"])
print(execution.execution_id)
client.close()
```

## Example Commands

```bash
export PYTHONPATH="$PWD/sdk/python"
python3 sdk/python/examples/basic_usage.py
python3 sdk/python/examples/list_executions.py --id exec-123
python3 sdk/python/examples/get_execution.py --id exec-123
python3 sdk/python/examples/upload_file.py --local ./README.md --remote /tmp/README.md
python3 sdk/python/examples/download_file.py --remote /tmp/README.md --local ./README.copy
python3 sdk/python/examples/download_archive.py --path /tmp --local ./tmp.zip
```

## RobotFramework

Install the optional extra if needed:

```bash
.venv/bin/pip install -e './sdk/python[robot]'
```

Run the Robot smoke suite from the repository root:

```bash
export PYTHONPATH="$PWD/sdk/python"
.venv/bin/robot \
  --variable ADDRESS:127.0.0.1:8443 \
  --variable CA_CERT:dev/certs/ca.crt \
  --variable CLIENT_CERT:dev/certs/client-a.crt \
  --variable CLIENT_KEY:dev/certs/client-a.key \
  sdk/python/examples/robot_smoke.robot
```

Run the CI-oriented Robot suite:

```bash
export PYTHONPATH="$PWD/sdk/python"
.venv/bin/robot \
  --variable ADDRESS:127.0.0.1:8443 \
  --variable CA_CERT:dev/certs/ca.crt \
  --variable CLIENT_CERT:dev/certs/client-a.crt \
  --variable CLIENT_KEY:dev/certs/client-a.key \
  sdk/python/examples/robot_ci.robot
```

## Ansible

Install the optional extra if needed:

```bash
.venv/bin/pip install -e './sdk/python[ansible]'
```

Run the packaged connection plugin with the example inventory and playbook:

```bash
export PYTHONPATH="$PWD/sdk/python"
export ANSIBLE_CONNECTION_PLUGINS="$PWD/sdk/python/cmdagent_client/ansible_plugins"
export ANSIBLE_LOCAL_TEMP="${TMPDIR:-/tmp}/cmdagent-ansible-local"
export ANSIBLE_REMOTE_TEMP="${TMPDIR:-/tmp}/cmdagent-ansible-remote"
mkdir -p "$ANSIBLE_LOCAL_TEMP" "$ANSIBLE_REMOTE_TEMP"

.venv/bin/ansible-playbook \
  -i sdk/python/examples/ansible/inventory.ini \
  sdk/python/examples/ansible/ping.yml
```

## Pytest

See `sdk/python/tests/README.md` for the exact test commands.

## CI Commands

From the repository root:

```bash
./scripts/ci-python.sh
./scripts/ci-verify-version.sh
./scripts/ci-verify-generated.sh
./scripts/build-python-package.sh
```

## Python Package Build

From the repository root:

```bash
./scripts/build-python-package.sh
```
