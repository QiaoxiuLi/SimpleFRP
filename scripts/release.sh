#!/usr/bin/env sh
set -eu
./scripts/build.sh
./scripts/package-linux.sh
./scripts/package-macos.sh
if command -v pwsh >/dev/null 2>&1; then
  pwsh ./scripts/package-windows.ps1
fi
