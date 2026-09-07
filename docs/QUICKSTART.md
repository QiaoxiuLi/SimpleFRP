# SimpleFRP 快速安装

v0.1.1 的服务端和客户端需要一起升级。密码在向导中隐藏输入，不放进命令行、URL 或下载地址。

## Linux 服务端（x86_64，DEB/RPM）

有 sudo 权限的普通账户或 root 执行一行：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/QiaoxiuLi/SimpleFRP/main/scripts/install/linux.sh) server
```

安装器自动选择 DEB/RPM、核对下载的 SHA-256、备份已有 SimpleFRP 文件、安装 systemd 服务；首次安装会进入密码向导并启动。需要管理员权限时会调用 sudo。控制端口默认 8388，管理面板默认仅本机 127.0.0.1:8387。按向导输出在云防火墙中放行控制端口和实际使用的公网端口；安装器不修改其他服务和网络规则。

Linux 客户端将末尾 `server` 换成 `client`。向导会询问服务器地址、密码、本地服务端口，并建立第一条映射。通过包管理器安装后，也可仅执行 `sudo simplefrp setup --role server` 或 `sudo simplefrp setup --role client`。

## 中国内地服务器 / 离线安装

在网络方便的电脑下载同一 Release 的 `install-linux.sh`、对应 DEB/RPM 和 `checksums.txt`，校验后经可信 SSH 上传。服务器不用连接 GitHub：

```bash
sha256sum --ignore-missing -c checksums.txt
bash install-linux.sh server --package ./simplefrp-server-linux-amd64.rpm
```

Ubuntu/Debian 替换为 `.deb`。客户端把两处 `server` 改为 `client`。下载校验失败时不要继续安装。升级 v0.1.0 请使用此安装器，它会避开旧版卸载钩子在升级时误删配置的问题；已有配置会保留，不会重新询问密码。

## macOS（Apple Silicon）

无需 sudo，一行完成下载、校验、用户级安装和向导：

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/QiaoxiuLi/SimpleFRP/main/scripts/install/macos.sh)
```

已经解压发布包时，执行 `sh install.sh` 即可。已知本地网站运行在 3000 时，可以执行 `sh install.sh --local-port 3000`，向导只需填写服务器和密码。

安装位置为 `~/.local/bin/simplefrp`，配置为 `~/Library/Application Support/SimpleFRP`，登录启动项为 `~/Library/LaunchAgents/com.simplefrp.client.plist`。安装器不修改系统 PATH。完成配置后自动启动；Mac 需要保持登录、开机和唤醒状态。

## Windows（x64）

在普通 PowerShell 中执行一行；不要求管理员，不永久修改执行策略：

```powershell
& ([scriptblock]::Create((Invoke-RestMethod 'https://raw.githubusercontent.com/QiaoxiuLi/SimpleFRP/main/scripts/install/windows.ps1')))
```

也可下载 ZIP 并解压，双击 `install.cmd`，或在该目录运行 `./install.cmd`。该脚本只为本次 PowerShell 子进程设置执行策略，以运行包内的安装程序。

程序安装到 `%LOCALAPPDATA%\Programs\SimpleFRP`，新配置保存到 `%LOCALAPPDATA%\SimpleFRP`。当前用户 Startup 文件夹中建立启动快捷方式，向导完成后直接启动代理。旧版 `%ProgramData%\SimpleFRP\client.toml` 如果存在则保留其配置位置，升级该旧安装可能仍需要原有权限。新用户安装不会接管旧的系统级计划任务。

## 常用配置

向导可重复运行，密码仍通过交互输入：

```text
simplefrp setup --role client --server SERVER:8388 --local-port 3000
simplefrp 0 local 3000
simplefrp 0 public 35001
simplefrp state
```

Mac 使用完整命令 `~/.local/bin/simplefrp`；Windows 使用安装目录中的 `simplefrp.exe`。本地端口是转发目标，服务已经占用该端口是正常情况。配置修改约两秒内生效，无需手动重启。公网端口须在服务端 `port_min`/`port_max` 范围内且未被其他服务占用。

服务器若只允许一个指定端口，可以在首次向导中指定：

```bash
sudo simplefrp setup --role server --control-port 7000 --public-port 9290
```

此命令会配置服务密码和 SimpleFRP 端口，不替你修改云防火墙或 DNS。管理面板没有登录认证，因此默认只监听回环地址；需要远程管理可通过 SSH 本地转发访问。SimpleFRP 的 HMAC 用于认证，转发内容本身不加密，敏感应用应使用 HTTPS/WSS 等应用层加密。

## 自动启动的边界

Linux 使用 systemd，随系统启动。Mac 使用当前用户的 LaunchAgent，Windows 使用当前用户的登录启动项；后两者需要用户登录，并非系统账户服务。Windows/macOS 机器休眠期间无法转发。

在线安装脚本只在安装时联网下载。日常转发不会检查或下载 GitHub 更新。没有测速或压力测试功能自动运行。
