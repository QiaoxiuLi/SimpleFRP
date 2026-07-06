#!/usr/bin/env sh
set -eu
mkdir -p dist
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o dist/simplefrp-client-darwin-arm64 ./cmd/simplefrp
tar -czf dist/simplefrp-client-darwin-arm64.tar.gz -C dist simplefrp-client-darwin-arm64
