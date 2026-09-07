#!/bin/sh
set -eu
[ "$(uname -s)" = Darwin ] && [ "$(uname -m)" = arm64 ] || { echo 'This release supports Apple Silicon Macs.' >&2; exit 1; }
work=$(mktemp -d)
trap 'rm -rf -- "$work"' EXIT
base='https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download'
name='simplefrp-client-darwin-arm64.tar.gz'
curl --fail --location --retry 2 --connect-timeout 15 "$base/$name" -o "$work/$name"
curl --fail --location --retry 2 --connect-timeout 15 "$base/checksums.txt" -o "$work/checksums.txt"
(cd "$work"; awk -v file="$name" '$2==file {print}' checksums.txt > selected.sha256; test -s selected.sha256; shasum -a 256 --check selected.sha256)
tar -xzf "$work/$name" -C "$work"
sh "$work/install.sh" "$@"
