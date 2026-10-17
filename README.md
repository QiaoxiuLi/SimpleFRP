# SimpleFRP

[English](#english) | [简体中文](#简体中文)

## English

### Overview

SimpleFRP lets you access an application on your own computer through a Linux server with a public IP address. Use it to share a local website or reach a TCP service behind a router.

Install both ends, copy one connection string, and connect. The first tunnel is created automatically; manage additional tunnels by their numbers.

- **Quick setup:** no connection details or configuration files to fill in manually.
- **Multiple tunnels:** automatic port allocation, with independent public and local ports.
- **Visible status:** connection state, current transfer speed and total traffic in the terminal; a read-only website on the server.
- **Background operation:** native startup integration and automatic reconnection; closing the status viewer does not stop forwarding.

### Installation and Basic Use

#### Requirements

| Platform | Architecture | Server | Client |
| --- | --- | --- | --- |
| Linux | x86_64 / ARM64 | Yes | Yes |
| Windows 10 / 11 | x64 | No | Yes |
| macOS | Apple Silicon / Intel | No | Yes |

- **Server:** a Linux machine with a reachable public IP, systemd, a DEB or RPM package manager, Bash, curl and root/sudo access. Linux clients need the same tools, but not a public IP.
- **Windows/macOS:** install as the current user, without administrator/root privileges. Windows uses PowerShell; macOS installation needs Bash and curl.
- **Network:** allow the server's heartbeat port and the public ports you use in the firewall/cloud security group. SimpleFRP does not open them for you.

#### Start a Linux Server

```bash
bash <(curl -fsSL https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download/install-linux.sh) server
sudo simplefrp run
```

The server selects three different available ports for the status website, heartbeat connection and first tunnel. It prints these ports and an `SF2` connection string. Keep the string private: anyone holding it can pair a client.

Copy the complete string for the next step. Running `run` again starts the existing installation and shows its current connection details without resetting it.

#### Connect a Client

Replace `CONNECTION_STRING` with the complete string from the server, keeping the quotation marks.

**Linux**

```bash
bash <(curl -fsSL https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download/install-linux.sh) client
sudo simplefrp set "CONNECTION_STRING"
```

**Windows: run in an ordinary PowerShell window**

```powershell
& ([scriptblock]::Create((Invoke-RestMethod 'https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download/install-windows.ps1')))
simplefrp set "CONNECTION_STRING"
```

**macOS**

```bash
bash <(curl -fsSL https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download/install-macos.sh)
```

Open a new terminal, then pair:

```bash
simplefrp set "CONNECTION_STRING"
```

In the original macOS terminal, you can use `~/.local/bin/simplefrp` instead. Windows and macOS open a terminal status window automatically.

#### Forward an Application

Start your application separately. The automatically chosen local port is a placeholder, not an application started by SimpleFRP.

**On Linux, prefix the following management commands with `sudo`.**

```bash
simplefrp state
simplefrp 0 local 3000
```

Use the actual tunnel number shown by `state`; `0` is an example. The tunnel now forwards its server-side public port to `127.0.0.1:3000` on the client.

For an HTTP application, open `http://SERVER_IP:PUBLIC_PORT`, replacing both values with those of your server and tunnel. Allow that public port through the server firewall/security group. Public and local ports do not need to match.

#### Status and Daily Commands

| Command | Purpose |
| --- | --- |
| `simplefrp run` | Start this installation; on the server, also display its connection string |
| `simplefrp next` | Create another automatically allocated tunnel, from the client |
| `simplefrp <id> public <port>` | Change a tunnel's server-side port |
| `simplefrp <id> local <port>` | Change a tunnel's client-side application port |
| `simplefrp delete <id>` | Delete a tunnel |
| `simplefrp state` | Show tunnel numbers, ports, speed, totals and heartbeat port |
| `simplefrp state --watch` | Keep displaying status |
| `simplefrp state --json` | Output status as JSON for scripts and monitoring |

Change a tunnel's public/local ports from either end. Clients manage only their own tunnels; a server can manage all tunnels, including deleting offline ones. Port changes require the client to be online. An occupied-port change fails without replacing the existing mapping.

Speed is shown in `B/s`; totals count application bytes transferred in both directions. Normal restarts preserve mappings and totals. Offline status shows cached totals and zero current speed. `connected` means the control connection is online, not that the target application is running.

### Updates, Offline Installation and Removal

#### Offline Installation

Download the appropriate role, architecture and package from [Releases](https://github.com/QiaoxiuLi/SimpleFRP/releases/latest). Verify its SHA-256 against `checksums.txt`. If the target cannot reach GitHub, download on another device and copy the files across.

| Platform | Installation |
| --- | --- |
| Linux | Download the DEB/RPM package and `install-linux.sh`; use the example below |
| Windows | Extract the x64 ZIP and double-click `install.cmd` |
| macOS | Extract the matching TAR.GZ, then run `sh install.sh` in that folder |

Example: Linux x86_64 server with RPM. For a client, replace `server` with `client` in both places; use the matching `.deb` or ARM64 package when applicable.

```bash
bash install-linux.sh server --package ./simplefrp-server-linux-amd64.rpm
sudo simplefrp run
```

After offline client installation, pair with `simplefrp set "CONNECTION_STRING"`, adding `sudo` on Linux.

#### Updates

Run the installer again to update within v0.2 while preserving configuration. On a Linux server, use `sudo simplefrp run` afterward to start it and view its connection details.

**v0.1 and v0.2 are incompatible.** Back up and uninstall v0.1 before installing v0.2, then pair clients using the new string.

#### Removal

| Platform | Command |
| --- | --- |
| Linux | `sudo simplefrp uninstall` |
| Windows / macOS | `simplefrp uninstall` |

Uninstall stops this installation's processes and removes its startup entries, program, configuration, data and logs. Separately saved backups are retained. Windows removes its executable shortly after the command exits.

An online client also removes its server-side tunnels and identity. If the server is unreachable, local uninstall continues; remove any remaining tunnels later with `delete` on the server.

### Advanced Use

#### Server Address and Ports

If automatic public-IP detection is unavailable, replace `SERVER_IP` with your actual public IP:

```bash
sudo simplefrp run --public-ip SERVER_IP
```

Change the existing website or heartbeat port on the Linux server:

```bash
sudo simplefrp port dashboard 29181
sudo simplefrp port heartbeat 29182
```

These commands do not create tunnels. Online clients reconnect after a heartbeat-port change; offline clients need the new connection string. Allow the new heartbeat port through the firewall/security group.

Local targets support ports `1-65535`. Server public, heartbeat and website ports support `1025-65535`; automatic allocation uses `20000-60000` and avoids common service ports.

#### Read-Only Status Website

The server prints its website address when you run `sudo simplefrp run`. It listens on `127.0.0.1`, requires no website password and cannot change configuration.

To view it remotely, run this on your own computer:

```bash
ssh -L 18080:127.0.0.1:DASHBOARD_PORT USER@SERVER_IP
```

Replace the website port, SSH username and server IP. Keep SSH connected and open `http://127.0.0.1:18080` in your browser. The same forwarded address exposes read-only JSON status at `http://127.0.0.1:18080/api/status`.

#### Deployment and Security

| Platform | Automatic startup | Default runtime locations |
| --- | --- | --- |
| Linux | systemd service: `simplefrp-server` or `simplefrp-client` | Configuration: `/etc/simplefrp`; data: `/var/lib/simplefrp`; logs: `/var/log/simplefrp` |
| Windows | Current-user Startup shortcut | Configuration/data/logs: `%LOCALAPPDATA%\SimpleFRP` |
| macOS | Current-user LaunchAgents | Configuration/data: `~/Library/Application Support/SimpleFRP`; logs: `~/Library/Logs/SimpleFRP` |

- Windows/macOS start **after user login**, not before it. Keep the user logged in and the computer awake for continuous forwarding.
- One default Linux installation has one role: server or client. Each client installation pairs with one server; a server can accept multiple clients.
- Only **TCP** forwarding is supported. SimpleFRP clients and servers must be used together; they are not compatible with upstream `frpc`/`frps`.
- Client-server connections use TLS 1.3 with server-identity verification. A forwarded application's public endpoint still needs its own authentication and HTTPS/TLS where appropriate.
- Treat connection strings and runtime backups as credentials. See the backup guide below to preserve server identity and client connections.

### Troubleshooting and Documentation

| Problem | What to check |
| --- | --- |
| Command not found | Open a new terminal; on macOS, use `~/.local/bin/simplefrp` immediately |
| Cannot connect | Complete connection string, correct public IP, allowed heartbeat port and compatible versions |
| Connected, but the application is unreachable | Target application running at the configured local port; public port allowed |
| Port change fails | New public port unused and client online; the old mapping remains in place |
| macOS shows a finished status window | Close completed SimpleFRP windows; they are not active forwarding processes |

- [Quick start (Chinese)](https://github.com/QiaoxiuLi/SimpleFRP/blob/main/docs/QUICKSTART.md)
- [PDF guide (Chinese)](https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download/SimpleFRP_Project_Guide_v0.2.0.pdf)
- [Backup and recovery (English)](https://github.com/QiaoxiuLi/SimpleFRP/blob/main/BACKUP.md)
- [Third-party components and licenses](https://github.com/QiaoxiuLi/SimpleFRP/blob/main/THIRD_PARTY_NOTICES.md)

Licensed under the [MIT License](https://github.com/QiaoxiuLi/SimpleFRP/blob/main/LICENSE).

---

## 简体中文

### 软件介绍

SimpleFRP 可以借助一台拥有公网 IP 的 Linux 服务器，让你从外部访问自己电脑上的应用。它适合分享本地网站，或连接路由器后方的 TCP 服务。

安装服务端和客户端，复制一串连接字符串，即可自动建立第一条隧道，后续通过隧道编号管理更多转发。

- **快速上手**：无需手动填写连接参数或配置文件。
- **多条隧道**：自动分配端口，公网端口与本地端口可分别设置。
- **状态可见**：终端显示连接状态、当前速度和累计流量，服务端提供只读状态网页。
- **后台运行**：接入系统自启动并自动重连，关闭状态窗口不会停止转发。

### 安装与基本使用

#### 前置条件

| 平台 | 架构 | 服务端 | 客户端 |
| --- | --- | --- | --- |
| Linux | x86_64 / ARM64 | 支持 | 支持 |
| Windows 10 / 11 | x64 | 不支持 | 支持 |
| macOS | Apple Silicon / Intel | 不支持 | 支持 |

- **服务端**：需要公网 IP 可达的 Linux 设备、systemd、DEB 或 RPM 包管理器、Bash、curl，以及 root/sudo 权限。Linux 客户端需要相同工具，但不需要公网 IP。
- **Windows/macOS**：以当前用户安装，无需管理员/root 权限。Windows 使用 PowerShell，macOS 安装需要 Bash 和 curl。
- **网络**：在服务器防火墙或云安全组中放行心跳端口和需要使用的公网转发端口，SimpleFRP 不会自动放行。

#### 启动 Linux 服务端

```bash
bash <(curl -fsSL https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download/install-linux.sh) server
sudo simplefrp run
```

服务端会为状态网页、心跳连接和第一条隧道选择三个不同的可用端口，并输出端口及以 `SF2` 开头的连接字符串。请妥善保管：持有该字符串的人可以配对客户端。

复制完整字符串，供下一步使用。再次执行 `run` 会启动现有安装并显示连接信息，不会重置配置。

#### 连接客户端

将 `CONNECTION_STRING` 替换为服务端输出的完整字符串，保留引号。

**Linux**

```bash
bash <(curl -fsSL https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download/install-linux.sh) client
sudo simplefrp set "CONNECTION_STRING"
```

**Windows：在普通 PowerShell 窗口中执行**

```powershell
& ([scriptblock]::Create((Invoke-RestMethod 'https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download/install-windows.ps1')))
simplefrp set "CONNECTION_STRING"
```

**macOS**

```bash
bash <(curl -fsSL https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download/install-macos.sh)
```

打开新终端后配对：

```bash
simplefrp set "CONNECTION_STRING"
```

在原来的 macOS 终端中，也可以用 `~/.local/bin/simplefrp` 代替命令名。Windows 和 macOS 会自动打开终端状态窗口。

#### 转发本地应用

请自行启动目标应用。自动选择的 local 端口只是占位端口，不代表 SimpleFRP 已启动应用。

**Linux 上执行以下管理命令时，请在前面加 `sudo`。**

```bash
simplefrp state
simplefrp 0 local 3000
```

请使用 `state` 显示的实际隧道编号，`0` 只是示例。修改后，该隧道会将服务端公网端口的流量转发到客户端的 `127.0.0.1:3000`。

如果目标是 HTTP 应用，在浏览器中访问 `http://SERVER_IP:PUBLIC_PORT`，将两个值替换为服务器公网 IP 和隧道的公网端口。请放行该端口的服务器防火墙或安全组规则。public 与 local 端口不必相同。

#### 状态与常用命令

| 命令 | 用途 |
| --- | --- |
| `simplefrp run` | 启动当前安装；服务端同时显示连接字符串 |
| `simplefrp next` | 在客户端新增一条自动分配端口的隧道 |
| `simplefrp <id> public <port>` | 修改隧道在服务端的公网端口 |
| `simplefrp <id> local <port>` | 修改隧道在客户端的应用端口 |
| `simplefrp delete <id>` | 删除隧道 |
| `simplefrp state` | 显示隧道编号、端口、速度、累计流量及心跳端口 |
| `simplefrp state --watch` | 持续显示状态 |
| `simplefrp state --json` | 输出 JSON 状态，供脚本或监控工具使用 |

两端均可修改隧道的 public/local 端口。客户端只能管理自己的隧道，服务端可管理所有隧道，也能删除离线隧道。修改端口需要客户端在线；端口被占用时修改失败，原映射保持不变。

速度单位为 `B/s`，累计流量统计双向转发的应用数据字节数。正常重启会保留映射和累计流量，离线时显示缓存累计值及零速度。`connected` 只表示控制连接在线，不代表目标应用已运行。

### 更新、离线安装与卸载

#### 离线安装

在 [Releases](https://github.com/QiaoxiuLi/SimpleFRP/releases/latest) 下载对应角色、架构和系统的安装包，并根据 `checksums.txt` 核对 SHA-256。目标设备无法访问 GitHub 时，可在其它设备下载后复制过去。

| 平台 | 安装方式 |
| --- | --- |
| Linux | 下载安装包和 `install-linux.sh`，按下面示例安装 DEB/RPM |
| Windows | 解压 x64 ZIP 后双击 `install.cmd` |
| macOS | 解压对应架构的 TAR.GZ，在该目录执行 `sh install.sh` |

以下示例适用于 Linux x86_64 服务端和 RPM 包。安装客户端时，将两处 `server` 改为 `client`；DEB 系统选择对应 `.deb` 文件，ARM64 设备选择对应架构的安装包。

```bash
bash install-linux.sh server --package ./simplefrp-server-linux-amd64.rpm
sudo simplefrp run
```

离线安装客户端后，使用 `simplefrp set "CONNECTION_STRING"` 配对，Linux 上需加 `sudo`。

#### 更新

再次运行安装器即可更新同一 v0.2 系列，并保留配置。Linux 服务端更新后执行 `sudo simplefrp run`，启动服务并查看连接信息。

**v0.1 与 v0.2 不兼容。** 请先备份并卸载 v0.1，再安装 v0.2，并使用新的字符串重新配对客户端。

#### 卸载

| 平台 | 命令 |
| --- | --- |
| Linux | `sudo simplefrp uninstall` |
| Windows / macOS | `simplefrp uninstall` |

卸载会停止本安装的进程，移除自启动项、程序、配置、数据和日志，另行保存的备份不会删除。Windows 的程序文件会在命令退出后短暂延迟删除。

在线客户端卸载时还会解除服务端的隧道和身份。服务端不可达时，本地仍会卸载；之后可在服务端使用 `delete` 清理剩余隧道。

### 进阶使用

#### 服务端地址与端口

无法自动识别公网 IP 时，将 `SERVER_IP` 替换为实际公网 IP：

```bash
sudo simplefrp run --public-ip SERVER_IP
```

在 Linux 服务端修改已有网页或心跳端口：

```bash
sudo simplefrp port dashboard 29181
sudo simplefrp port heartbeat 29182
```

这两个命令不会新增隧道。修改心跳端口后，在线客户端会自动重连；离线客户端需要使用新的连接字符串。请放行新心跳端口的防火墙或安全组规则。

本地目标端口支持 `1-65535`，服务端公网、心跳和网页端口支持 `1025-65535`；自动分配范围为 `20000-60000`，并避开常用服务端口。

#### 只读状态网页

执行 `sudo simplefrp run` 后可看到服务端网页地址。网页监听 `127.0.0.1`，无需设置网页密码，也不能修改配置。

远程查看时，在自己的电脑上执行：

```bash
ssh -L 18080:127.0.0.1:DASHBOARD_PORT USER@SERVER_IP
```

替换网页端口、SSH 用户名和服务器 IP。保持 SSH 连接，在浏览器中打开 `http://127.0.0.1:18080`。同一转发地址还提供只读 JSON 状态：`http://127.0.0.1:18080/api/status`。

#### 部署与安全

| 平台 | 自动启动方式 | 默认运行目录 |
| --- | --- | --- |
| Linux | systemd 服务：`simplefrp-server` 或 `simplefrp-client` | 配置：`/etc/simplefrp`；数据：`/var/lib/simplefrp`；日志：`/var/log/simplefrp` |
| Windows | 当前用户的启动文件夹快捷方式 | 配置、数据及日志：`%LOCALAPPDATA%\SimpleFRP` |
| macOS | 当前用户的 LaunchAgents | 配置及数据：`~/Library/Application Support/SimpleFRP`；日志：`~/Library/Logs/SimpleFRP` |

- Windows/macOS 在**用户登录后**自动启动，不是登录前的系统服务。持续转发时需保持用户登录，并避免电脑休眠。
- 一个 Linux 默认安装只能选择服务端或客户端角色。每个客户端安装连接一台服务端，服务端可接入多个客户端。
- 仅支持 **TCP** 转发。两端均需使用 SimpleFRP，不能直接与上游 `frpc`/`frps` 配对。
- 客户端与服务端使用 TLS 1.3 连接并验证服务器身份。被转发应用的公网入口仍需按需配置自身的身份验证和 HTTPS/TLS。
- 连接字符串和运行备份应作为凭据保管。保留服务器身份及客户端连接的方法见下方备份指南。

### 常见问题与使用资料

| 问题 | 检查方法 |
| --- | --- |
| 找不到命令 | 打开新终端；macOS 当前终端可直接使用 `~/.local/bin/simplefrp` |
| 无法连接 | 检查连接字符串是否完整、公网 IP 是否正确、心跳端口是否放行、版本是否兼容 |
| 已连接但无法访问应用 | 检查目标应用是否运行在指定本地端口，以及公网端口是否放行 |
| 修改端口失败 | 确认新公网端口空闲、客户端在线；原映射会保留 |
| macOS 显示已结束的状态窗口 | 可关闭已结束的 SimpleFRP 窗口，它们不是正在转发的进程 |

- [快速使用指南（中文）](https://github.com/QiaoxiuLi/SimpleFRP/blob/main/docs/QUICKSTART.md)
- [PDF 使用手册（中文）](https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download/SimpleFRP_Project_Guide_v0.2.0.pdf)
- [备份与恢复（英文）](https://github.com/QiaoxiuLi/SimpleFRP/blob/main/BACKUP.md)
- [第三方组件与许可证](https://github.com/QiaoxiuLi/SimpleFRP/blob/main/THIRD_PARTY_NOTICES.md)

SimpleFRP 采用 [MIT License](https://github.com/QiaoxiuLi/SimpleFRP/blob/main/LICENSE)。
