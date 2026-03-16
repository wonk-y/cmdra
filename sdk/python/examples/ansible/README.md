# Ansible Example

Set the plugin path and temp directories, then run:

```bash
export PYTHONPATH="$PWD/sdk/python"
export ANSIBLE_CONNECTION_PLUGINS="$PWD/sdk/python/cmdagent_client/ansible_plugins"
export ANSIBLE_LOCAL_TEMP="${TMPDIR:-/tmp}/cmdagent-ansible-local"
export ANSIBLE_REMOTE_TEMP="${TMPDIR:-/tmp}/cmdagent-ansible-remote"
mkdir -p "$ANSIBLE_LOCAL_TEMP" "$ANSIBLE_REMOTE_TEMP"

.venv/bin/ansible-playbook -i sdk/python/examples/ansible/inventory.ini sdk/python/examples/ansible/ping.yml
```

The Ansible connection plugin covers command execution and file transfer. History-management operations such as `delete_execution` and `clear_history` remain SDK and operator-tool features; they are not exposed as Ansible module actions.
