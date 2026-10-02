$ErrorActionPreference = 'Stop'

if (-not ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole('Administrator')) {
    throw 'Run this from an Administrator PowerShell.'
}

$backend = Join-Path $PSScriptRoot '..\backend' | Resolve-Path
$dest = 'C:\Armory'
$task = 'Armory Server'

New-Item -ItemType Directory -Force $dest | Out-Null
Stop-ScheduledTask -TaskName $task -ErrorAction SilentlyContinue
Get-Process -Name armory-server -ErrorAction SilentlyContinue | Stop-Process -Force

Push-Location $backend
go build -o "$dest\armory-server.exe" ./cmd/server
Pop-Location

if (-not (Test-Path "$dest\certs")) { Copy-Item "$backend\certs" "$dest\certs" -Recurse }
if (-not (Test-Path "$dest\armory.db")) { Copy-Item "$backend\armory.db" "$dest\armory.db" }

[Environment]::SetEnvironmentVariable('ARMORY_DB', "$dest\armory.db", 'Machine')
[Environment]::SetEnvironmentVariable('ARMORY_CERTS', "$dest\certs", 'Machine')

Remove-NetFirewallRule -DisplayName 'Armory*' -ErrorAction SilentlyContinue
New-NetFirewallRule -DisplayName 'Armory sensor UDP' -Direction Inbound -Protocol UDP -LocalPort 47810 -Action Allow | Out-Null
New-NetFirewallRule -DisplayName 'Armory web' -Direction Inbound -Protocol TCP -LocalPort 8080,8443 -Action Allow | Out-Null

$action = New-ScheduledTaskAction -Execute "$dest\armory-server.exe" -WorkingDirectory $dest
$trigger = New-ScheduledTaskTrigger -AtStartup
$principal = New-ScheduledTaskPrincipal -UserId 'SYSTEM' -LogonType ServiceAccount -RunLevel Highest
$settings = New-ScheduledTaskSettingsSet -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1) -ExecutionTimeLimit ([TimeSpan]::Zero) -StartWhenAvailable -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
Register-ScheduledTask -TaskName $task -Action $action -Trigger $trigger -Principal $principal -Settings $settings -Force | Out-Null
Start-ScheduledTask -TaskName $task

Write-Host "Armory installed in $dest and set to start at boot. Admin: https://192.168.4.200:8443/admin/login"
