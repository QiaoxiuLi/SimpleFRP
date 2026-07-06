#!/usr/bin/env sh
set -eu
role="$(cat /etc/simplefrp/role 2>/dev/null || printf client)"
getent group simplefrp >/dev/null 2>&1 || groupadd --system simplefrp
id simplefrp >/dev/null 2>&1 || useradd --system --gid simplefrp --home-dir /var/lib/simplefrp --shell /usr/sbin/nologin simplefrp
install -d -m 0750 -o simplefrp -g simplefrp /etc/simplefrp /var/lib/simplefrp /var/log/simplefrp
printf '%s\n' "$role" >/etc/simplefrp/role
chown simplefrp:simplefrp /etc/simplefrp/role
chmod 0640 /etc/simplefrp/role
systemctl daemon-reload
systemctl enable "simplefrp-$role"
if [ "$role" = "server" ]; then
  printf 'SimpleFRP Server Port: 8388\nSimpleFRP Dashboard Port: 8387\n'
  printf 'Run "sudo simplefrp pwd" to set the server password and start SimpleFRP.\n'
else
  printf 'Run "simplefrp set server" to connect this client and start SimpleFRP.\n'
fi
