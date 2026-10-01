#!/bin/sh
set -eu
role=$(cat /etc/simplefrp/role)
/usr/bin/simplefrp install --role "$role" --no-start
if [ "$role" = client ] || [ -f /etc/simplefrp/server.toml ]; then
 systemctl restart "simplefrp-$role"
fi
printf 'Installed SimpleFRP v2 (%s).\n' "$role"
if [ "$role" = server ]; then
 printf 'Initialize and print the connection string: sudo simplefrp run\n'
else
 printf 'Pair: sudo simplefrp set <connection-string>\n'
fi
