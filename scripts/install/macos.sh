#!/bin/sh
set -eu
[ "$(uname -s)" = Darwin ] || { echo 'macOS is required.' >&2; exit 1; }
case $(uname -m) in arm64) arch=arm64 ;; x86_64) arch=amd64 ;; *) echo 'Unsupported CPU architecture.' >&2; exit 1 ;; esac
work=$(mktemp -d)
trap 'rm -rf -- "$work"' EXIT
base='https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download'
name="simplefrp-client-darwin-$arch.tar.gz"
curl -fL --retry 2 --connect-timeout 15 "$base/$name" -o "$work/$name"
curl -fL --retry 2 --connect-timeout 15 "$base/checksums.txt" -o "$work/checksums.txt"
(cd "$work"; awk -v file="$name" '$2==file {print}' checksums.txt > selected.sha256; test -s selected.sha256; shasum -a 256 --check selected.sha256)
tar -xzf "$work/$name" -C "$work"
sh "$work/install.sh" "$@"
