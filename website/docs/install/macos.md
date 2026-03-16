---
sidebar_position: 3
---

# Install on macOS

## Build or copy the binary

From source:

```bash
go build -o cmdrad ./cmd/cmdrad
go build -o cmdractl ./cmd/cmdractl
```

For cross-built release output, use:

```bash
./scripts/build-release.sh
```

Then copy `dist/release/<version>/darwin-amd64/cmdrad` or `darwin-arm64/cmdrad` to the target host.

## Create a daemon config file

Create `dev/cmdrad.json`:

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

## Run in the foreground

```bash
./cmdrad run --config ./dev/cmdrad.json
```

## Install as a launchd service

```bash
sudo ./cmdrad service install \
  --name cmdrad \
  --config ./dev/cmdrad.json

sudo ./cmdrad service start --name cmdrad
sudo ./cmdrad service status --name cmdrad
```

`service install` configures startup at boot through `launchd`.

## Remove the service

```bash
sudo ./cmdrad service stop --name cmdrad
sudo ./cmdrad service uninstall --name cmdrad
```

## Smoke-test the service flow

```bash
sudo ./scripts/service-smoke-macos.sh
```
