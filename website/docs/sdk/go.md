---
sidebar_position: 1
---

# Go SDK

The Go client lives in `pkg/cmdraclient` and wraps the generated gRPC client with mTLS setup and higher-level helpers.

## Dial a client

```go
package main

import (
  "context"
  "time"

  "cmdra/pkg/cmdraclient"
)

func main() {
  ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
  defer cancel()

  client, err := cmdraclient.Dial(ctx, cmdraclient.DialConfig{
    Address:        "127.0.0.1:8443",
    CAFile:         "dev/certs/ca.crt",
    ClientCertFile: "dev/certs/client-a.crt",
    ClientKeyFile:  "dev/certs/client-a.key",
  })
  if err != nil {
    panic(err)
  }
  defer client.Close()
}
```

## Start argv and shell commands

```go
execution, err := client.StartArgv(ctx, "/bin/echo", []string{"hello"})
execution, err := client.StartShellCommand(ctx, "/bin/sh", "printf 'hello\\n'")
execution, err := client.StartShellSession(ctx, "/bin/sh", nil)
execution, err := client.StartShellCommandWithOptions(ctx, "/bin/sh", "printf 'hello from pty\\n'", cmdraclient.ShellOptions{UsePTY: true})
```

## Asynchronous start helpers

```go
future := client.StartArgvAsync(ctx, "/bin/echo", []string{"hello"})
execution, err := future.Wait()
```

Available async helpers:

- `StartArgvAsync`
- `StartShellCommandAsync`
- `StartShellSessionAsync`
- `UploadFileAsync`
- `DownloadFileAsync`
- `DownloadArchiveAsync`

## Optional PTY mode

Use `ShellOptions{UsePTY: true}` when starting shell commands or shell sessions that should behave like a terminal-attached process.

```go
execution, err := client.StartShellSessionWithOptions(ctx, "/bin/sh", nil, cmdraclient.ShellOptions{
  UsePTY:  true,
  PTYRows: 24,
  PTYCols: 80,
})
session, err := client.Attach(ctx, execution.GetExecutionId(), true, 0)
err = session.ResizePTY(40, 100)
```

PTY mode is implemented on Unix-like platforms and on Windows through ConPTY. PTY-backed output is terminal-style and effectively merged into one stream.

## List and inspect executions

```go
executions, err := client.ListExecutions(ctx, nil)
details, err := client.GetExecutionWithOutput(ctx, execution.GetExecutionId(), false)
```

`GetExecutionWithOutput` returns both metadata and replayed output chunks.

## Send stdin without attaching

```go
err = client.WriteStdin(ctx, execution.GetExecutionId(), []byte("printf 'from-write-stdin\\n'\n"), false)
err = client.WriteStdin(ctx, execution.GetExecutionId(), []byte("exit\n"), true)
```

`WriteStdin` opens a short-lived attach stream under the hood, writes stdin to the target execution, and closes that helper stream again. Use it when you need to feed a running command or shell session by execution ID without holding a full attach session open.

## Delete history entries

```go
err = client.DeleteExecution(ctx, execution.GetExecutionId())
result, err := client.ClearHistory(ctx)
```

`DeleteExecution` removes one finished execution or transfer from history. `ClearHistory` deletes all finished history owned by the authenticated client and reports how many running items were skipped.

## Upload and download files

```go
uploadResp, err := client.UploadFile(ctx, "./README.md", "/tmp/README.md", cmdraclient.UploadOptions{})
downloadResp, err := client.DownloadFile(ctx, "/tmp/README.md", "./README.copy", cmdraclient.DownloadOptions{})
archiveResp, err := client.DownloadArchive(ctx, []string{"/tmp"}, "./tmp.zip", cmdraclient.DownloadOptions{})
```

## Attach to a running session

```go
session, err := client.Attach(ctx, execution.GetExecutionId(), true, 0)
if err != nil {
  panic(err)
}
defer session.CloseSend()
```

## Example programs

All examples read these environment variables:

- `CMDRA_ADDRESS`
- `CMDRA_CA`
- `CMDRA_CERT`
- `CMDRA_KEY`
- `CMDRA_SERVER_NAME` optional

Run them from the repository root:

```bash
export CMDRA_ADDRESS=127.0.0.1:8443
export CMDRA_CA=dev/certs/ca.crt
export CMDRA_CERT=dev/certs/client-a.crt
export CMDRA_KEY=dev/certs/client-a.key

go run ./sdk/go/examples/start_argv -- /bin/echo hello
go run ./sdk/go/examples/start_shell --command "printf 'hello\n'"
go run ./sdk/go/examples/start_session --shell /bin/sh
go run ./sdk/go/examples/list_executions
go run ./sdk/go/examples/get_execution --id exec-123
go run ./sdk/go/examples/upload_file --local ./README.md --remote /tmp/README.md
go run ./sdk/go/examples/download_file --remote /tmp/README.md --local ./README.copy
go run ./sdk/go/examples/download_archive --path /tmp --local ./tmp.zip
```
