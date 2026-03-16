param(
  [string]$Binary = (Join-Path $PSScriptRoot "..\cmdrad.exe"),
  [string]$Config = (Join-Path $PSScriptRoot "..\dev/cmdrad.json"),
  [string]$Name = "cmdrad-smoke"
)

& $Binary service install --name $Name --binary $Binary --config $Config
& $Binary service start --name $Name
& $Binary service status --name $Name
& $Binary service stop --name $Name
& $Binary service uninstall --name $Name
