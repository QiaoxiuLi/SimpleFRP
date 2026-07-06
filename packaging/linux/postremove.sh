#!/usr/bin/env sh
set -eu
rm -f /etc/systemd/system/simplefrp-server.service /etc/systemd/system/simplefrp-client.service
rm -f /usr/lib/systemd/system/simplefrp-server.service /usr/lib/systemd/system/simplefrp-client.service
rm -f /lib/systemd/system/simplefrp-server.service /lib/systemd/system/simplefrp-client.service
rm -rf /etc/simplefrp /var/lib/simplefrp /var/log/simplefrp
systemctl daemon-reload 2>/dev/null || true
