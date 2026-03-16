# Cmdra

Cmdra is a long-running Go daemon that exposes a gRPC API over mutual TLS for:

- starting argv commands
- starting shell command strings
- starting persistent shell sessions and attaching to them
- optionally running shell commands and shell sessions under a PTY, including Windows ConPTY support
- replaying stdout/stderr output
- listing and retrieving execution metadata for running and finished jobs
- deleting one finished execution or transfer from history
- clearing finished history for the authenticated identity
- uploading files
- downloading files and zip archives
- running as a foreground daemon or as an OS service on Linux, macOS, and Windows

Execution and transfer metadata are stored in SQLite under `-data-dir`. Output is persisted there for replay and can be discarded after a successful read.

## Build

```bash
go build ./cmd/cmdrad
go build ./cmd/cmdractl
go build ./cmd/cmdraui
./cmdrad version
./cmdractl version
./cmdraui version
```

## Proto Generation

Regenerate all Go and Python protobuf stubs with:

```bash
./scripts/gen-proto.sh
```

## Development Certificates

Generate development certificates with a local CA, a server certificate for `localhost` and `127.0.0.1`, and development client certificates:

```bash
./scripts/generate-dev-certs.sh dev/certs
```

Notes:

- Normal operation only requires one client certificate plus the server certificate.
- The development script generates both `client-a` and `client-b` so the test suite can exercise cross-identity authorization paths.
- Server certificates should include DNS/IP SANs for the hostname or IP clients connect to when clients perform normal certificate verification.
- Client certificates do not need SANs for this project's CN-based authorization model.
- The daemon authorizes clients by certificate CN via `--allowed-client-cn`.

## Run The Daemon

```bash
./cmdrad run \
  --listen-address 127.0.0.1:8443 \
  --server-cert dev/certs/server.crt \
  --server-key dev/certs/server.key \
  --client-ca dev/certs/ca.crt \
  --allowed-client-cn client-a \
  --data-dir ./dev/data \
  --audit-log ./dev/data/audit.log
```

Backward-compatible direct flag invocation also works:

```bash
./cmdrad \
  --listen-address 127.0.0.1:8443 \
  --server-cert dev/certs/server.crt \
  --server-key dev/certs/server.key \
  --client-ca dev/certs/ca.crt \
  --allowed-client-cn client-a \
  --data-dir ./dev/data
```

## JSON Config

`cmdrad` can also load a JSON config file:

```json
{
  "listen_address": "127.0.0.1:8443",
  "server_cert_file": "dev/certs/server.crt",
  "server_key_file": "dev/certs/server.key",
  "client_ca_file": "dev/certs/ca.crt",
  "allowed_client_cn": "client-a",
  "data_dir": "dev/data",
  "audit_log_path": "dev/data/audit.log",
  "chunk_size": 32768,
  "flush_interval": "100ms",
  "grace_period": "5s"
}
```

Run it with:

```bash
./cmdrad run --config ./dev/cmdrad.json
```

## Service Management

`service install` configures startup-at-boot by default.

### Linux

```bash
sudo ./cmdrad service install \
  --name cmdrad \
  --config ./dev/cmdrad.json

sudo ./cmdrad service start --name cmdrad
sudo ./cmdrad service status --name cmdrad
sudo ./cmdrad service uninstall --name cmdrad
```

### macOS

```bash
sudo ./cmdrad service install \
  --name cmdrad \
  --config ./dev/cmdrad.json

sudo ./cmdrad service start --name cmdrad
sudo ./cmdrad service status --name cmdrad
sudo ./cmdrad service uninstall --name cmdrad
```

### Windows

```powershell
cmdrad.exe service install --name cmdrad --config .\dev/cmdrad.json
cmdrad.exe service start --name cmdrad
cmdrad.exe service status --name cmdrad
cmdrad.exe service uninstall --name cmdrad
```

On Windows, the installed service uses the native Service Control Manager execution path.

## `cmdractl`

Connection flags are shared across subcommands:

```bash
./cmdractl \
  --address 127.0.0.1:8443 \
  --ca dev/certs/ca.crt \
  --cert dev/certs/client-a.crt \
  --key dev/certs/client-a.key \
  list
```

Examples:

```bash
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key start-argv --binary /bin/echo hello
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key start-shell --command "printf 'hello\n'"
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key start-shell --pty --pty-rows 24 --pty-cols 80 --command "printf 'hello from a PTY\n'"
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key start-session --shell /bin/sh --pty --pty-rows 24 --pty-cols 80
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key get --id exec-123
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key delete --id exec-123
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key clear-history --yes
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key upload --local ./README.md --remote /tmp/README.md
./cmdractl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key download --remote /tmp/README.md --local ./README.copy
```

`get` includes persisted output by internally calling both `GetExecution` and `ReadOutput`.
`delete` removes one finished execution or transfer from persisted history. `clear-history --yes` removes all finished history for the authenticated identity and reports how many running items were preserved.
`--pty` is available on `start-shell` and `start-session`. `--pty-rows` and `--pty-cols` set the initial terminal size. `attach` automatically pushes terminal size changes for PTY-backed executions and switches the local terminal into raw mode for PTY sessions. PTY mode merges terminal-style output into one stream and is implemented on Unix-like platforms plus Windows through ConPTY.

## `cmdraui`

`cmdraui` is a Bubble Tea TUI built on top of the Go SDK. It exposes the same operational surface as `cmdractl` in a keyboard-driven interface for:

- listing executions and transfers
- inspecting metadata and persisted output
- starting argv commands, shell commands, and shell sessions
- optionally enabling PTY mode for shell commands and shell sessions
- uploading files
- downloading files and archives
- canceling running work
- deleting one finished execution or transfer from history
- clearing all finished history for the authenticated identity
- attaching to running executions and shell sessions

Start it with the same connection flags:

```bash
./cmdraui \
  --address 127.0.0.1:8443 \
  --ca dev/certs/ca.crt \
  --cert dev/certs/client-a.crt \
  --key dev/certs/client-a.key
```

Layout:

- left panel: navigation
- top-right panel: active list or form
- bottom-right panel: detail/output/instructions

Common controls:

- `tab` / `shift+tab`: move to the next or previous field or panel
- when the navigation panel is focused, `j/k` changes the active section
- when the main panel is focused on a list, `j/k` moves through the list
- when the main panel is focused on a form, `tab` and `shift+tab` move between form fields
- the shell and session forms include a `Use PTY` field
- when the detail panel is focused, `j/k` scrolls the detail content
- `r`: refresh
- `a`: attach to the selected running execution when the main list is focused
- `c`: cancel the selected running execution or transfer when the main list is focused
- `x`: press twice from the executions or transfers list/detail to delete the selected finished item from history
- `X`: press twice from the executions or transfers sections to clear finished history for the authenticated identity
- `o`: toggle persisted output when the detail panel is focused
- `enter`: submit the active command or transfer form
- `[` / `]`: switch form mode
- `?`: toggle help
- `q`: quit

Attach mode reserves `ctrl+g` as the escape prefix:

- `ctrl+g q`: detach from the live session
- `ctrl+g c`: request cancellation
- `ctrl+g h`: show attach help
- `ctrl+d`: send EOF

Attach behavior splits by execution type:

- PTY-backed executions use the dedicated fullscreen emulator-backed attach view
- non-PTY executions stay on the simpler transcript-oriented attach view

PTY mode is useful for prompt-oriented shells and terminal-aware programs. `cmdraui` automatically sends its current dimensions when attaching to a PTY-backed execution and updates the remote PTY as the terminal is resized. If a PTY-backed process exits after `ctrl+g c`, `cmdraui` returns to the normal 3-pane view automatically. PTY mode merges interactive output into one stream and is implemented on Unix-like platforms plus Windows through ConPTY. `cmdraui` is still not a fully complete terminal emulator, so full-screen TUIs and complex cursor-control applications can still render imperfectly there even though `cmdractl` PTY attach is tighter now.

Set `CMDRAUI_PTY_DEBUG=1` before starting `cmdraui` if you want a small timing/counter line under the PTY view while tuning performance.

Manual PTY verification checklist:

- `docs/pty-attach-checklist.md`

## Layout

High-level repository structure:

- `cmd/`: Go binaries
- `internal/`: daemon internals
- `pkg/`: exported Go packages
- `proto/`: protobuf source
- `sdk/go/`: Go SDK examples and SDK-facing docs
- `sdk/python/`: Python SDK package, examples, tests, and wrapper integrations
- `dev/`: local development config, certificates, and runtime state
- `docs/`: repo-local operator checklists and other non-site reference material
- `website/`: Docusaurus documentation site

See also:

- `sdk/README.md`
- `sdk/go/README.md`
- `dev/cmdrad.json`

## SDKs And Wrappers

- Go SDK: `pkg/cmdraclient`
- Python SDK: `sdk/python/cmdra_client`
- RobotFramework library: `sdk/python/cmdra_client/robot_library.py`
- Ansible connection plugin: `sdk/python/cmdra_client/ansible_plugins/connection/cmdra.py`

See:

- `sdk/go/examples`
- `sdk/python/examples`
- `sdk/python/README.md`
- `sdk/python/tests/README.md`

## Python Tests

```bash
./scripts/ci-python.sh
```

## Documentation Site

The Docusaurus site lives in `website/`.

From the repository root:

```bash
cd website
npm install
npm start
```

Build the static site:

```bash
./scripts/ci-docs.sh
```

The generated site is written to `website/build`.

## GitHub Pages

The repository includes a GitHub Pages deployment workflow at `.github/workflows/docs-pages.yml`.

After pushing the repository to GitHub:

1. Go to `Settings -> Pages`.
2. Set `Source` to `GitHub Actions`.
3. Push to `main` or run the `docs-pages` workflow manually.

The workflow computes the default Pages URL automatically:

- project pages repo: `https://<owner>.github.io/<repo>/`
- user or organization pages repo (`<owner>.github.io`): `https://<owner>.github.io/`

Optional repository variables:

- `DOCS_URL`
  Use this when deploying behind a custom domain or a non-default Pages host.
- `DOCS_BASE_URL`
  Use this when the site should not be served from the default repo-derived base path.

## Go Tests

```bash
./scripts/ci-go.sh
```

Individual packages can still be run directly:

```bash
go test ./pkg/cmdraclient
go test ./cmd/cmdractl
go test ./cmd/cmdrad
```

## CI Verification

Local commands that match the GitHub Actions workflow:

```bash
./scripts/ci-go.sh
./scripts/ci-python.sh
./scripts/ci-docs.sh
./scripts/ci-verify-version.sh
./scripts/ci-verify-generated.sh
./scripts/build-release.sh
./scripts/build-python-package.sh
```

The repository workflow is defined in `.github/workflows/ci.yml`.

## Service Smoke Scripts

Manual service smoke helpers are provided for environments where you can run privileged install/start/stop/uninstall flows:

```bash
sudo ./scripts/service-smoke-linux.sh
sudo ./scripts/service-smoke-macos.sh
```

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\service-smoke-windows.ps1
```

## Windows PTY Smoke Script

When you have access to a Windows host, you can run an automated ConPTY smoke test that:

- starts a foreground `cmdrad`
- verifies a PTY-backed shell command
- verifies a PTY-backed shell session plus attach flow

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\pty-smoke-windows.ps1
```

The script defaults to the repository's `cmdrad.exe`, `cmdractl.exe`, and `dev/certs/*` development certificates.

For interactive `cmdraui` PTY validation on a Windows host, use:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\pty-smoke-windows-cmdraui.ps1
```

That helper starts `cmdrad`, launches `cmdraui` with the matching mTLS flags, and cleans up when you exit the TUI. Use it with:

- `docs/pty-attach-checklist.md`

## Release Packaging

Build versioned Go release binaries:

```bash
./scripts/build-release.sh
```

Build the Python source and wheel distributions:

```bash
./scripts/build-python-package.sh
```
