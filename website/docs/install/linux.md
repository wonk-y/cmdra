---
sidebar_position: 2
---

# Install on Linux

## Build or copy the binary

From source:

```bash
go build -o cmdrad ./cmd/cmdrad
go build -o cmdractl ./cmd/cmdractl
```

If you are distributing prebuilt binaries from this repo, create them with:

```bash
./scripts/build-release.sh
```

Then copy `dist/release/<version>/linux-amd64/cmdrad` or `linux-arm64/cmdrad` to the target host.

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

## Install as a systemd service

```bash
sudo ./cmdrad service install \
  --name cmdrad \
  --config ./dev/cmdrad.json

sudo ./cmdrad service start --name cmdrad
sudo ./cmdrad service status --name cmdrad
```

`service install` enables startup at boot.

## Remove the service

```bash
sudo ./cmdrad service stop --name cmdrad
sudo ./cmdrad service uninstall --name cmdrad
```

## Smoke-test the service flow

```bash
sudo ./scripts/service-smoke-linux.sh
```
