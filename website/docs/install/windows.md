---
sidebar_position: 4
---

# Install on Windows

## Build or copy the binary

From source in PowerShell:

```powershell
go build -o cmdrad.exe .\cmd\cmdrad
go build -o cmdractl.exe .\cmd\cmdractl
```

For release artifacts, use the cross-build script from the repository root:

```bash
./scripts/build-release.sh
```

Then copy `dist/release/<version>/windows-amd64/cmdrad.exe` or `windows-arm64/cmdrad.exe` to the target host.

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

```powershell
.\cmdrad.exe run --config .\dev/cmdrad.json
```

## Install as a Windows service

```powershell
.\cmdrad.exe service install --name cmdrad --config .\dev/cmdrad.json
.\cmdrad.exe service start --name cmdrad
.\cmdrad.exe service status --name cmdrad
```

`service install` configures the service for automatic start at boot.

The installed service uses the native Windows Service Control Manager execution path.

## Remove the service

```powershell
.\cmdrad.exe service stop --name cmdrad
.\cmdrad.exe service uninstall --name cmdrad
```

## Smoke-test the service flow

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\service-smoke-windows.ps1
```

## Smoke-test Windows PTY and ConPTY

Use the Windows PTY smoke helper when you want to validate PTY-backed shell command execution and PTY-backed attach behavior on a real Windows host:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\pty-smoke-windows.ps1
```

The script starts a foreground daemon with the repository's development certificates, runs a PTY-backed `cmd.exe` command, then starts a PTY-backed `cmd.exe` session and verifies that `cmdractl attach` can drive it.

## Interactive cmdraui PTY validation on Windows

When you want to validate the fullscreen PTY attach experience in `cmdraui` on a real Windows host, use the interactive helper:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\pty-smoke-windows-cmdraui.ps1
```

That helper:

- starts a foreground daemon with the development certificates
- launches `cmdraui` with the matching mTLS flags
- leaves cleanup to the script after you exit the TUI

Use it together with:

- [`PTY Attach Checklist`](pathname:///docs/cli/pty-attach-checklist)
