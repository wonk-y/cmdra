---
sidebar_position: 2
---

# Python SDK

The Python client package lives under `sdk/python/cmdra_client` and can be installed in editable mode from the repository root.

## Install the package

```bash
.venv/bin/pip install -e ./sdk/python
```

With optional extras:

```bash
.venv/bin/pip install -e './sdk/python[ansible,robot,dev]'
```

## Create a client

```python
from cmdra_client import Client

client = Client(
    address="127.0.0.1:8443",
    ca_cert="dev/certs/ca.crt",
    client_cert="dev/certs/client-a.crt",
    client_key="dev/certs/client-a.key",
)
```

## Start argv and shell commands

```python
execution = client.start_argv("/bin/echo", ["hello"])
execution = client.start_shell_command("printf 'hello\\n'", shell_binary="/bin/sh")
execution = client.start_shell_session("/bin/sh")
execution = client.start_shell_session("/bin/sh", use_pty=True)
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

## Optional PTY mode

Shell-oriented Python client methods accept `use_pty=True` plus optional `pty_rows` and `pty_cols`:

```python
execution = client.start_shell_command("printf 'hello from pty\\n'", shell_binary="/bin/sh", use_pty=True, pty_rows=24, pty_cols=80)
session = client.start_shell_session("/bin/sh", use_pty=True, pty_rows=24, pty_cols=80)
session.resize_pty(40, 100)
```

PTY mode is implemented on Unix-like platforms and on Windows through ConPTY. PTY-backed output is terminal-style and effectively merged into one stream.

## List and inspect executions

```python
items = client.list_executions()
details = client.get_execution_with_output(execution.execution_id)
```

## Delete history entries

```python
client.delete_execution(execution.execution_id)
result = client.clear_history()
```

`delete_execution` removes one finished execution or transfer from history. `clear_history` deletes all finished history for the authenticated client and reports how many running items were skipped.

## Upload and download files

```python
upload = client.upload_file("./README.md", "/tmp/README.md")
download = client.download_file("/tmp/README.md", "./README.copy")
archive = client.download_archive(["/tmp"], "./tmp.zip")
```

## Example scripts

All examples read these environment variables:

- `CMDRA_ADDRESS`
- `CMDRA_CA`
- `CMDRA_CERT`
- `CMDRA_KEY`
- `CMDRA_SERVER_NAME` optional

Run them from the repository root:

```bash
export PYTHONPATH="$PWD/sdk/python"
export CMDRA_ADDRESS=127.0.0.1:8443
export CMDRA_CA=dev/certs/ca.crt
export CMDRA_CERT=dev/certs/client-a.crt
export CMDRA_KEY=dev/certs/client-a.key

python3 sdk/python/examples/basic_usage.py
python3 sdk/python/examples/list_executions.py
python3 sdk/python/examples/get_execution.py --id exec-123
python3 sdk/python/examples/upload_file.py --local ./README.md --remote /tmp/README.md
python3 sdk/python/examples/download_file.py --remote /tmp/README.md --local ./README.copy
python3 sdk/python/examples/download_archive.py --path /tmp --local ./tmp.zip
```
