param(
  [string]$DaemonBinary = (Join-Path $PSScriptRoot "..\cmdrad.exe"),
  [string]$UIBinary = (Join-Path $PSScriptRoot "..\cmdraui.exe"),
  [string]$Address = "127.0.0.1:8443",
  [string]$ServerCert = (Join-Path $PSScriptRoot "..\dev\certs\server.crt"),
  [string]$ServerKey = (Join-Path $PSScriptRoot "..\dev\certs\server.key"),
  [string]$CA = (Join-Path $PSScriptRoot "..\dev\certs\ca.crt"),
  [string]$ClientCert = (Join-Path $PSScriptRoot "..\dev\certs\client-a.crt"),
  [string]$ClientKey = (Join-Path $PSScriptRoot "..\dev\certs\client-a.key"),
  [string]$AllowedClientCN = "client-a",
  [string]$DataDir = "",
  [int]$TimeoutSeconds = 20
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function New-TempDir {
  $path = Join-Path ([System.IO.Path]::GetTempPath()) ("cmdraui-pty-smoke-" + [guid]::NewGuid().ToString("N"))
  New-Item -ItemType Directory -Path $path | Out-Null
  return $path
}

function Quote-Arg {
  param([string]$Value)
  if ($null -eq $Value) {
    return '""'
  }
  if ($Value -notmatch '[\s"]') {
    return $Value
  }
  return '"' + ($Value -replace '"', '\"') + '"'
}

function Wait-ForPort {
  param(
    [Parameter(Mandatory = $true)][string]$Host,
    [Parameter(Mandatory = $true)][int]$Port,
    [Parameter(Mandatory = $true)][int]$TimeoutSeconds
  )

  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  while ((Get-Date) -lt $deadline) {
    try {
      $client = [System.Net.Sockets.TcpClient]::new()
      $async = $client.BeginConnect($Host, $Port, $null, $null)
      if ($async.AsyncWaitHandle.WaitOne(250)) {
        $client.EndConnect($async)
        $client.Dispose()
        return
      }
      $client.Dispose()
    } catch {
    }
    Start-Sleep -Milliseconds 250
  }

  throw "cmdrad did not become reachable on ${Host}:${Port} within ${TimeoutSeconds}s"
}

if ([string]::IsNullOrWhiteSpace($DataDir)) {
  $DataDir = New-TempDir
}
$removeDataDir = $true
if (-not (Test-Path $DataDir)) {
  New-Item -ItemType Directory -Path $DataDir | Out-Null
} else {
  $removeDataDir = $false
}

$auditLog = Join-Path $DataDir "audit.log"
$daemonStdout = Join-Path $DataDir "cmdrad-stdout.log"
$daemonStderr = Join-Path $DataDir "cmdrad-stderr.log"
$daemonArgs = @(
  "run",
  "--listen-address", $Address,
  "--server-cert", $ServerCert,
  "--server-key", $ServerKey,
  "--client-ca", $CA,
  "--allowed-client-cn", $AllowedClientCN,
  "--data-dir", $DataDir,
  "--audit-log", $auditLog
)

$daemon = Start-Process -FilePath $DaemonBinary -ArgumentList $daemonArgs -PassThru -RedirectStandardOutput $daemonStdout -RedirectStandardError $daemonStderr

try {
  $hostPart = $Address.Split(":")[0]
  $portPart = [int]($Address.Split(":")[1])
  Wait-ForPort -Host $hostPart -Port $portPart -TimeoutSeconds $TimeoutSeconds

  Write-Host ""
  Write-Host "cmdrad is running for interactive cmdraui PTY validation."
  Write-Host ""
  Write-Host "Suggested PTY checks:"
  Write-Host "  1. New Command -> session -> shell=cmd.exe -> Use PTY=true"
  Write-Host "  2. Attach to the new session from Executions"
  Write-Host "  3. Verify prompt, editing, clear/cls, resize, ctrl+g q, ctrl+g c"
  Write-Host "  4. Optionally repeat with powershell as the shell binary"
  Write-Host ""
  Write-Host "Reference checklist:"
  Write-Host "  docs/pty-attach-checklist.md"
  Write-Host ""

  $uiArgs = @(
    "--address", $Address,
    "--ca", $CA,
    "--cert", $ClientCert,
    "--key", $ClientKey
  )

  & $UIBinary @uiArgs
  $uiExit = $LASTEXITCODE
  if ($uiExit -ne 0) {
    throw "cmdraui exited with code $uiExit"
  }
}
finally {
  if ($daemon -and -not $daemon.HasExited) {
    $daemon.Kill()
    $daemon.WaitForExit()
  }
  if ($removeDataDir -and (Test-Path $DataDir)) {
    Remove-Item -Recurse -Force $DataDir
  }
}
