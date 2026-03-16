param(
  [string]$Binary = (Join-Path $PSScriptRoot "..\cmdagentd.exe"),
  [string]$Config = (Join-Path $PSScriptRoot "..\cmdagentd.json"),
  [string]$Name = "cmdagentd-smoke"
)

& $Binary service install --name $Name --binary $Binary --config $Config
& $Binary service start --name $Name
& $Binary service status --name $Name
& $Binary service stop --name $Name
& $Binary service uninstall --name $Name
