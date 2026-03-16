# Go SDK Examples

All examples read these environment variables:

- `CMDAGENT_ADDRESS`
- `CMDAGENT_CA`
- `CMDAGENT_CERT`
- `CMDAGENT_KEY`
- `CMDAGENT_SERVER_NAME` (optional)

Run examples from the repository root:

```bash
export CMDAGENT_ADDRESS=127.0.0.1:8443
export CMDAGENT_CA=dev/certs/ca.crt
export CMDAGENT_CERT=dev/certs/client-a.crt
export CMDAGENT_KEY=dev/certs/client-a.key

go run ./sdk/go/examples/start_argv -- /bin/echo hello
go run ./sdk/go/examples/start_shell --command "printf 'hello\n'"
go run ./sdk/go/examples/start_session --shell /bin/sh
go run ./sdk/go/examples/list_executions
go run ./sdk/go/examples/get_execution --id exec-123
go run ./sdk/go/examples/upload_file --local ./README.md --remote /tmp/README.md
go run ./sdk/go/examples/download_file --remote /tmp/README.md --local ./README.copy
go run ./sdk/go/examples/download_archive --path /tmp --local ./tmp.zip
```
