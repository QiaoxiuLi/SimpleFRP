# SimpleFRP

SimpleFRP is a lightweight cross-platform TCP reverse forwarding tool for quick deployment. It provides an FRP-style server/client workflow with a small command line interface, automatic tunnel creation, package-based installation, autostart setup, dashboard visibility, and clean uninstall behavior.

## What It Does

SimpleFRP is designed for users who need to expose a TCP service behind NAT or a firewall without writing FRP configuration files by hand.

Typical flow:

1. Install the server package on a public Linux server.
2. Run `sudo simplefrp pwd` once to set the server password.
3. Install the client package on the machine that owns the local TCP service.
4. Run `simplefrp set server` once to connect to the server.
5. SimpleFRP creates the first tunnel automatically.
6. Use `simplefrp state` to view local/public ports.

Default server ports:

- Dashboard: `8387`
- Control/heartbeat: `8388`
- Public tunnel ports: automatically allocated from `20000` to `60000`

## Linux Installation From GitHub

Download packages from:

```text
https://github.com/QiaoxiuLi/SimpleFRP/releases/latest
```

### Ubuntu 22.04 Or Newer

Server:

```bash
wget https://github.com/QiaoxiuLi/SimpleFRP/releases/download/v0.1.0/simplefrp-server-linux-amd64.deb
sudo dpkg -i simplefrp-server-linux-amd64.deb
sudo simplefrp pwd
```

Client:

```bash
wget https://github.com/QiaoxiuLi/SimpleFRP/releases/download/v0.1.0/simplefrp-client-linux-amd64.deb
sudo dpkg -i simplefrp-client-linux-amd64.deb
simplefrp set server
```

### Rocky Linux 9, RHEL 9, AL2023, AliLinux, HCE

Server:

```bash
curl -LO https://github.com/QiaoxiuLi/SimpleFRP/releases/download/v0.1.0/simplefrp-server-linux-amd64.rpm
sudo rpm -i simplefrp-server-linux-amd64.rpm
sudo simplefrp pwd
```

Client:

```bash
curl -LO https://github.com/QiaoxiuLi/SimpleFRP/releases/download/v0.1.0/simplefrp-client-linux-amd64.rpm
sudo rpm -i simplefrp-client-linux-amd64.rpm
simplefrp set server
```

Open firewall/security-group ports on the public server:

- `8388/tcp` for client control traffic
- `8387/tcp` for dashboard access
- Any allocated public tunnel port, for example `35001/tcp`

## Client Setup

Run:

```bash
simplefrp set server
```

Enter the server address:

```text
SERVER_PUBLIC_IP:8388
```

Enter the server password created by:

```bash
sudo simplefrp pwd
```

The client saves a derived password key, creates the first tunnel, and starts or restarts the client daemon when a service manager is available.

## Commands

Set or reset the server password:

```bash
sudo simplefrp pwd
```

Configure the client:

```bash
simplefrp set server
```

View tunnels:

```bash
simplefrp state
```

Create another tunnel:

```bash
simplefrp next
```

Change a local port:

```bash
simplefrp 0 local 25001
```

Change a public port:

```bash
simplefrp 0 public 35001
```

Delete a tunnel:

```bash
simplefrp delete 0
```

Uninstall and clean runtime files:

```bash
simplefrp uninstall
```

## Dashboard

The server dashboard is available at:

```text
http://SERVER_PUBLIC_IP:8387
```

It shows:

- Active tunnel count
- Traffic totals
- Traffic range data
- Connection logs
- Tunnel records

The dashboard uses embedded HTML, CSS, JavaScript, standard `net/http`, and Server-Sent Events. It does not require Node.js, nginx, React, Vue, Angular, or a CDN.

## Implementation

### Server

The server daemon runs with:

```bash
simplefrp daemon --role server
```

It performs these tasks:

- Listens on `8388/tcp` for authenticated client control messages.
- Starts the dashboard on `8387/tcp`.
- Allocates available public tunnel ports.
- Listens on public tunnel ports.
- Requests the connected client to open a data stream when an external user connects.
- Bridges public TCP traffic to the client data connection.
- Stores traffic, tunnel, and connection data in bbolt.

### Client

The client daemon runs with:

```bash
simplefrp daemon --role client
```

It performs these tasks:

- Maintains a control connection to the server.
- Registers configured tunnels with the server.
- Opens a data connection when the server requests a new stream.
- Connects that data stream to `127.0.0.1:<local_port>`.
- Restarts cleanly after `simplefrp set server`.

### Tunnel Lifecycle

When a tunnel is created:

1. The client picks or receives a local port.
2. The server allocates or reserves a public port.
3. The server starts listening on the public port.
4. External traffic arrives on the public port.
5. The server asks the client to open a data connection.
6. The client connects to the local service.
7. The server and client copy TCP bytes in both directions.

### Authentication

SimpleFRP never stores the plaintext password.

- Server password is stored as an Argon2id hash.
- Client stores derived password material, not the original password.
- Control requests use HMAC-SHA256 signatures.
- The derived client secret is not sent over the network.

### Storage

Runtime statistics use bbolt:

- `traffic`: uploaded/downloaded byte counters
- `connections`: connection logs
- `tunnels`: active tunnel records

Default runtime paths:

- Linux config: `/etc/simplefrp`
- Linux data: `/var/lib/simplefrp`
- Linux logs: `/var/log/simplefrp`
- macOS config/data: `/Library/Application Support/SimpleFRP`
- macOS logs: `/Library/Logs/SimpleFRP`
- Windows config/data/logs: `%ProgramData%\SimpleFRP`

## Packaging And Autostart

Linux packages create:

- `simplefrp` system user and group
- `/etc/simplefrp`
- `/var/lib/simplefrp`
- `/var/log/simplefrp`
- `/etc/simplefrp/role`
- `simplefrp-server.service` or `simplefrp-client.service`
- systemd enablement

The service is enabled after install, but the server waits for `sudo simplefrp pwd`, and the client waits for `simplefrp set server`.

macOS uses launchd:

- `/Library/LaunchDaemons/com.simplefrp.client.plist`

Windows uses a login task:

- Scheduled task name: `SimpleFRP`

## Clean Uninstall

`simplefrp uninstall` stops services, disables autostart, kills remaining SimpleFRP processes except the current uninstall process, and removes runtime files.

Linux cleanup includes:

- `systemctl stop simplefrp-server` or `simplefrp-client`
- `systemctl disable simplefrp-server` or `simplefrp-client`
- removal of systemd unit files
- `systemctl daemon-reload`
- removal of `/etc/simplefrp`, `/var/lib/simplefrp`, `/var/log/simplefrp`

macOS cleanup includes:

- `launchctl unload`
- removal of `com.simplefrp.client.plist`
- removal of application support and log directories

Windows cleanup includes:

- removal of the `SimpleFRP` scheduled task
- stopping `simplefrp.exe`
- removal of `%ProgramData%\SimpleFRP`

## Third-Party Libraries And Tools

Runtime Go dependencies:

- `github.com/fatedier/frp v0.62.1`: upstream FRP dependency and integration boundary. SimpleFRP keeps a lightweight wrapper while retaining an explicit dependency on the mature FRP project.
- `github.com/spf13/cobra v1.8.1`: command line framework for `pwd`, `state`, `next`, `delete`, `set server`, `uninstall`, and `daemon`.
- `github.com/spf13/viper v1.19.0`: TOML configuration loading and decoding.
- `go.etcd.io/bbolt v1.3.11`: embedded key/value storage for tunnel, connection, and traffic data.
- `golang.org/x/crypto v0.37.0`: Argon2id password hashing and XChaCha20-Poly1305 helpers.

Build and release tools:

- Go 1.23 or newer
- GoReleaser v2 for multi-platform release builds
- nFPM through GoReleaser for `.deb` and `.rpm` packages
- PowerShell for optional Windows zip packaging script
- systemd on Linux
- launchd on macOS
- Windows Task Scheduler on Windows

Standard library packages used heavily:

- `net`, `net/http`: TCP listeners, TCP dialing, dashboard server
- `encoding/json`: control message encoding
- `crypto/hmac`, `crypto/sha256`: request signing
- `os`, `os/exec`, `runtime`: platform-specific service and filesystem operations
- `embed`: embedded dashboard assets

## Build From Source

```bash
git clone https://github.com/QiaoxiuLi/SimpleFRP.git
cd SimpleFRP
make test
make release
```

Release artifacts are written to:

```text
dist/
```

## Backup And Recovery

Source code is backed up in GitHub:

```bash
git clone https://github.com/QiaoxiuLi/SimpleFRP.git
```

Runtime configuration is not committed to git because it contains local state and secrets. Back up production runtime files separately:

```bash
sudo tar -czf simplefrp-runtime-backup.tar.gz /etc/simplefrp /var/lib/simplefrp
```

Restore:

```bash
sudo tar -xzf simplefrp-runtime-backup.tar.gz -C /
sudo systemctl restart simplefrp-server || sudo systemctl restart simplefrp-client
```

## Troubleshooting

`SERVER_UNREACHABLE`: Check the server address, service status, firewall, security group, and `8388/tcp`.

`AUTH_FAILED`: Run `simplefrp set server` again and enter the correct password.

`LOCAL_PORT_IN_USE`: Choose another local port.

`SERVER_PORT_IN_USE`: Choose another public port or free it on the server.

`INVALID_PASSWORD_FORMAT`: Use at least 8 letters or numbers. Do not use spaces, symbols, or Chinese characters.
