$ErrorActionPreference = "Stop"
New-Item -ItemType Directory -Force -Path dist | Out-Null
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
go build -trimpath -ldflags="-s -w" -o dist/simplefrp-client-windows-amd64.exe ./cmd/simplefrp
Compress-Archive -Force -Path dist/simplefrp-client-windows-amd64.exe -DestinationPath dist/simplefrp-client-windows-amd64.zip
