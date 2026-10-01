#!/bin/sh
set -eu
# Reject v1 before a package manager replaces its files or executes old hooks.
if [ -x /usr/bin/simplefrp ]; then
 case "$(/usr/bin/simplefrp --version)" in
  'simplefrp version 0.2.'*)
   role=$(cat /etc/simplefrp/role)
   /usr/bin/simplefrp install --role "$role" --prepare-upgrade
   ;;
  *) echo 'Legacy SimpleFRP was preserved. Back it up and uninstall it before installing v2.' >&2; exit 1 ;;
 esac
elif [ -e /etc/simplefrp/server.toml ] || [ -e /etc/simplefrp/client.toml ]; then
 echo 'Existing unverified SimpleFRP configuration was preserved.' >&2; exit 1
fi
