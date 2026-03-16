---
sidebar_position: 1
slug: /
---

# CmdAgent Documentation

CmdAgent is a long-running Go daemon that exposes a gRPC API over mutual TLS for remote process execution and file transfer.

Use this site to:

- install `cmdagentd` on Linux, macOS, and Windows
- configure mTLS and CN-based client authorization
- operate the daemon with `cmdagentctl` or `cmdagentui`
- build against the Go and Python SDKs
- integrate the Python client with Robot Framework and Ansible

## What the daemon provides

`cmdagentd` supports:

- argv command execution
- shell command execution
- persistent shell sessions with attach/reconnect
- stdout and stderr replay from persisted history
- file upload, file download, and archive download
- execution and transfer metadata retention in SQLite under `-data-dir`
- deletion of one finished execution or transfer from history
- clearing finished history for the authenticated identity
- foreground mode and service mode on Linux, macOS, and Windows

## Recommended reading order

1. [Generate development certificates](pathname:///docs/install/certs)
2. [Install on Linux](pathname:///docs/install/linux), [macOS](pathname:///docs/install/macos), or [Windows](pathname:///docs/install/windows)
3. [Use `cmdagentctl`](pathname:///docs/cli/cmdagentctl) or [use `cmdagentui`](pathname:///docs/cli/cmdagentui)
4. [Use the Go SDK](pathname:///docs/sdk/go) or [Python SDK](pathname:///docs/sdk/python)
5. [Integrate Robot Framework](pathname:///docs/integrations/robot-framework) or [Ansible](pathname:///docs/integrations/ansible)

## Repository commands

Build the binaries from the repository root:

```bash
go build ./cmd/cmdagentd
go build ./cmd/cmdagentctl
go build ./cmd/cmdagentui
./cmdagentctl version
./cmdagentui version
```

Generate protobuf stubs:

```bash
./scripts/gen-proto.sh
```

Build release artifacts:

```bash
./scripts/build-release.sh
./scripts/build-python-package.sh
```
