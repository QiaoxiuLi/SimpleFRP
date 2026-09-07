$ErrorActionPreference = "Stop"
New-Item -ItemType Directory -Force -Path dist | Out-Null
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
go build -trimpath -ldflags="-s -w" -o dist/simplefrp-client-windows-amd64.exe ./cmd/simplefrp
Copy-Item packaging/windows/install.ps1,packaging/windows/install.cmd,docs/QUICKSTART.md -Destination dist
Compress-Archive -Force -Path dist/simplefrp-client-windows-amd64.exe,dist/install.ps1,dist/install.cmd,dist/QUICKSTART.md -DestinationPath dist/simplefrp-client-windows-amd64.zip
