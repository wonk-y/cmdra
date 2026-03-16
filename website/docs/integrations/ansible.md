---
sidebar_position: 2
---

# Ansible

Cmdra ships an Ansible connection plugin backed by the Python SDK.

Plugin path:

```text
sdk/python/cmdra_client/ansible_plugins/connection/cmdra.py
```

## Install the Ansible extra

```bash
.venv/bin/pip install -e './sdk/python[ansible]'
```

## Set the plugin path

From the repository root:

```bash
export PYTHONPATH="$PWD/sdk/python"
export ANSIBLE_CONNECTION_PLUGINS="$PWD/sdk/python/cmdra_client/ansible_plugins"
export ANSIBLE_LOCAL_TEMP="${TMPDIR:-/tmp}/cmdra-ansible-local"
export ANSIBLE_REMOTE_TEMP="${TMPDIR:-/tmp}/cmdra-ansible-remote"
mkdir -p "$ANSIBLE_LOCAL_TEMP" "$ANSIBLE_REMOTE_TEMP"
```

## Inventory variables

The plugin reads:

- `ansible_cmdra_address`
- `ansible_cmdra_ca_cert`
- `ansible_cmdra_client_cert`
- `ansible_cmdra_client_key`
- `ansible_cmdra_server_name` optional

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

This lets standard Ansible modules operate through `cmdrad` rather than SSH.

History-management operations such as deleting one execution from history or clearing finished history are not exposed through the connection plugin. Use `cmdractl`, `cmdraui`, or one of the SDKs for those actions.
