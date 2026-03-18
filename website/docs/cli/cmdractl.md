---
sidebar_position: 1
---

# Use cmdractl

`cmdractl` is the operator-facing CLI for interacting with one `cmdrad` endpoint over mTLS.

If you want a full-screen operator console instead of subcommands, use [`cmdraui`](pathname:///docs/cli/cmdraui).

## Shared connection flags

```text
--address
--ca
--cert
--key
--server-name
--insecure-skip-verify
--timeout
```

## List supported subcommands

```text
start-argv
start-shell
start-session
list
get
delete
clear-history
cancel
output
stdin
attach
upload
download
download-archive
```

## Base invocation

```bash
./cmdractl \
  --address 127.0.0.1:8443 \
  --ca dev/certs/ca.crt \
  --cert dev/certs/client-a.crt \
  --key dev/certs/client-a.key \
  list
```

## Start a direct argv command

```bash
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  start-argv --binary /bin/echo hello
```

## Start one shell command string

```bash
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  start-shell --command "printf 'hello from shell\n'"
```

## Start one shell command string under a PTY

```bash
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  start-shell --shell /bin/sh --pty --pty-rows 24 --pty-cols 80 --command "printf 'hello from pty\n'"
```

## Start a persistent shell session

```bash
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  start-session --shell /bin/sh
```

Add `--pty` when you want terminal-style shell behavior:

```bash
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  start-session --shell /bin/sh --pty --pty-rows 24 --pty-cols 80
```

`--pty-rows` and `--pty-cols` set the initial terminal size. PTY mode is implemented for shell commands and shell sessions on Unix-like platforms and on Windows through ConPTY. PTY-backed output is terminal-style and effectively merged into one stream.

## Inspect metadata and replay output

```bash
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  get --id exec-123
```

`get` combines metadata from `GetExecution` with persisted output from `ReadOutput`.

## Delete one finished execution or transfer from history

```bash
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  delete --id exec-123
```

## Clear finished history for the authenticated identity

```bash
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  clear-history --yes
```

`clear-history` deletes finished executions and transfers for the calling client identity. Running items are preserved and counted in the response.

## Cancel a running command

```bash
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  cancel --id exec-123 --grace 5s
```

## Send stdin to a running shell command or shell session

```bash
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  stdin --id exec-123 --data "printf 'hello from stdin\n'\n"
```

Add `--eof` when the remote process should see end-of-input after the chunk:

```bash
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  stdin --id exec-123 --data "exit\n" --eof
```

`stdin` is the one-shot CLI equivalent of the SDK `write_stdin(execution_id, data, eof=False)` helpers. It opens a short-lived attach stream internally, sends one chunk, and closes it again.

## Upload a file

```bash
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  upload --local ./README.md --remote /tmp/README.md
```

## Download a file

```bash
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  download --remote /tmp/README.md --local ./README.copy
```

## Download an archive

```bash
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  download-archive --path /tmp --local ./tmp.zip
```

## Interactive attach

Start a session, then attach by execution ID from another terminal:

```bash
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  attach --id exec-123
```

When the attached execution uses a PTY, `cmdractl` sends the local terminal size on connect, keeps the remote PTY updated as the terminal is resized, and switches the local terminal into raw mode for a more normal shell experience.
