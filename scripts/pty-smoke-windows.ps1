param(
  [string]$DaemonBinary = (Join-Path $PSScriptRoot "..\cmdagentd.exe"),
  [string]$CtlBinary = (Join-Path $PSScriptRoot "..\cmdagentctl.exe"),
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
  $path = Join-Path ([System.IO.Path]::GetTempPath()) ("cmdagent-pty-smoke-" + [guid]::NewGuid().ToString("N"))
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

function Get-CtlBaseArgs {
  return @(
    "--address", $Address,
    "--ca", $CA,
    "--cert", $ClientCert,
    "--key", $ClientKey
  )
}

function Invoke-Ctl {
  param(
    [Parameter(Mandatory = $true)][string[]]$Args,
    [switch]$AllowFailure
  )

  $allArgs = @()
  $allArgs += Get-CtlBaseArgs
  $allArgs += $Args

  $output = & $CtlBinary @allArgs 2>&1 | Out-String
  $exitCode = $LASTEXITCODE
  if (-not $AllowFailure -and $exitCode -ne 0) {
    throw "cmdagentctl failed with exit code $exitCode`n$output"
  }
  return @{
    Output = $output
    ExitCode = $exitCode
  }
}

function Wait-ForDaemon {
  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  while ((Get-Date) -lt $deadline) {
    $result = Invoke-Ctl -Args @("list") -AllowFailure
    if ($result.ExitCode -eq 0) {
      return
    }
    Start-Sleep -Milliseconds 250
  }
  throw "cmdagentd did not become ready within ${TimeoutSeconds}s"
}

function Get-ExecutionId {
  param([string]$Output)
  $match = [regex]::Match($Output, 'ID:\s+(\S+)')
  if (-not $match.Success) {
    throw "could not parse execution id from output`n$Output"
  }
  return $match.Groups[1].Value
}

function Wait-ForGetOutput {
  param(
    [Parameter(Mandatory = $true)][string]$ExecutionId,
    [Parameter(Mandatory = $true)][string[]]$Contains
  )

  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  while ((Get-Date) -lt $deadline) {
    $result = Invoke-Ctl -Args @("get", "--id", $ExecutionId) -AllowFailure
    if ($result.ExitCode -eq 0) {
      $allPresent = $true
      foreach ($needle in $Contains) {
        if ($result.Output -notmatch [regex]::Escape($needle)) {
          $allPresent = $false
          break
        }
      }
      if ($allPresent) {
        return $result.Output
      }
    }
    Start-Sleep -Milliseconds 250
  }
  throw "timed out waiting for metadata/output for $ExecutionId"
}

function Invoke-Attach {
  param(
    [Parameter(Mandatory = $true)][string]$ExecutionId,
    [Parameter(Mandatory = $true)][string[]]$InputLines
  )

  $allArgs = @()
  $allArgs += Get-CtlBaseArgs
  $allArgs += @("attach", "--id", $ExecutionId)

  $psi = [System.Diagnostics.ProcessStartInfo]::new()
  $psi.FileName = $CtlBinary
  $psi.Arguments = ($allArgs | ForEach-Object { Quote-Arg $_ }) -join " "
  $psi.UseShellExecute = $false
  $psi.RedirectStandardInput = $true
  $psi.RedirectStandardOutput = $true
  $psi.RedirectStandardError = $true

  $proc = [System.Diagnostics.Process]::Start($psi)
  try {
    foreach ($line in $InputLines) {
      $proc.StandardInput.WriteLine($line)
    }
    $proc.StandardInput.Close()
    $stdout = $proc.StandardOutput.ReadToEnd()
    $stderr = $proc.StandardError.ReadToEnd()
    $proc.WaitForExit()
    if ($proc.ExitCode -ne 0) {
      throw "attach failed with exit code $($proc.ExitCode)`nSTDOUT:`n$stdout`nSTDERR:`n$stderr"
    }
    return $stdout
  } finally {
    if (-not $proc.HasExited) {
      $proc.Kill()
    }
    $proc.Dispose()
  }
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
$daemonStdout = Join-Path $DataDir "cmdagentd-stdout.log"
$daemonStderr = Join-Path $DataDir "cmdagentd-stderr.log"
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
  Wait-ForDaemon

  Write-Host "Running PTY shell-command smoke test..."
  $shellStart = Invoke-Ctl -Args @(
    "start-shell",
    "--shell", "cmd.exe",
    "--pty",
    "--pty-rows", "30",
    "--pty-cols", "100",
    "--command", "echo PTY-SHELL-SMOKE"
  )
  $shellId = Get-ExecutionId -Output $shellStart.Output
  $shellGet = Wait-ForGetOutput -ExecutionId $shellId -Contains @("Uses PTY: true", "PTY Size: 30x100", "PTY-SHELL-SMOKE")

  Write-Host "Running PTY shell-session attach smoke test..."
  $sessionStart = Invoke-Ctl -Args @(
    "start-session",
    "--shell", "cmd.exe",
    "--pty",
    "--pty-rows", "24",
    "--pty-cols", "80"
  )
  $sessionId = Get-ExecutionId -Output $sessionStart.Output
  $null = Wait-ForGetOutput -ExecutionId $sessionId -Contains @("Uses PTY: true", "PTY Size: 24x80")
  $attachOut = Invoke-Attach -ExecutionId $sessionId -InputLines @(
    "echo PTY-ATTACH-SMOKE",
    "exit"
  )
  if ($attachOut -notmatch "PTY-ATTACH-SMOKE") {
    throw "attach output did not contain PTY-ATTACH-SMOKE`n$attachOut"
  }

  Write-Host "Windows PTY smoke test passed."
  Write-Host "Shell execution id:  $shellId"
  Write-Host "Session execution id: $sessionId"
} finally {
  if ($daemon -and -not $daemon.HasExited) {
    $daemon.Kill()
    $daemon.WaitForExit()
  }
  if ($removeDataDir -and (Test-Path $DataDir)) {
    Remove-Item -Recurse -Force $DataDir
  }
}
