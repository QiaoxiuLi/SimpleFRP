#!/bin/sh
set -eu
# User installation: no administrator password or unrelated system changes.
source_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
binary="$source_dir/simplefrp"
[ -f "$binary" ] || binary="$source_dir/simplefrp-client-darwin-arm64"
[ -f "$binary" ] || { echo 'Place this installer beside the SimpleFRP binary.' >&2; exit 1; }
install_dir="$HOME/.local/bin"
mkdir -p "$install_dir"
if [ -f "$install_dir/simplefrp" ]; then
 backup_dir="$HOME/Library/Application Support/SimpleFRP/backups/$(date +%Y%m%d-%H%M%S)"
 mkdir -p "$backup_dir"
 cp -p "$install_dir/simplefrp" "$backup_dir/simplefrp"
fi
install -m 755 "$binary" "$install_dir/simplefrp"
printf 'Installed: %s\nConfigure and start: "%s" setup\n' "$install_dir/simplefrp" "$install_dir/simplefrp"
printf 'The client starts at login after configuration. No PATH or system files were changed.\n'
if [ "${1:-}" != "--no-configure" ]; then
 "$install_dir/simplefrp" setup --role client "$@"
fi
