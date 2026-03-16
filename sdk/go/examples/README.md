# Go SDK Examples

All examples read these environment variables:

- `CMDRA_ADDRESS`
- `CMDRA_CA`
- `CMDRA_CERT`
- `CMDRA_KEY`
- `CMDRA_SERVER_NAME` (optional)

Run examples from the repository root:

```bash
export CMDRA_ADDRESS=127.0.0.1:8443
export CMDRA_CA=dev/certs/ca.crt
export CMDRA_CERT=dev/certs/client-a.crt
export CMDRA_KEY=dev/certs/client-a.key

go run ./sdk/go/examples/start_argv -- /bin/echo hello
go run ./sdk/go/examples/start_shell --command "printf 'hello\n'"
go run ./sdk/go/examples/start_shell --pty --command "printf 'hello from pty\n'"
go run ./sdk/go/examples/start_session --shell /bin/sh
go run ./sdk/go/examples/start_session --shell /bin/sh --pty
go run ./sdk/go/examples/list_executions
go run ./sdk/go/examples/get_execution --id exec-123
go run ./sdk/go/examples/upload_file --local ./README.md --remote /tmp/README.md
go run ./sdk/go/examples/download_file --remote /tmp/README.md --local ./README.copy
go run ./sdk/go/examples/download_archive --path /tmp --local ./tmp.zip
```

History management is also available through the Go SDK client:

```go
err := client.DeleteExecution(ctx, "exec-123")
result, err := client.ClearHistory(ctx)
```

PTY mode is available for shell commands and shell sessions on Unix-like platforms and on Windows through ConPTY.
