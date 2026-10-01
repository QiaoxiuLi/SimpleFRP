#!/usr/bin/env bash
set -euo pipefail
role=${1:-server}
[[ $# -eq 0 ]] || shift
[[ "$role" == server || "$role" == client ]] || { echo 'Usage: linux.sh server|client [--package FILE]' >&2; exit 2; }
package=''
while [[ $# -gt 0 ]]; do
 case "$1" in --package) package=${2:?Missing package}; shift 2 ;; *) echo 'Unknown option.' >&2; exit 2 ;; esac
done
[[ $(uname -s) == Linux ]] || { echo 'Linux is required.' >&2; exit 1; }
case $(uname -m) in x86_64) arch=amd64 ;; aarch64|arm64) arch=arm64 ;; *) echo 'Unsupported CPU architecture.' >&2; exit 1 ;; esac
if command -v dpkg >/dev/null; then format=deb; elif command -v rpm >/dev/null; then format=rpm; else echo 'A DEB or RPM package manager is required.' >&2; exit 1; fi
privilege=()
if [[ $EUID -ne 0 ]]; then sudo -v; privilege=(sudo); fi
work=''
trap '[[ -z "$work" ]] || rm -rf -- "$work"' EXIT
if [[ -z "$package" ]]; then
 work=$(mktemp -d)
 base='https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download'
 name="simplefrp-$role-linux-$arch.$format"
 curl -fL --retry 2 --connect-timeout 15 "$base/$name" -o "$work/$name"
 curl -fL --retry 2 --connect-timeout 15 "$base/checksums.txt" -o "$work/checksums.txt"
 (cd "$work"; awk -v file="$name" '$2==file {print}' checksums.txt > selected.sha256; test -s selected.sha256; sha256sum --check selected.sha256)
 package="$work/$name"
fi
[[ -f "$package" && "$package" == *."$format" ]] || { echo 'Package format does not match this system.' >&2; exit 1; }
if [[ "$format" == deb ]]; then actual=$(dpkg-deb -f "$package" Package); else actual=$(rpm -qp --qf '%{NAME}' "$package"); fi
[[ "$actual" == "simplefrp-$role" ]] || { echo 'Package role does not match.' >&2; exit 1; }
if [[ "$format" == deb ]]; then "${privilege[@]}" dpkg -i "$package"; else "${privilege[@]}" rpm -U --replacepkgs "$package"; fi
echo 'Installation complete. Server: sudo simplefrp run; client: sudo simplefrp set <connection-string>'
