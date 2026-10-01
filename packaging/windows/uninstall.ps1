$ErrorActionPreference = 'Stop'
$binary = Join-Path $env:LOCALAPPDATA 'Programs\SimpleFRP\simplefrp.exe'
if (!(Test-Path -LiteralPath $binary)) { throw 'SimpleFRP is not installed for this user.' }
& $binary uninstall
if ($LASTEXITCODE -ne 0) { throw 'Uninstall was not completed. No unrelated process was stopped.' }
