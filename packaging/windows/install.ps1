$ErrorActionPreference = "Stop"
$base = "$env:ProgramData\SimpleFRP"
New-Item -ItemType Directory -Force -Path $base, "$base\data", "$base\logs" | Out-Null
"client" | Set-Content -Path "$base\role"
schtasks /Create /TN "SimpleFRP" /SC ONLOGON /TR "cmd /k simplefrp daemon --role client" /F
Start-Process cmd -ArgumentList "/k simplefrp set server"
