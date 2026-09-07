#!/usr/bin/env sh
set -eu
# RPM passes 1/2 during upgrades; dpkg passes upgrade/failed-upgrade.
# Only a real removal may stop services or remove runtime state.
case "${1:-remove}" in
  0|remove|purge) ;;
  *) exit 0 ;;
esac
rm -f /etc/systemd/system/simplefrp-server.service /etc/systemd/system/simplefrp-client.service
rm -f /usr/lib/systemd/system/simplefrp-server.service /usr/lib/systemd/system/simplefrp-client.service
rm -f /lib/systemd/system/simplefrp-server.service /lib/systemd/system/simplefrp-client.service
rm -rf /etc/simplefrp /var/lib/simplefrp /var/log/simplefrp
systemctl daemon-reload 2>/dev/null || true
