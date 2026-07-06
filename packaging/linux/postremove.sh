#!/usr/bin/env sh
set -eu
rm -rf /etc/simplefrp /var/lib/simplefrp /var/log/simplefrp
systemctl daemon-reload 2>/dev/null || true
