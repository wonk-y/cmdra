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
export CMDAGENT_CA=certs/ca.crt
export CMDAGENT_CERT=certs/client-a.crt
export CMDAGENT_KEY=certs/client-a.key

go run ./examples/go_sdk/start_argv -- /bin/echo hello
go run ./examples/go_sdk/start_shell --command "printf 'hello\n'"
go run ./examples/go_sdk/start_session --shell /bin/sh
go run ./examples/go_sdk/list_executions
go run ./examples/go_sdk/get_execution --id exec-123
go run ./examples/go_sdk/upload_file --local ./README.md --remote /tmp/README.md
go run ./examples/go_sdk/download_file --remote /tmp/README.md --local ./README.copy
go run ./examples/go_sdk/download_archive --path /tmp --local ./tmp.zip
```
