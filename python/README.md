# Python SDK

Install the package in editable mode from the repository root:

```bash
cd python
../.venv/bin/pip install -e .
```

Or with optional extras:

```bash
cd python
../.venv/bin/pip install -e '.[ansible,robot,dev]'
```

## Basic Usage

```python
from cmdagent_client import Client

client = Client(
    address="127.0.0.1:8443",
    ca_cert="certs/ca.crt",
    client_cert="certs/client-a.crt",
    client_key="certs/client-a.key",
)
execution = client.start_argv("/bin/echo", ["hello"])
print(execution.execution_id)
client.close()
```

## Example Commands

```bash
export PYTHONPATH="$PWD/python"
python3 examples/python_sdk/basic_usage.py
python3 examples/python_sdk/list_executions.py --id exec-123
python3 examples/python_sdk/get_execution.py --id exec-123
python3 examples/python_sdk/upload_file.py --local ./README.md --remote /tmp/README.md
python3 examples/python_sdk/download_file.py --remote /tmp/README.md --local ./README.copy
python3 examples/python_sdk/download_archive.py --path /tmp --local ./tmp.zip
```

## RobotFramework

Install the optional extra if needed:

```bash
cd python
../.venv/bin/pip install -e '.[robot]'
```

Run the Robot smoke suite from the repository root:

```bash
export PYTHONPATH="$PWD/python"
.venv/bin/robot \
  --variable ADDRESS:127.0.0.1:8443 \
  --variable CA_CERT:certs/ca.crt \
  --variable CLIENT_CERT:certs/client-a.crt \
  --variable CLIENT_KEY:certs/client-a.key \
  python/examples/robot_smoke.robot
```

Run the CI-oriented Robot suite:

```bash
export PYTHONPATH="$PWD/python"
.venv/bin/robot \
  --variable ADDRESS:127.0.0.1:8443 \
  --variable CA_CERT:certs/ca.crt \
  --variable CLIENT_CERT:certs/client-a.crt \
  --variable CLIENT_KEY:certs/client-a.key \
  python/examples/robot_ci.robot
```

## Ansible

Install the optional extra if needed:

```bash
cd python
../.venv/bin/pip install -e '.[ansible]'
```

Run the packaged connection plugin with the example inventory and playbook:

```bash
export PYTHONPATH="$PWD/python"
export ANSIBLE_CONNECTION_PLUGINS="$PWD/python/cmdagent_client/ansible_plugins"
export ANSIBLE_LOCAL_TEMP="${TMPDIR:-/tmp}/cmdagent-ansible-local"
export ANSIBLE_REMOTE_TEMP="${TMPDIR:-/tmp}/cmdagent-ansible-remote"
mkdir -p "$ANSIBLE_LOCAL_TEMP" "$ANSIBLE_REMOTE_TEMP"

.venv/bin/ansible-playbook \
  -i python/examples/ansible/inventory.ini \
  python/examples/ansible/ping.yml
```

## Pytest

See `python/tests/README.md` for the exact test commands.
