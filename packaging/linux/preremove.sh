#!/usr/bin/env sh
set -eu
# RPM passes 1/2 during upgrades; dpkg passes upgrade/failed-upgrade.
# Only a real removal may stop services or remove runtime state.
case "${1:-remove}" in
  0|remove|purge) ;;
  *) exit 0 ;;
esac
role="$(cat /etc/simplefrp/role 2>/dev/null || printf client)"
systemctl stop "simplefrp-$role" 2>/dev/null || true
systemctl disable "simplefrp-$role" 2>/dev/null || true
