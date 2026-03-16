---
sidebar_position: 4
---

# Install on Windows

## Build or copy the binary

From source in PowerShell:

```powershell
go build -o cmdagentd.exe .\cmd\cmdagentd
go build -o cmdagentctl.exe .\cmd\cmdagentctl
```

For release artifacts, use the cross-build script from the repository root:

```bash
./scripts/build-release.sh
```

Then copy `dist/release/<version>/windows-amd64/cmdagentd.exe` or `windows-arm64/cmdagentd.exe` to the target host.

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

```powershell
.\cmdagentd.exe run --config .\cmdagentd.json
```

## Install as a Windows service

```powershell
.\cmdagentd.exe service install --name cmdagentd --config .\cmdagentd.json
.\cmdagentd.exe service start --name cmdagentd
.\cmdagentd.exe service status --name cmdagentd
```

`service install` configures the service for automatic start at boot.

The installed service uses the native Windows Service Control Manager execution path.

## Remove the service

```powershell
.\cmdagentd.exe service stop --name cmdagentd
.\cmdagentd.exe service uninstall --name cmdagentd
```

## Smoke-test the service flow

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\service-smoke-windows.ps1
```
