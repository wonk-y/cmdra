# Python SDK Examples

All examples read these environment variables:

- `CMDRA_ADDRESS`
- `CMDRA_CA`
- `CMDRA_CERT`
- `CMDRA_KEY`
- `CMDRA_SERVER_NAME` (optional)

Run examples from the repository root:

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

History management is available directly from the Python client:

```python
client.delete_execution("exec-123")
result = client.clear_history()
```

Shell-oriented Python client methods also accept `use_pty=True` for prompt-oriented sessions on Unix-like platforms and on Windows through ConPTY.
