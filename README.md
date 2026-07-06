# SimpleFRP

SimpleFRP is a lightweight cross-platform TCP forwarding tool built in Go. It wraps mature FRP-style reverse proxy behavior behind a small command line interface so ordinary users do not need to edit FRP configuration files or choose tunnel ports during first setup.

## Default Server Ports

- Dashboard: `8387`
- Heartbeat/control: `8388`

## Install Server

Install the `simplefrp-server` `.rpm` or `.deb` package. The post-install script creates the `simplefrp` user, configuration directory, data directory, log directory, role file, systemd service, and autostart entry. The service is enabled but waits for the first password setup before it starts cleanly.

After installation, run:

```bash
sudo simplefrp pwd
```

The password must be at least 8 characters and contain only letters and numbers. SimpleFRP stores an Argon2id hash, never the plaintext password.
This command saves the server configuration and starts or restarts `simplefrp-server`.

## Install Client

Install the package for your platform:

- Ubuntu 22.04 or newer: `.deb`
- Rocky Linux 9, RHEL 9, AL2023, AliLinux 3/4, Huawei HCE: `.rpm`
- Windows Server/Home/Pro: `.zip` or `.msi`
- macOS Apple Silicon: `.tar.gz` or `.pkg`

Linux and macOS use terminal configuration. Windows opens a CMD console and creates a login task instead of using a graphical UI.

## First Client Setup

```bash
simplefrp set server
```

Enter the server address, such as `1.2.3.4:8388` or `example.com:8388`, then enter the password. The client creates the first TCP tunnel automatically with random available local and public ports.
This command saves the client configuration and starts or restarts `simplefrp-client` when a service manager is available.

## Commands

```bash
simplefrp pwd
simplefrp state
simplefrp next
simplefrp 0 local 25001
simplefrp 0 public 35001
simplefrp delete 0
simplefrp set server
simplefrp uninstall
```

`simplefrp state` only displays `success` or `failed` for tunnel status.

## Dashboard

The server dashboard is available at:

```text
http://SERVER_PUBLIC_IP:8387
```

It shows client count, active tunnel count, total traffic, traffic ranges, and connection logs. The page uses embedded HTML, CSS, JavaScript, and SSE. It does not require Node.js, nginx, React, Vue, Angular, or an external CDN.

## Common Errors

`SERVER_UNREACHABLE`: Check the server address, server service status, firewall, and port `8388`.

`AUTH_FAILED`: Run `simplefrp set server` and enter the correct password.

`LOCAL_PORT_IN_USE`: Choose another local port.

`SERVER_PORT_IN_USE`: Choose another public port or free the port on the server.

`INVALID_PASSWORD_FORMAT`: Use at least 8 letters or numbers. Do not use spaces, symbols, or Chinese characters.

## Uninstall

```bash
simplefrp uninstall
```

Server uninstall stops and disables `simplefrp-server`, releases ports and processes, removes service files, and deletes configuration, data, and logs.

Client uninstall stops the client, deletes tunnels, removes autostart entries, releases local ports, and deletes configuration, data, and logs.

## Build

```bash
make build
make test
make package-linux
make package-windows
make package-macos
make release
```

`make release` writes artifacts to `dist/`.

## Security

- Passwords are never stored in plaintext.
- Client password material is stored as derived key material, not the original password.
- Control requests use HMAC-SHA256 signatures, so the derived client secret is not sent over the network.
- Dashboard APIs are read-only.
- Public port mutation is reserved for authenticated client control traffic.
- Logs must not contain plaintext passwords or session keys.

## Automatic Configuration

Server packages automatically create:

- SimpleFRP system user
- `/etc/simplefrp/`
- `/var/lib/simplefrp/`
- `/var/log/simplefrp/`
- `/etc/simplefrp/role`
- `/etc/simplefrp/server.toml`
- `/var/lib/simplefrp/simplefrp.db`
- `simplefrp-server.service`
- systemd autostart and restart policy

Client packages automatically create the role file, config directory, data directory, log directory, and platform autostart entry.

## Backup and Recovery

The source repository is the primary backup. After cloning on a new machine:

```bash
git clone https://github.com/QiaoxiuLi/SimpleFRP.git
cd SimpleFRP
make test
make release
```

For day-to-day development:

```bash
git pull --rebase
git status
git add .
git commit -m "Describe the change"
git push
```

Runtime configuration is intentionally not stored in git. Back up production config separately:

```bash
sudo tar -czf simplefrp-runtime-backup.tar.gz /etc/simplefrp /var/lib/simplefrp
```
