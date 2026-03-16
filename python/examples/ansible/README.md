# Ansible Example

Set the plugin path and temp directories, then run:

```bash
export PYTHONPATH="$PWD/python"
export ANSIBLE_CONNECTION_PLUGINS="$PWD/python/cmdagent_client/ansible_plugins"
export ANSIBLE_LOCAL_TEMP="${TMPDIR:-/tmp}/cmdagent-ansible-local"
export ANSIBLE_REMOTE_TEMP="${TMPDIR:-/tmp}/cmdagent-ansible-remote"
mkdir -p "$ANSIBLE_LOCAL_TEMP" "$ANSIBLE_REMOTE_TEMP"

.venv/bin/ansible-playbook -i python/examples/ansible/inventory.ini python/examples/ansible/ping.yml
```
