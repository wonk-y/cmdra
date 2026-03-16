---
sidebar_position: 1
---

# Robot Framework

The Robot wrapper lives at `sdk/python/cmdagent_client/robot_library.py` and exposes the Python SDK as Robot keywords.

## Install the Robot extra

```bash
.venv/bin/pip install -e './sdk/python[robot]'
```

## Import the library

```robot
*** Settings ***
Library    cmdagent_client.robot_library.CmdAgentLibrary    ${ADDRESS}    ${CA_CERT}    ${CLIENT_CERT}    ${CLIENT_KEY}
```

## Available keyword families

The library exposes the same high-level operations as the Python SDK, including:

- `Start Argv`
- `Start Argv Async`
- `Start Shell Command`
- `Start Shell Command Async`
- `Start Shell Session`
- `Start Shell Session Async`
- `Get Execution`
- `List Executions`
- `Get Execution With Output`
- `Cancel Execution`
- `Read Output`
- `Upload File`
- `Upload File Async`
- `Download File`
- `Download File Async`
- `Download Archive`
- `Download Archive Async`

## Run the smoke suite

From the repository root:

```bash
export PYTHONPATH="$PWD/sdk/python"
.venv/bin/robot \
  --variable ADDRESS:127.0.0.1:8443 \
  --variable CA_CERT:dev/certs/ca.crt \
  --variable CLIENT_CERT:dev/certs/client-a.crt \
  --variable CLIENT_KEY:dev/certs/client-a.key \
  sdk/python/examples/robot_smoke.robot
```

Run the CI-oriented suite:

```bash
export PYTHONPATH="$PWD/sdk/python"
.venv/bin/robot \
  --variable ADDRESS:127.0.0.1:8443 \
  --variable CA_CERT:dev/certs/ca.crt \
  --variable CLIENT_CERT:dev/certs/client-a.crt \
  --variable CLIENT_KEY:dev/certs/client-a.key \
  sdk/python/examples/robot_ci.robot
```
