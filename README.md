# CmdAgent

CmdAgent is a long-running Go daemon that exposes a gRPC API over mutual TLS for:

- starting argv commands
- starting shell command strings
- starting persistent shell sessions and attaching to them
- replaying stdout/stderr output
- listing and retrieving execution metadata for running and finished jobs
- uploading files
- downloading files and zip archives
- running as a foreground daemon or as an OS service on Linux, macOS, and Windows

Execution and transfer metadata are stored in SQLite under `-data-dir`. Output is persisted there for replay and can be discarded after a successful read.

## Build

```bash
go build ./cmd/cmdagentd
go build ./cmd/cmdagentctl
./cmdagentd version
./cmdagentctl version
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
./cmdagentd run \
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
./cmdagentd \
  --listen-address 127.0.0.1:8443 \
  --server-cert dev/certs/server.crt \
  --server-key dev/certs/server.key \
  --client-ca dev/certs/ca.crt \
  --allowed-client-cn client-a \
  --data-dir ./dev/data
```

## JSON Config

`cmdagentd` can also load a JSON config file:

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
./cmdagentd run --config ./dev/cmdagentd.json
```

## Service Management

`service install` configures startup-at-boot by default.

### Linux

```bash
sudo ./cmdagentd service install \
  --name cmdagentd \
  --config ./dev/cmdagentd.json

sudo ./cmdagentd service start --name cmdagentd
sudo ./cmdagentd service status --name cmdagentd
sudo ./cmdagentd service uninstall --name cmdagentd
```

### macOS

```bash
sudo ./cmdagentd service install \
  --name cmdagentd \
  --config ./dev/cmdagentd.json

sudo ./cmdagentd service start --name cmdagentd
sudo ./cmdagentd service status --name cmdagentd
sudo ./cmdagentd service uninstall --name cmdagentd
```

### Windows

```powershell
cmdagentd.exe service install --name cmdagentd --config .\dev/cmdagentd.json
cmdagentd.exe service start --name cmdagentd
cmdagentd.exe service status --name cmdagentd
cmdagentd.exe service uninstall --name cmdagentd
```

On Windows, the installed service uses the native Service Control Manager execution path.

## `cmdagentctl`

Connection flags are shared across subcommands:

```bash
./cmdagentctl \
  --address 127.0.0.1:8443 \
  --ca dev/certs/ca.crt \
  --cert dev/certs/client-a.crt \
  --key dev/certs/client-a.key \
  list
```

Examples:

```bash
./cmdagentctl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key start-argv --binary /bin/echo hello
./cmdagentctl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key start-shell --command "printf 'hello\n'"
./cmdagentctl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key get --id exec-123
./cmdagentctl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key upload --local ./README.md --remote /tmp/README.md
./cmdagentctl --address 127.0.0.1:8443 --ca dev/certs/ca.crt --cert dev/certs/client-a.crt --key dev/certs/client-a.key download --remote /tmp/README.md --local ./README.copy
```

`get` includes persisted output by internally calling both `GetExecution` and `ReadOutput`.

## Layout

High-level repository structure:

- `cmd/`: Go binaries
- `internal/`: daemon internals
- `pkg/`: exported Go packages
- `proto/`: protobuf source
- `sdk/go/`: Go SDK examples and SDK-facing docs
- `sdk/python/`: Python SDK package, examples, tests, and wrapper integrations
- `dev/`: local development config, certificates, and runtime state
- `website/`: Docusaurus documentation site

See also:

- `sdk/README.md`
- `sdk/go/README.md`
- `dev/cmdagentd.json`

## SDKs And Wrappers

- Go SDK: `pkg/cmdagentclient`
- Python SDK: `sdk/python/cmdagent_client`
- RobotFramework library: `sdk/python/cmdagent_client/robot_library.py`
- Ansible connection plugin: `sdk/python/cmdagent_client/ansible_plugins/connection/cmdagent.py`

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
go test ./pkg/cmdagentclient
go test ./cmd/cmdagentctl
go test ./cmd/cmdagentd
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

## Release Packaging

Build versioned Go release binaries:

```bash
./scripts/build-release.sh
```

Build the Python source and wheel distributions:

```bash
./scripts/build-python-package.sh
```
