---
sidebar_position: 3
---

# Install on macOS

## Build or copy the binary

From source:

```bash
go build -o cmdagentd ./cmd/cmdagentd
go build -o cmdagentctl ./cmd/cmdagentctl
```

For cross-built release output, use:

```bash
./scripts/build-release.sh
```

Then copy `dist/release/<version>/darwin-amd64/cmdagentd` or `darwin-arm64/cmdagentd` to the target host.

## Create a daemon config file

Create `cmdagentd.json`:

```json
{
  "listen_address": "127.0.0.1:8443",
  "server_cert_file": "certs/server.crt",
  "server_key_file": "certs/server.key",
  "client_ca_file": "certs/ca.crt",
  "allowed_client_cn": "client-a",
  "data_dir": "data",
  "audit_log_path": "data/audit.log",
  "chunk_size": 32768,
  "flush_interval": "100ms",
  "grace_period": "5s"
}
```

## Run in the foreground

```bash
./cmdagentd run --config ./cmdagentd.json
```

## Install as a launchd service

```bash
sudo ./cmdagentd service install \
  --name cmdagentd \
  --config ./cmdagentd.json

sudo ./cmdagentd service start --name cmdagentd
sudo ./cmdagentd service status --name cmdagentd
```

`service install` configures startup at boot through `launchd`.

## Remove the service

```bash
sudo ./cmdagentd service stop --name cmdagentd
sudo ./cmdagentd service uninstall --name cmdagentd
```

## Smoke-test the service flow

```bash
sudo ./scripts/service-smoke-macos.sh
```
