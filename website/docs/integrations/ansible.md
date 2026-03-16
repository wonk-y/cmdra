---
sidebar_position: 2
---

# Ansible

CmdAgent ships an Ansible connection plugin backed by the Python SDK.

Plugin path:

```text
sdk/python/cmdagent_client/ansible_plugins/connection/cmdagent.py
```

## Install the Ansible extra

```bash
.venv/bin/pip install -e './sdk/python[ansible]'
```

## Set the plugin path

From the repository root:

```bash
export PYTHONPATH="$PWD/sdk/python"
export ANSIBLE_CONNECTION_PLUGINS="$PWD/sdk/python/cmdagent_client/ansible_plugins"
export ANSIBLE_LOCAL_TEMP="${TMPDIR:-/tmp}/cmdagent-ansible-local"
export ANSIBLE_REMOTE_TEMP="${TMPDIR:-/tmp}/cmdagent-ansible-remote"
mkdir -p "$ANSIBLE_LOCAL_TEMP" "$ANSIBLE_REMOTE_TEMP"
```

## Inventory variables

The plugin reads:

- `ansible_cmdagent_address`
- `ansible_cmdagent_ca_cert`
- `ansible_cmdagent_client_cert`
- `ansible_cmdagent_client_key`
- `ansible_cmdagent_server_name` optional

## Example run

```bash
.venv/bin/ansible-playbook \
  -i sdk/python/examples/ansible/inventory.ini \
  sdk/python/examples/ansible/ping.yml
```

## What the plugin maps to

- `exec_command` runs a shell command through `start_shell_command`
- `put_file` runs through `upload_file`
- `fetch_file` runs through `download_file`

This lets standard Ansible modules operate through `cmdagentd` rather than SSH.
