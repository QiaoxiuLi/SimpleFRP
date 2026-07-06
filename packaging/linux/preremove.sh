#!/usr/bin/env sh
set -eu
role="$(cat /etc/simplefrp/role 2>/dev/null || printf client)"
systemctl stop "simplefrp-$role" 2>/dev/null || true
systemctl disable "simplefrp-$role" 2>/dev/null || true
