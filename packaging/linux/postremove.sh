#!/bin/sh
set -eu
case "${1:-remove}" in 0|remove|purge) ;; *) exit 0 ;; esac
# Files belonging to the package are removed by the package manager itself.
systemctl daemon-reload
