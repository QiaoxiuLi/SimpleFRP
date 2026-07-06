$ErrorActionPreference = "SilentlyContinue"
schtasks /Delete /TN "SimpleFRP" /F
Remove-Item -Recurse -Force "$env:ProgramData\SimpleFRP"
