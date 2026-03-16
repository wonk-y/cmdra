# Go SDK

The exported Go client library lives at `pkg/cmdagentclient`.

The `sdk/go/` tree contains example programs that exercise the public client API:

- `sdk/go/examples/start_argv`
- `sdk/go/examples/start_shell`
- `sdk/go/examples/start_session`
- `sdk/go/examples/list_executions`
- `sdk/go/examples/get_execution`
- `sdk/go/examples/upload_file`
- `sdk/go/examples/download_file`
- `sdk/go/examples/download_archive`

Run them from the repository root. See `sdk/go/examples/README.md` for exact commands.

The Go SDK also exposes history-management helpers for:

- deleting one finished execution or transfer with `DeleteExecution`
- clearing finished history for the authenticated identity with `ClearHistory`
