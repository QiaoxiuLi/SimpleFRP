# SimpleFRP

Lightweight TCP reverse forwarding for Linux, Windows and macOS. Install, copy
one connection string, and connect. Ports are negotiated automatically; use
tunnel IDs to change mappings. There is no configuration wizard or web admin.

## Install And Connect

The Linux server needs systemd and root/sudo. Linux clients also use systemd.
Windows and macOS clients install for the current user without elevation.

**Linux server, two commands:**

```bash
bash <(curl -fsSL https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download/install-linux.sh) server
sudo simplefrp run
```

`run` discovers the public IP, selects three different available uncommon ports
and prints an `SF2` connection string. Save it privately. Repeating `run` preserves
the identity, configuration and ports; it does not generate another server.

**Linux client:**

```bash
bash <(curl -fsSL https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download/install-linux.sh) client
sudo simplefrp set <connection-string>
```

**macOS client, Apple Silicon or Intel:**

```bash
bash <(curl -fsSL https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download/install-macos.sh)
simplefrp set <connection-string>
```

Use a new terminal after installation so its PATH includes the command. The
absolute command is `~/.local/bin/simplefrp` when needed immediately.

**Windows 10/11 x64 client, ordinary PowerShell:**

```powershell
& ([scriptblock]::Create((Invoke-RestMethod 'https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download/install-windows.ps1')))
simplefrp set <connection-string>
```

Replace `<connection-string>` with the entire string printed by the server;
do not type the angle brackets. The first tunnel is created automatically.
Its local port is initially an available placeholder: an application is not
started by SimpleFRP. Point the tunnel at your existing application's port:

```bash
simplefrp state
simplefrp 0 local 3000
```

Use the actual tunnel ID shown by `state`, not necessarily `0`. Local and public
ports do not have to match. A local service already listening on 3000 is valid.

Downloads, offline package installation and troubleshooting are explained in
[the Chinese quick start](docs/QUICKSTART.md). The release also includes a PDF
project and operations guide.

## Commands

| Command | Operation |
| --- | --- |
| `simplefrp run` | Start; initialize a Linux server once and print its connection string |
| `simplefrp set <connection-string>` | Pair a client; repeated pairing with the same server does not add a tunnel |
| `simplefrp state` | Show tunnel ID, public/local ports, live B/s, total bytes and heartbeat port |
| `simplefrp state --json` | Machine-readable live status, or explicitly marked offline cached status |
| `simplefrp state --watch` | Live terminal status; one status viewer per installation |
| `simplefrp next` | Create one new automatically allocated tunnel, from the client |
| `simplefrp <id> public <port>` | Change that tunnel's Linux-server port, from either peer |
| `simplefrp <id> local <port>` | Change that tunnel's client-side local target, from either peer |
| `simplefrp delete <id>` | Delete a tunnel; server administrators can also delete offline mappings |
| `simplefrp port dashboard <port>` | Change the existing Linux status-web port |
| `simplefrp port heartbeat <port>` | Change the existing Linux communication port |
| `simplefrp uninstall` | Stop and remove this installation and its owned startup/runtime files |

`port` does not add a port or a tunnel. `next` adds a tunnel; tunnel IDs identify
the mapping whose `public` or `local` port is being changed. Server-wide tunnel
IDs are unique. Clients can only manage their own tunnels.

Use `sudo` for the Linux system installation's management commands. Public,
dashboard and heartbeat listeners must use available ports from 1025 to 65535.
Automatic selection uses 20000-60000 and avoids common service ports. Explicit
changes can use common nonprivileged ports if they are free. Local targets can
use 1-65535, including an existing service on 80, 443 or 3389.

Changing heartbeat ports notifies online clients and retains the old listener
for 30 seconds while they reconnect. Offline clients need the updated connection
string. Changing service ports never creates an extra tunnel. Open the new
heartbeat/public ports in your firewall or cloud security group; SimpleFRP does
not modify unrelated network rules. Occupied-port changes fail without replacing
the old mapping. Port changes require an online client to acknowledge and save
its configuration; deleting an offline tunnel from the server does not.

## Status And Runtime

Only Linux supports the server. Windows/macOS/Linux support clients. Windows
and macOS open a terminal status window automatically. Its viewer is separate
from the forwarding daemon: closing the viewer does not stop forwarding.

The server has a **read-only, loopback-only** status website, at the dashboard
port printed by `run`. It displays the same ports, rate and totals as `state`.
There are no login, configuration, password or write controls. GET/HEAD are
allowed; write methods return HTTP 405. There are no historical charts or SSE.

For remote viewing, forward only the dashboard through SSH:

```bash
ssh -L 18080:127.0.0.1:DASHBOARD_PORT user@SERVER_IP
```

Then open `http://127.0.0.1:18080` locally. Do not expose the dashboard with
Nginx or change other sites. `connected` indicates a live control connection,
not that the target application is running. Check `127.0.0.1:<local_port>` on
the client if forwarding fails.

Rates and totals count payload bytes copied in both directions, not TCP/TLS
overhead. Rates update about once a second. Mappings, client identity and total
traffic are saved to bbolt and survive normal restart. Client offline status
retains known totals and shows zero current speed; it is not claimed to be live.
Automatic reconnection preserves existing tunnel IDs and ports.

## How It Works

1. Server initialization generates its identity, invitation key, administrator
   key and an Ed25519 certificate. Standard Go TLS 1.3 encrypts control and data
   connections. The server reserves the first public listener.
2. The shared `SF2` encoder packs protocol version, IPv4/IPv6 address, heartbeat
   port, server identity, invitation key and certificate fingerprint into a
   canonical Base62 string with a corruption checksum. It contains letters and
   digits and is approximately 95 characters for IPv4.
3. Clients pin the exact SHA-256 certificate fingerprint before pairing, then
   receive an independent client ID/key and the first globally numbered tunnel.
   The invitation key is not reused as every client's runtime identity.
4. Control requests use HMAC-SHA256 over the complete canonical JSON message,
   with timestamps and random nonces. Replay records, clients, tunnels and
   pending connections are bounded.
5. When traffic arrives on a public port, the server requests a separate client
   data stream. The client connects to its own `127.0.0.1:<local_port>`. A
   bidirectional TCP bridge preserves data and TCP half-close semantics.
6. The server is authoritative for mapping changes. It binds new public ports
   before releasing old ports and waits for client configuration-save ACKs.
   Failed changes roll back. Shutdown waits for owned workers to exit before
   closing storage.

The string is **encoded, not encrypted**. Anyone who has it can pair a client.
Do not put it in public logs, screenshots, issue reports, URLs or source control.
The server prints it deliberately for copying. Treat it like a credential.
TLS protects the SimpleFRP link; the public application endpoint is still plain
TCP unless the forwarded application itself uses TLS, such as HTTPS.

This is an FRP-style workflow, not wire compatibility with upstream `frpc` or
`frps`. The v2 transport uses Go's standard TLS/crypto/net libraries. The unused
upstream FRP version-only dependency and old password/Argon2 helpers were removed.
There is no `pwd`, `setup`, `set server`, old-password or old-protocol mode.

## Installation, Updates And Cleanup

| Platform | Binary / Configuration | Startup |
| --- | --- | --- |
| Linux | `/usr/bin/simplefrp`; `/etc/simplefrp`; `/var/lib/simplefrp`; `/var/log/simplefrp` | `simplefrp-server.service` or `simplefrp-client.service` |
| macOS | `~/.local/bin/simplefrp`; `~/Library/Application Support/SimpleFRP`; `~/Library/Logs/SimpleFRP` | current-user daemon and status LaunchAgents |
| Windows | `%LOCALAPPDATA%\Programs\SimpleFRP\simplefrp.exe`; `%LOCALAPPDATA%\SimpleFRP` | current-user Startup `SimpleFRP.lnk` |

Linux uses an unprivileged `simplefrp` service account. The client/server packages
conflict because one default Linux installation has one role. Windows/macOS use
login startup, not an unattended pre-login system service. Keep the user logged
in and the machine awake. Autostart entries are configured by installation;
reboot/login acceptance is not part of this release's requested test scope.

Windows adds only its own directory to the user PATH. macOS adds a marked PATH
block only if needed. Uninstall removes only changes recorded as owned, leaving
other PATH entries, profile content and startup items intact. Process cleanup
uses executable/start-time/command identity, never a broad process-name kill.
Owned service accounts are removed only if created by this installation.

Run `sudo simplefrp uninstall` on Linux or `simplefrp uninstall` on clients.
Windows deletes its executable shortly after the uninstaller exits. Runtime
files, owned startup entries and processes are removed. An online client also
releases its server-side tunnels and identity; an unreachable server is warned
about, while local cleanup proceeds. Backups kept separately by the user are
not deleted. Linux package-manager removal follows the same cleanup path.

For v2 updates, run the installer again. Verified v2 processes are stopped
before replacement; existing configuration is reused. v0.1 is incompatible:
installers refuse to overwrite it. Back up and explicitly uninstall v0.1 before
installing v2; clients must pair again using the new connection string.

## Dependencies And Build

| Runtime module | Version | Purpose | License |
| --- | --- | --- | --- |
| [Cobra](https://github.com/spf13/cobra) | 1.8.1 | CLI command/argument parsing | Apache-2.0 |
| [go-toml/v2](https://github.com/pelletier/go-toml) | 2.2.2 | Structured TOML encoding/decoding and secure configuration writes | MIT |
| [bbolt](https://github.com/etcd-io/bbolt) | 1.3.11 | Embedded durable mapping, identity and traffic storage; process locks | MIT |
| [x/sys](https://go.googlesource.com/sys) | 0.44.0 | Platform system calls and Windows console mode | BSD-3-Clause |

Indirect runtime modules include Cobra's `pflag` and Windows `mousetrap`.
`testify`, `go-spew`, `go-difflib` and `x/sync` are retained for dependency tests,
not added application services. Exact versions are in `go.mod`/`go.sum`; bundled
licenses are in `THIRD_PARTY_NOTICES.md`.

Go 1.25+ is the source minimum; release builds use Go 1.27.1. GoReleaser v2/nFPM
produce Linux DEB/RPM, Windows ZIP and macOS TAR.GZ packages. systemd, launchd,
PowerShell 5.1, WScript.Shell and built-in terminals provide native integration.
The application needs no Python, Node.js, Docker, external database, Nginx or
browser framework. Standard Go packages provide TLS, Ed25519, HMAC, randomness,
TCP, JSON, HTTP and embedded web assets. ReportLab/Poppler/Noto Sans SC are documentation-only
build tools for the PDF, not runtime dependencies.

```bash
git clone https://github.com/QiaoxiuLi/SimpleFRP.git
cd SimpleFRP
go test -race ./...
go vet ./...
make build
make release
```

Packages are in `dist/` and include SHA-256 checksums. Supported build targets
are Linux amd64/arm64, Windows amd64 and macOS amd64/arm64. Actual native test
evidence is recorded separately from cross-compilation in
`docs/RELEASE_ACCEPTANCE.md` and `test/integration/README.md`.

## Backup And Troubleshooting

[Backup and recovery](BACKUP.md) covers GitHub, local Git bundles, private runtime
backups and restoring a stopped installation. Never publish runtime secrets or
live bbolt files. Source archives alone do not contain a deployed server's keys.

Connection failures: check the heartbeat endpoint, public IP, firewall/security
group, pinned server identity and whether both peers are v2. A local port already
in use by the intended application is correct; a public listener in use by another
process is not. Port-change failures preserve the old mapping. A reserved local
placeholder does not mean an application is serving there. Servers without a
working public-IP discovery route can use `sudo simplefrp run --public-ip IP`.
