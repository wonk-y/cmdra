# Python SDK Examples

All examples read these environment variables:

- `CMDAGENT_ADDRESS`
- `CMDAGENT_CA`
- `CMDAGENT_CERT`
- `CMDAGENT_KEY`
- `CMDAGENT_SERVER_NAME` (optional)

Run examples from the repository root:

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
