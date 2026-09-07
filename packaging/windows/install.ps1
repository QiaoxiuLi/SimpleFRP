param([switch]$NoConfigure,[int]$LocalPort=0,[string]$Server="")
$ErrorActionPreference = "Stop"
$source = Join-Path $PSScriptRoot 'simplefrp.exe'
if (!(Test-Path $source)) { $source = Join-Path $PSScriptRoot 'simplefrp-client-windows-amd64.exe' }
if (!(Test-Path $source)) { throw 'Place install.ps1 beside the SimpleFRP executable.' }
$destination = Join-Path $env:LOCALAPPDATA 'Programs\SimpleFRP'
New-Item -ItemType Directory -Force -Path $destination | Out-Null
$binary = Join-Path $destination 'simplefrp.exe'
if (Test-Path $binary) {
    $backup = Join-Path $env:LOCALAPPDATA ('SimpleFRP\backups\'+(Get-Date -Format 'yyyyMMdd-HHmmss'))
    New-Item -ItemType Directory -Force -Path $backup | Out-Null
    Copy-Item $binary (Join-Path $backup 'simplefrp.exe')
    # Only stop the daemon from this exact installation, so unrelated services stay running.
    Get-CimInstance Win32_Process | Where-Object { $_.ExecutablePath -eq $binary -and $_.CommandLine -match 'daemon --role client' } | ForEach-Object { Stop-Process -Id $_.ProcessId }
}
Copy-Item -LiteralPath $source -Destination $binary -Force
Write-Host "Installed: $binary"
if (!$NoConfigure) {
    $arguments = @('setup','--role','client')
    if ($LocalPort -gt 0) { $arguments += @('--local-port',[string]$LocalPort) }
    if ($Server) { $arguments += @('--server',$Server) }
    & $binary @arguments
    if ($LASTEXITCODE -ne 0) { throw 'Configuration failed. Run the installed executable with setup to retry.' }
}
Write-Host 'User installation complete. Login startup is enabled after successful setup; keep the user logged in.'
