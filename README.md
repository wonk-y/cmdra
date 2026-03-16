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

Generate development certificates with a local CA, a server certificate for `localhost` and `127.0.0.1`, and two client certificates (`client-a`, `client-b`):

```bash
./scripts/generate-dev-certs.sh certs
```

Notes:

- Server certificates should include SANs when clients verify hostnames.
- Client certificates do not need SANs for the server-side CN-based RBAC in this project.
- The daemon authorizes clients by certificate CN via `--allowed-client-cn`.

## Run The Daemon

```bash
./cmdagentd run \
  --listen-address 127.0.0.1:8443 \
  --server-cert certs/server.crt \
  --server-key certs/server.key \
  --client-ca certs/ca.crt \
  --allowed-client-cn client-a,client-b \
  --data-dir ./data \
  --audit-log ./data/audit.log
```

Backward-compatible direct flag invocation also works:

```bash
./cmdagentd \
  --listen-address 127.0.0.1:8443 \
  --server-cert certs/server.crt \
  --server-key certs/server.key \
  --client-ca certs/ca.crt \
  --allowed-client-cn client-a,client-b \
  --data-dir ./data
```

## JSON Config

`cmdagentd` can also load a JSON config file:

```json
{
  "listen_address": "127.0.0.1:8443",
  "server_cert_file": "certs/server.crt",
  "server_key_file": "certs/server.key",
  "client_ca_file": "certs/ca.crt",
  "allowed_client_cn": "client-a,client-b",
  "data_dir": "data",
  "audit_log_path": "data/audit.log",
  "chunk_size": 32768,
  "flush_interval": "100ms",
  "grace_period": "5s"
}
```

Run it with:

```bash
./cmdagentd run --config ./cmdagentd.json
```

## Service Management

`service install` configures startup-at-boot by default.

### Linux

```bash
sudo ./cmdagentd service install \
  --name cmdagentd \
  --config ./cmdagentd.json

sudo ./cmdagentd service start --name cmdagentd
sudo ./cmdagentd service status --name cmdagentd
sudo ./cmdagentd service uninstall --name cmdagentd
```

### macOS

```bash
sudo ./cmdagentd service install \
  --name cmdagentd \
  --config ./cmdagentd.json

sudo ./cmdagentd service start --name cmdagentd
sudo ./cmdagentd service status --name cmdagentd
sudo ./cmdagentd service uninstall --name cmdagentd
```

### Windows

```powershell
cmdagentd.exe service install --name cmdagentd --config .\cmdagentd.json
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
  --ca certs/ca.crt \
  --cert certs/client-a.crt \
  --key certs/client-a.key \
  list
```

Examples:

```bash
./cmdagentctl --address 127.0.0.1:8443 --ca certs/ca.crt --cert certs/client-a.crt --key certs/client-a.key start-argv --binary /bin/echo hello
./cmdagentctl --address 127.0.0.1:8443 --ca certs/ca.crt --cert certs/client-a.crt --key certs/client-a.key start-shell --command "printf 'hello\n'"
./cmdagentctl --address 127.0.0.1:8443 --ca certs/ca.crt --cert certs/client-a.crt --key certs/client-a.key get --id exec-123
./cmdagentctl --address 127.0.0.1:8443 --ca certs/ca.crt --cert certs/client-a.crt --key certs/client-a.key upload --local ./README.md --remote /tmp/README.md
./cmdagentctl --address 127.0.0.1:8443 --ca certs/ca.crt --cert certs/client-a.crt --key certs/client-a.key download --remote /tmp/README.md --local ./README.copy
```

`get` includes persisted output by internally calling both `GetExecution` and `ReadOutput`.

## SDKs And Wrappers

- Go SDK: `pkg/cmdagentclient`
- Python SDK: `python/cmdagent_client`
- RobotFramework library: `python/cmdagent_client/robot_library.py`
- Ansible connection plugin: `python/cmdagent_client/ansible_plugins/connection/cmdagent.py`

See:

- `examples/go_sdk`
- `examples/python_sdk`
- `python/README.md`
- `python/tests/README.md`

## Python Tests

```bash
./scripts/ci-python.sh
```

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
