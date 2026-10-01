# SimpleFRP Backup And Recovery

## Source And Release Recovery

GitHub stores committed source and release packages, not deployed credentials.

```bash
git clone https://github.com/QiaoxiuLi/SimpleFRP.git
cd SimpleFRP
git switch --detach v0.2.0
go test ./...
```

For normal development, inspect `git status` before `git pull --ff-only`.
Commit only intended source/doc changes, then `git push origin main`. Do not
force-push or overwrite local work to resolve a pull conflict.

A separate local/offline backup can preserve all Git history and tags:

```bash
git bundle create /PRIVATE_BACKUP_DIR/SimpleFRP.bundle --all
git bundle verify /PRIVATE_BACKUP_DIR/SimpleFRP.bundle
git clone /PRIVATE_BACKUP_DIR/SimpleFRP.bundle SimpleFRP-recovered
```

Restore its GitHub remote with `git remote set-url origin
https://github.com/QiaoxiuLi/SimpleFRP.git` and fetch normally. Release ZIP,
TAR.GZ, DEB and RPM packages can be downloaded again from GitHub Releases.
Verify SHA-256 against `checksums.txt` before installation.

## Private Runtime Backups

Server certificates, private keys, invitations, client keys and bbolt state
must never enter Git, a public release, screenshots or issue logs. Use a private
directory with mode 700 and backup files with mode 600; encrypt offsite backups
using your existing backup system. These instructions do not change any other
application's backup policy.

Stop **only** the applicable SimpleFRP service before archiving a bbolt database:

```bash
sudo systemctl stop simplefrp-server
sudo sh -c 'umask 077; tar -czf /PRIVATE_BACKUP_DIR/simplefrp-runtime.tar.gz /etc/simplefrp /var/lib/simplefrp'
sudo systemctl start simplefrp-server
```

Use `simplefrp-client` for a Linux client. A normal stop/start is not an uninstall
and does not remove configuration. Do not archive a live database as though it
were a guaranteed consistent snapshot.

On macOS, stop the two SimpleFRP user LaunchAgents and its own monitor before
archiving `~/Library/Application Support/SimpleFRP`. On Windows, stop the exact
daemon/viewer recorded in this installation before copying `%LOCALAPPDATA%\SimpleFRP`.
Do not kill processes by name or terminate Terminal/Windows Terminal itself.
Keep receipt/startup metadata with the backup. Do not restore PID/lock files as
evidence of a currently running process.

## Restore

Restore into a stopped installation of the **same major protocol version**.
Inspect the archive contents and paths before extraction. Keep a backup of the
current configuration instead of overwriting it blindly. Restore the configuration
and database, omit stale `*.pid.json`/`*.lock`/cached status files, and ensure the
service user can read its files. Then run the matching installer to restore
startup metadata and start the matching SimpleFRP service.

Restoring the server certificate, keys and state preserves existing clients'
identity pin and tunnel IDs. Restoring only source/binaries does not. Clients
need a new connection string if the server identity was reinitialized.

The v0.1-to-v0.2 transition is not a state migration. Keep the v0.1 binary/package,
service definition and private runtime snapshot together for rollback; v0.1
clients cannot use a v0.2 server. A v0.2 reinstall should use its existing config
rather than generating a fresh identity unless that is intentional.
