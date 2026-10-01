param([switch]$NoStart)
$ErrorActionPreference = 'Stop'
$source = Join-Path $PSScriptRoot 'simplefrp.exe'
if (!(Test-Path -LiteralPath $source)) { $source = Join-Path $PSScriptRoot 'simplefrp-client-windows-amd64.exe' }
if (!(Test-Path -LiteralPath $source)) { throw 'Place install.ps1 beside the SimpleFRP executable.' }
$destination = Join-Path $env:LOCALAPPDATA 'Programs\SimpleFRP'
$binary = Join-Path $destination 'simplefrp.exe'
if (Test-Path -LiteralPath $binary) {
    $version = & $binary --version
    if ($LASTEXITCODE -ne 0 -or $version -notmatch '^simplefrp version 0\.2\.') { throw 'Legacy installation preserved. Back it up and uninstall it before installing v2.' }
    & $binary install --role client --prepare-upgrade
    if ($LASTEXITCODE -ne 0) { throw 'Existing installation was not replaced.' }
}
New-Item -ItemType Directory -Force -Path $destination | Out-Null
$stage = Join-Path $destination ('simplefrp-'+[Guid]::NewGuid().ToString('N')+'.exe')
try {
    Copy-Item -LiteralPath $source -Destination $stage
    Move-Item -LiteralPath $stage -Destination $binary -Force
} finally { if (Test-Path -LiteralPath $stage) { Remove-Item -LiteralPath $stage } }
$arguments = @('install','--role','client')
if ($NoStart) { $arguments += '--no-start' }
& $binary @arguments
if ($LASTEXITCODE -ne 0) { throw 'Installation did not complete. The error above is authoritative.' }
$directory = Split-Path -Parent $binary
if (($env:Path -split ';') -notcontains $directory) { $env:Path += ';'+$directory }
Write-Host 'Installed for the current user. Use: simplefrp set <connection-string>'
