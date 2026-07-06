#!/usr/bin/env sh
set -eu
mkdir -p dist
goreleaser release --snapshot --clean
