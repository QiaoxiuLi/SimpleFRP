#!/bin/sh
set -eu
[ "$(id -u)" -ne 0 ] || { echo 'Install as the logged-in user, not root.' >&2; exit 1; }
source_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
binary="$source_dir/simplefrp"
[ -f "$binary" ] || binary="$source_dir/simplefrp-client-darwin-$(uname -m)"
[ -f "$binary" ] || { echo 'Place this installer beside the SimpleFRP executable.' >&2; exit 1; }
destination="$HOME/.local/bin/simplefrp"
if [ -f "$destination" ]; then
 case "$("$destination" --version)" in
  'simplefrp version 0.2.'*) "$destination" install --role client --prepare-upgrade ;;
  *) echo 'Legacy installation preserved. Back it up and uninstall it before installing v2.' >&2; exit 1 ;;
 esac
fi
mkdir -p "$(dirname "$destination")"
stage="$destination.next.$$"
trap 'rm -f -- "$stage"' EXIT
install -m 755 "$binary" "$stage"
mv -f -- "$stage" "$destination"
"$destination" install --role client "$@"
printf 'Installed for this user. In a new terminal: simplefrp set <connection-string>\n'
