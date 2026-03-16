---
sidebar_position: 1
---

# Use cmdagentctl

`cmdagentctl` is the operator-facing CLI for interacting with one `cmdagentd` endpoint over mTLS.

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
cancel
output
attach
upload
download
download-archive
```

## Base invocation

```bash
./cmdagentctl \
  --address 127.0.0.1:8443 \
  --ca dev/certs/ca.crt \
  --cert dev/certs/client-a.crt \
  --key dev/certs/client-a.key \
  list
```

## Start a direct argv command

```bash
./cmdagentctl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  start-argv --binary /bin/echo hello
```

## Start one shell command string

```bash
./cmdagentctl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  start-shell --command "printf 'hello from shell\n'"
```

## Start a persistent shell session

```bash
./cmdagentctl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  start-session --shell /bin/sh
```

## Inspect metadata and replay output

```bash
./cmdagentctl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  get --id exec-123
```

`get` combines metadata from `GetExecution` with persisted output from `ReadOutput`.

## Cancel a running command

```bash
./cmdagentctl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  cancel --id exec-123 --grace 5s
```

## Upload a file

```bash
./cmdagentctl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  upload --local ./README.md --remote /tmp/README.md
```

## Download a file

```bash
./cmdagentctl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  download --remote /tmp/README.md --local ./README.copy
```

## Download an archive

```bash
./cmdagentctl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  download-archive --path /tmp --local ./tmp.zip
```

## Interactive attach

Start a session, then attach by execution ID from another terminal:

```bash
./cmdagentctl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key \
  attach --id exec-123
```
