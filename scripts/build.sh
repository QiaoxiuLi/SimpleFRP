#!/usr/bin/env sh
set -eu
mkdir -p dist
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o dist/simplefrp ./cmd/simplefrp
