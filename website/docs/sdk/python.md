---
sidebar_position: 2
---

# Python SDK

The Python client package lives under `python/cmdagent_client` and can be installed in editable mode from the repository root.

## Install the package

```bash
cd python
../.venv/bin/pip install -e .
```

With optional extras:

```bash
cd python
../.venv/bin/pip install -e '.[ansible,robot,dev]'
```

## Create a client

```python
from cmdagent_client import Client

client = Client(
    address="127.0.0.1:8443",
    ca_cert="certs/ca.crt",
    client_cert="certs/client-a.crt",
    client_key="certs/client-a.key",
)
```

## Start argv and shell commands

```python
execution = client.start_argv("/bin/echo", ["hello"])
execution = client.start_shell_command("printf 'hello\\n'", shell_binary="/bin/sh")
execution = client.start_shell_session("/bin/sh")
```

## Asynchronous helpers

```python
future = client.start_argv_async("/bin/echo", ["hello"])
execution = future.result(timeout=10)
```

Available async helpers:

- `start_argv_async`
- `start_shell_command_async`
- `start_shell_session_async`
- `upload_file_async`
- `download_file_async`
- `download_archive_async`

## List and inspect executions

```python
items = client.list_executions()
details = client.get_execution_with_output(execution.execution_id)
```

## Upload and download files

```python
upload = client.upload_file("./README.md", "/tmp/README.md")
download = client.download_file("/tmp/README.md", "./README.copy")
archive = client.download_archive(["/tmp"], "./tmp.zip")
```

## Example scripts

All examples read these environment variables:

- `CMDAGENT_ADDRESS`
- `CMDAGENT_CA`
- `CMDAGENT_CERT`
- `CMDAGENT_KEY`
- `CMDAGENT_SERVER_NAME` optional

Run them from the repository root:

```bash
export PYTHONPATH="$PWD/python"
export CMDAGENT_ADDRESS=127.0.0.1:8443
export CMDAGENT_CA=certs/ca.crt
export CMDAGENT_CERT=certs/client-a.crt
export CMDAGENT_KEY=certs/client-a.key

python3 examples/python_sdk/basic_usage.py
python3 examples/python_sdk/list_executions.py
python3 examples/python_sdk/get_execution.py --id exec-123
python3 examples/python_sdk/upload_file.py --local ./README.md --remote /tmp/README.md
python3 examples/python_sdk/download_file.py --remote /tmp/README.md --local ./README.copy
python3 examples/python_sdk/download_archive.py --path /tmp --local ./tmp.zip
```
