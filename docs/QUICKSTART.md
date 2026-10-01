# SimpleFRP v0.2 快速使用指南

## 1. 安装 Linux 服务端

在有公网 IP、使用 systemd 的 Linux 服务器上执行：

```bash
bash <(curl -fsSL https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download/install-linux.sh) server
sudo simplefrp run
```

第一条安装并配置开机自启，第二条完成初始化和启动，输出连接字符串。
首次自动选择三个不同、非常用且可绑定的端口，分别用于网页、心跳和第一条隧道。
重复执行 run 保留现有配置和身份，不重复初始化。
公网 IP 自动识别失败时使用 `sudo simplefrp run --public-ip 你的公网IP`。

在云防火墙/安全组中允许输出的心跳端口和 public 隧道端口；无需开放网页端口。
软件不会替你修改其它防火墙规则、Nginx 或网站。
连接字符串不是加密保险箱，它含有配对凭据，不能公开或提交到 Git。

## 2. 安装并连接客户端

Linux 客户端：

```bash
bash <(curl -fsSL https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download/install-linux.sh) client
sudo simplefrp set <connection-string>
```

macOS 客户端，支持 Apple Silicon 和 Intel：

```bash
bash <(curl -fsSL https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download/install-macos.sh)
simplefrp set <connection-string>
```

安装后开一个新终端；当前终端也可以直接使用 `~/.local/bin/simplefrp`。

Windows 10/11 x64 客户端，在普通 PowerShell 中执行：

```powershell
& ([scriptblock]::Create((Invoke-RestMethod 'https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download/install-windows.ps1')))
simplefrp set <connection-string>
```

用服务端输出的整串字符替换 `<connection-string>`，不要保留尖括号。
双方自动建立第一条隧道。Windows/macOS 自动弹出状态终端，关闭它不会停止转发。
首次自动分配的 local 端口是目标占位端口，软件不会启动你的应用。

## 3. 对接已有应用

```bash
simplefrp state
simplefrp 0 local 3000
```

使用 state 显示的实际隧道编号；不保证总是 0。上述命令将客户端本机 3000 端口作为目标。
访问服务端的公网 IP 和该隧道的 public 端口，即访问客户端应用。
local 与 public 不必一致，local 已被你的应用监听是正确的，不是冲突。
如果应用没有监听 `127.0.0.1:local端口`，控制连接在线也无法转发应用数据。

Linux 默认系统安装的管理命令需要 sudo。local 可为 1-65535；服务端监听端口为 1025-65535。
自动选择会避开常用端口；手动指定可使用空闲的常用非特权端口。

## 4. 隧道操作

```bash
simplefrp next
simplefrp 0 public 35001
simplefrp 0 local 3000
simplefrp delete 0
```

next 新建一条隧道，不是新增一个独立端口。public/local 命令修改编号对应的现有隧道，客户端和服务端都可执行。
端口冲突、客户端无法保存或离线时，修改失败并保留原映射。
客户端只能改自己的隧道；服务端可以删除离线隧道。

服务端网页和心跳端口也可修改，但这不创建隧道：

```bash
sudo simplefrp port dashboard 29181
sudo simplefrp port heartbeat 29182
```

在线客户端会接收新心跳端口并重连，旧端口保留约 30 秒过渡。
离线客户端需要复制新的连接字符串，再执行 set。请为新的心跳/public 端口配置相应的网络放行规则。

## 5. 状态与网页

```bash
simplefrp state
simplefrp state --json
simplefrp state --watch
```

显示隧道编号、public/local 端口、实时 B/s 和累计字节数；心跳端口单独显示。
流量是双向应用数据之和，不包含 TCP/TLS 开销。断线时显示离线和缓存累计值，速度为零，不伪装成实时数据。
服务端重启后保留映射、身份和累计流量；客户端自动重连，不会新建额外隧道。

只有 Linux 服务端有网页。网页仅监听本机，纯只读，不要求设置网页密码，也没有配置按钮。
从自己的电脑建立 SSH 转发：

```bash
ssh -L 18080:127.0.0.1:DASHBOARD_PORT user@SERVER_IP
```

将大写占位内容替换成实际端口、用户名和 IP，在本机浏览器打开 `http://127.0.0.1:18080`。
无需也不应为了这个网页修改其它网站的 Nginx 配置。

## 6. 离线安装、更新与卸载

从 Release 下载符合 CPU/系统的包并校验 checksums.txt。
Linux 可以将 RPM/DEB 和安装脚本上传后执行：

```bash
bash install-linux.sh server --package ./simplefrp-server-linux-amd64.rpm
sudo simplefrp run
```

客户端将 server 和包名中的 server 改为 client。Linux 两种角色不能在同一个默认安装目录同时安装。
Windows 解压 ZIP 后双击 install.cmd 或运行 install.ps1；macOS 解压 TAR.GZ 后执行 `sh install.sh`。
v2 更新可再次运行安装器，会停止已确认归属的旧进程、替换文件并复用配置。

v0.1 与 v0.2 不兼容，旧版 pwd/setup/set server 不再保留。安装器拒绝覆盖旧配置。
请先按 BACKUP.md 做私有备份，明确卸载旧版后安装 v2，并用新的字符串重新配对。

```bash
sudo simplefrp uninstall
```

Linux 使用上面的 sudo 命令；Windows/macOS 客户端使用 `simplefrp uninstall`。
卸载停止所属进程，删除所属自启项、运行文件及自己添加的 PATH 内容。Windows 程序文件会在卸载命令退出后短暂延迟删除。
在线客户端还会解除服务端的隧道与身份；服务端不可达时，本地仍卸载并提示后续检查服务端残留映射。
其它软件、其它 PATH 项、用户另存的备份不删除。此次验收不执行重启/登录自启测试，但安装器仍配置自启。

## 7. 排错与恢复

连接失败先检查两端均为 v2、心跳端口已放行、服务端 IP 正确、连接字符串完整。
转发失败再检查客户端目标应用是否运行及 local 端口是否正确。不要为了排错停止其它项目。
身份校验失败时不能关闭 TLS 验证，应确认连接的确为目标服务端，并重新取得它的正确字符串。
运行配置含凭据和私钥，备份必须保持私有；不能上传 GitHub。源码和运行状态是两类备份，分别参见 BACKUP.md。
