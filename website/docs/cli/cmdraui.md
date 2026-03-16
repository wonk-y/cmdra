---
sidebar_position: 2
---

# Use cmdraui

`cmdraui` is the terminal user interface for interacting with one `cmdrad` endpoint over mTLS. It is built on the Go client library and covers the same operational surface as `cmdractl` with a keyboard-driven interface.

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

## Start the TUI

```bash
./cmdraui \
  --address 127.0.0.1:8443 \
  --ca dev/certs/ca.crt \
  --cert dev/certs/client-a.crt \
  --key dev/certs/client-a.key
```

`cmdraui version` prints the build version in the same way as the other binaries.

## Layout

The TUI keeps a fixed three-panel layout:

- left panel: navigation
- top-right panel: the active list or form
- bottom-right panel: detail, output, or section guidance

The active section changes from the navigation panel. `tab` and `shift+tab` are the only way to move focus between panels.

## What you can do from the TUI

`cmdraui` supports:

- listing executions and transfers
- inspecting metadata for running and finished jobs
- replaying persisted stdout and stderr output
- starting argv commands
- starting shell commands
- starting persistent shell sessions
- enabling PTY mode for shell commands and shell sessions
- uploading files
- downloading files
- downloading zip archives
- canceling running work
- deleting one finished execution or transfer from history
- clearing finished history for the authenticated identity
- attaching to a running execution or shell session

## Common controls

```text
tab                 next field or panel
shift+tab           previous field or panel
j / k or arrows     drive the focused non-form panel
r                   refresh the current data
a                   attach from the focused execution list
c                   cancel from the focused execution or transfer list
x                   press twice to delete the selected finished execution or transfer
X                   press twice to clear finished history for the authenticated identity
o                   toggle persisted output from the focused detail panel
enter               submit the focused form
[ / ]               switch form mode
?                   toggle help
q                   quit
```

Panel behavior:

- navigation focused: `j/k` switches between `Executions`, `Transfers`, `New Command`, `New Transfer`, and `Connection`
- main panel focused on a list: `j/k` moves through list items
- main panel focused on a form: `tab` and `shift+tab` move between form fields; when the last field is reached, `tab` moves to the next pane, and when the first field is reached, `shift+tab` moves to the previous pane
- the shell and session forms include a `Use PTY` field
- detail panel focused: `j/k` scrolls detail and output content

History cleanup behavior:

- `x` only deletes finished items from history; running items return a failed-precondition error
- `X` clears finished history for the authenticated identity and leaves running items in place

## Attach mode

Attach mode sends most keypresses directly to the remote process. To avoid conflicting with common `tmux` prefixes, `cmdraui` reserves `ctrl+g` as the escape prefix.

```text
ctrl+g q   detach from the live session
ctrl+g c   request cancellation
ctrl+g h   show attach help
ctrl+d     send EOF
```

Attach mode has two behaviors:

- PTY-backed executions switch into a dedicated fullscreen emulator-backed attach view
- non-PTY executions stay on the simpler transcript-oriented attach view

The PTY-backed path is the default attach behavior whenever the selected execution was started with `Use PTY=true`.

PTY notes:

- PTY can be requested from the `New Command` shell and session forms
- `cmdraui` sends its current dimensions when attaching to a PTY-backed execution and updates the remote PTY as the terminal is resized
- when the attached PTY-backed process exits after `ctrl+g c`, `cmdraui` returns to the normal 3-pane layout automatically
- PTY is useful when you want prompt-oriented shell behavior
- PTY-backed output is terminal-style and effectively merged into one stream
- PTY mode is implemented on Unix-like platforms and on Windows through ConPTY
- `cmdraui` is still not a fully complete terminal emulator, so full-screen terminal applications can still render imperfectly even with PTY enabled
- set `CMDRAUI_PTY_DEBUG=1` before starting `cmdraui` if you want a small timing/counter line under the PTY view while tuning performance

## PTY validation checklist

Use the operator checklist when validating prompt behavior, line editing, resize propagation, attach lifecycle, and `cmdractl` comparison behavior:

- [`PTY Attach Checklist`](pathname:///docs/cli/pty-attach-checklist)

## Recommended workflow

1. Start in `Executions` to inspect current and historical jobs.
2. Use `New Command` for argv, shell, or persistent session creation.
3. Switch to `Transfers` to monitor uploads and downloads.
4. Attach to a running shell session from `Executions` when interactive control is needed.

## Current v1 boundary

The `Connection` section shows the live connection configuration, but it does not reconnect with edited values from inside the TUI yet. Launch `cmdraui` with the intended TLS and address flags when starting the program.
