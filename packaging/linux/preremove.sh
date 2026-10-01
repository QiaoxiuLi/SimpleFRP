#!/bin/sh
set -eu
case "${1:-remove}" in 0|remove|purge) ;; *) exit 0 ;; esac
if [ -f /etc/simplefrp/installation.json ]; then
 SIMPLEFRP_PACKAGE_REMOVING=1 /usr/bin/simplefrp uninstall
fi
