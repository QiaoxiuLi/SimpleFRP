# SimpleFRP

SimpleFRP 是一个面向快速部署的轻量 TCP 内网穿透工具。它将客户端本地的服务转发到 Linux 服务器的公网端口，让你可以从外部访问本机或内网中的应用。

安装服务端和客户端，复制一串连接字符串，即可建立第一条隧道。无需手动填写服务器地址、通信端口或连接密钥，后续通过隧道编号管理转发。

## 功能与平台

- **快速连接**：服务端自动识别公网 IP，客户端使用一串连接字符串完成配对。
- **自动分配端口**：首次初始化自动选择状态网页、心跳连接和第一条转发隧道的端口；新增隧道也会自动协商端口。
- **按编号管理隧道**：查看、新增、删除隧道，或从服务端、客户端修改现有隧道的公网端口与本地目标端口。
- **实时查看状态**：显示连接状态、端口、当前转发速度和累计流量；Linux 服务端提供只读状态网页。
- **安装后自动运行**：Linux 配置系统服务；Windows、macOS 配置当前用户登录后启动，并自动打开终端状态窗口。
- **一条命令卸载**：停止转发，移除本安装的自启动配置、程序和运行文件。

| 平台 | 架构 | 服务端 | 客户端 |
| --- | --- | --- | --- |
| Linux | x86_64、ARM64 | 支持 | 支持 |
| Windows 10 / 11 | x64 | 不支持 | 支持 |
| macOS | Apple Silicon、Intel | 不支持 | 支持 |

Linux 需要 systemd、DEB 或 RPM 包管理器及 root/sudo 权限。Windows 和 macOS 使用当前用户安装，无需管理员权限；持续转发时需保持用户登录，并避免设备进入睡眠。

SimpleFRP 支持 TCP 转发，不包含 UDP 转发。服务端与客户端均需使用 SimpleFRP，不能直接与上游 `frpc`、`frps` 配对。

## 1. 安装服务端

在有公网 IP 的 Linux 服务器上执行：

```bash
bash <(curl -fsSL https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download/install-linux.sh) server
sudo simplefrp run
```

启动后，终端会显示心跳端口、状态网页地址及以 `SF2` 开头的连接字符串。复制完整字符串，稍后在客户端使用。再次执行 `run` 会沿用现有配置，不会重复创建服务器身份。

在服务器防火墙或云安全组中放行 **心跳端口** 和需要使用的 **public 转发端口**。自动分配的空闲端口不一定已被云安全组放行；状态网页只监听本机，不需要开放到公网。

如果无法自动识别公网 IP，可以手动指定：

```bash
sudo simplefrp run --public-ip 你的公网IP
```

连接字符串包含配对凭据，请只交给需要连接的客户端，不要公开分享。

## 2. 安装并连接客户端

将下面命令中的 `连接字符串` 替换为服务端输出的完整内容，保留引号。

### Linux

```bash
bash <(curl -fsSL https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download/install-linux.sh) client
sudo simplefrp set "连接字符串"
```

一个 Linux 默认安装只能选择服务端或客户端其中一种角色。

### Windows

在普通 PowerShell 窗口中执行：

```powershell
& ([scriptblock]::Create((Invoke-RestMethod 'https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download/install-windows.ps1')))
simplefrp set "连接字符串"
```

### macOS

```bash
bash <(curl -fsSL https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download/install-macos.sh)
simplefrp set "连接字符串"
```

macOS 安装后请打开一个新终端，再执行配对命令。也可以在当前终端直接使用 `~/.local/bin/simplefrp set "连接字符串"`。

连接成功后会自动创建第一条隧道。Windows 和 macOS 会自动打开状态终端窗口；关闭这个窗口不会停止后台转发。

## 3. 接入本地应用

SimpleFRP 不会替你启动目标应用。首次自动分配的 local 端口是占位端口，需要将它改为应用实际监听的端口。

Linux 的管理命令需要在前面加 `sudo`；下面的通用命令示例在 Windows 和 macOS 上可以直接执行。

先查看隧道编号：

```bash
simplefrp state
```

以下示例假设隧道编号为 `0`，客户端上的应用监听 `3000` 端口：

```bash
simplefrp 0 local 3000
simplefrp 0 public 35001
```

此时，服务器公网 `35001` 端口的流量会转发到客户端的 `127.0.0.1:3000`。如果目标是 HTTP 服务，可以在浏览器中访问 `http://服务器公网IP:35001`。

请使用 `state` 显示的实际隧道编号，不一定是 `0`。public 与 local 端口无需相同，local 端口已被目标应用监听是正常情况。

## 4. 管理隧道

| 命令 | 用途 |
| --- | --- |
| `simplefrp next` | 从客户端新增一条自动分配端口的隧道 |
| `simplefrp <编号> public <端口>` | 修改该隧道在服务器上的公网端口 |
| `simplefrp <编号> local <端口>` | 修改该隧道在客户端上的本地目标端口 |
| `simplefrp delete <编号>` | 删除指定隧道 |
| `simplefrp state` | 查看隧道和连接状态 |
| `simplefrp state --watch` | 持续显示状态 |
| `simplefrp state --json` | 以 JSON 格式输出状态 |

新增隧道使用 `next`，不是新增一个独立端口。修改已有隧道的 public/local 端口，可在客户端或服务端执行；客户端只能管理自己的隧道。

public 端口需在服务器上空闲且已被防火墙放行。端口被占用或客户端离线时，修改不会生效，原映射会保留。服务端可以删除离线客户端的隧道。

local 端口范围为 `1-65535`；服务器的 public、心跳和状态网页端口范围为 `1025-65535`。

### 修改服务端端口

在 Linux 服务端执行：

```bash
sudo simplefrp port dashboard 29181
sudo simplefrp port heartbeat 29182
```

`dashboard` 修改状态网页端口，`heartbeat` 修改客户端与服务端的通信端口。这两个命令修改已有服务端端口，不会新增隧道。

修改心跳端口后，在线客户端会自动重连；离线客户端需要使用新的连接字符串重新执行 `set`。请同时放行新的心跳端口。

## 5. 查看状态

`simplefrp state` 显示各隧道的编号、public/local 端口、当前转发速度、累计流量和连接状态，心跳端口单独显示。

速度单位为 `B/s`，累计流量单位为字节，统计客户端与服务端双向转发的应用数据。断线时显示离线状态、已知累计流量及零速度；正常重启后保留隧道映射和累计流量。

`connected` 表示客户端与服务端已建立连接，不代表本地目标应用一定正在运行。

### 服务端状态网页

Linux 服务端的状态网页地址由 `run` 输出。网页无需登录，只用于查看状态，不能修改配置。

在服务器本机可以直接打开输出的地址。远程查看时，在自己的电脑上建立 SSH 端口转发：

```bash
ssh -L 18080:127.0.0.1:DASHBOARD_PORT user@SERVER_IP
```

将 `DASHBOARD_PORT`、`user`、`SERVER_IP` 替换为实际网页端口、SSH 用户名和服务器 IP。保持 SSH 连接，在本机浏览器打开 `http://127.0.0.1:18080`。

## 6. 离线安装、更新与卸载

### 离线安装

在 [Releases](https://github.com/QiaoxiuLi/SimpleFRP/releases/latest) 下载对应系统、架构和角色的安装包。Linux 提供 DEB/RPM，Windows 提供 ZIP，macOS 提供 TAR.GZ；`checksums.txt` 提供文件的 SHA-256 校验值。

- **Linux**：将安装包和 `install-linux.sh` 放在一起，使用下面的命令安装。客户端将 `server` 及包名中的 `server` 改为 `client`，DEB 系统选择对应的 `.deb` 文件。
- **Windows**：解压 ZIP 后双击 `install.cmd`。
- **macOS**：解压 TAR.GZ 后，在解压目录执行 `sh install.sh`。

```bash
bash install-linux.sh server --package ./simplefrp-server-linux-amd64.rpm
sudo simplefrp run
```

无法连接 GitHub 下载服务时，可以在能访问 GitHub 的设备上下载文件，再复制到目标设备进行离线安装。

### 更新

再次运行对应平台的安装命令或安装器即可更新，同一 v0.2 系列会保留现有连接配置。

v0.1 与 v0.2 不兼容。从 v0.1 升级时，请先备份并卸载旧版，再安装新版，使用新的连接字符串重新配对。

### 卸载

Linux：

```bash
sudo simplefrp uninstall
```

Windows、macOS：

```bash
simplefrp uninstall
```

卸载会停止本安装的转发进程，移除它的自启动项、程序、配置、运行数据和日志。Windows 的程序文件会在卸载命令退出后短暂延迟删除；另行保存的备份不会被删除。

在线客户端卸载时，也会解除它在服务端的隧道和身份。如果服务端不可达，本地仍会卸载；可以随后在服务端使用 `state` 和 `delete` 清理剩余隧道。

## 常见问题

| 问题 | 检查方法 |
| --- | --- |
| 找不到 `simplefrp` 命令 | 重新打开终端；macOS 可使用 `~/.local/bin/simplefrp` |
| 无法配对 | 确认连接字符串完整、服务器公网 IP 正确、心跳端口已放行，且两端均使用 v0.2 |
| 显示 connected，但访问失败 | 检查目标应用是否运行、是否能从客户端的 `127.0.0.1:local端口` 访问，以及 public 端口是否放行 |
| 修改 public 端口失败 | 确认新端口未被其它程序或隧道占用，且客户端在线 |
| 修改心跳端口后离线客户端无法连接 | 从服务端取得新的连接字符串，重新执行 `simplefrp set` |
| 服务器身份校验失败 | 确认服务器身份或配置是否被重置，并从目标服务器重新取得连接字符串 |

客户端与服务端之间的连接使用 TLS 加密。公网应用端点是否加密由目标应用决定，例如转发 HTTPS 服务时仍由该服务提供 HTTPS。

## 使用资料

- [快速使用指南](docs/QUICKSTART.md)
- [PDF 使用手册](docs/SimpleFRP_Project_Guide_v0.2.0.pdf)
- [备份与恢复](BACKUP.md)：保留连接身份、配置和运行数据
- [第三方组件与许可证](THIRD_PARTY_NOTICES.md)

SimpleFRP 采用 [MIT License](LICENSE)。
