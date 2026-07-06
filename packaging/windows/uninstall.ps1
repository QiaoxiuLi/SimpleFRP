$ErrorActionPreference = "SilentlyContinue"
schtasks /Delete /TN "SimpleFRP" /F
Get-Process simplefrp | Where-Object { $_.Id -ne $PID } | Stop-Process -Force
Remove-Item -Recurse -Force "$env:ProgramData\SimpleFRP"
