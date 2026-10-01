import pathlib
import re
import textwrap
from xml.sax.saxutils import escape

from reportlab.lib import colors
from reportlab.lib.enums import TA_LEFT
from reportlab.lib.pagesizes import A4
from reportlab.lib.styles import ParagraphStyle
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont
from fontTools.ttLib import TTFont as SourceFont
from fontTools.varLib.instancer import instantiateVariableFont
from reportlab.platypus import Paragraph, Preformatted, SimpleDocTemplate, Spacer, Table, TableStyle

font_file = pathlib.Path(".tools/fonts/NotoSansSC-Regular.ttf")
if not font_file.is_file():
    source_font = SourceFont(".tools/fonts/NotoSansSC-variable.ttf")
    instantiateVariableFont(source_font, {"wght": 400}, inplace=True).save(font_file)
pdfmetrics.registerFont(TTFont("NotoSC", str(font_file)))
body = ParagraphStyle("body", fontName="NotoSC", fontSize=10, leading=16, spaceAfter=8, wordWrap="CJK")
h1 = ParagraphStyle("h1", parent=body, fontSize=23, leading=30, spaceAfter=16, textColor=colors.HexColor("#176B56"))
h2 = ParagraphStyle("h2", parent=body, fontSize=14, leading=21, spaceBefore=12, spaceAfter=9, keepWithNext=True, textColor=colors.HexColor("#176B56"))
code = ParagraphStyle("code", fontName="Courier", fontSize=8, leading=11, spaceBefore=4, spaceAfter=10, backColor=colors.HexColor("#F1F4F3"), borderPadding=8)
small = ParagraphStyle("small", parent=body, fontSize=8.5, leading=13)
story = []

def inline(text):
    text = re.sub(r"\[([^\]]+)\]\([^)]+\)", r"\1", text)
    return escape(text.replace("`", "").replace("**", ""))

def add_markdown(text):
    block = []
    prose = []
    language = None
    def flush():
        if prose:
            story.append(Paragraph(inline(" ".join(prose)), body))
            prose.clear()
    for line in text.splitlines():
        if line.startswith("```"):
            flush()
            if language is None:
                language = line[3:].strip()
            else:
                rendered = []
                continuation = " `" if language == "powershell" else " " + chr(92)
                for value in block:
                    wrapped = textwrap.wrap(value, 90, break_long_words=False, break_on_hyphens=False) or [""]
                    for index, part in enumerate(wrapped):
                        suffix = continuation if index < len(wrapped)-1 else ""
                        rendered.append(("  " if index else "") + part + suffix)
                story.append(Preformatted("\n".join(rendered), code))
                block.clear()
                language = None
        elif language is not None:
            block.append(line)
        elif line.startswith("## "):
            flush()
            story.append(Paragraph(inline(line[3:]), h2))
        elif line.startswith("# "):
            flush()
        elif not line.strip():
            flush()
        else:
            prose.append(line)
    flush()

story.append(Paragraph("SimpleFRP v0.2.0", h1))
story.append(Paragraph("项目介绍、实现说明与操作指南", h2))
story.append(Paragraph("版本日期：2026-09-30。目标是快速部署 TCP 反向转发。Linux 提供服务端和客户端，Windows/macOS 提供客户端。安装后自动配置自启，连接依赖服务端输出的一串字符，不再使用旧版密码向导。", body))
story.append(Paragraph("本指南不含任何真实服务器凭据或私钥。所有示例中的 IP、端口、隧道编号和连接字符串占位符必须换成你的实际值。先确认目标应用在客户端运行，再将 local 端口指向它。", body))
add_markdown(pathlib.Path("docs/QUICKSTART.md").read_text())

implementation = """
## 8. 实现说明

初始化只执行一次。服务端使用加密随机数生成服务端身份、邀请密钥、管理密钥和 Ed25519 证书，自动探测可用的三个端口并保存 TOML 配置。再次运行复用身份和端口，不使已有客户端失效。实际监听绑定是最后的占用校验；碰到其它应用占用不会结束那个应用。

SF2 字符串使用双方相同的 Base62 编解码算法，包含版本、IPv4/IPv6、公网心跳端口、服务端身份、邀请密钥和证书 SHA-256 指纹，还附带校验值用于发现复制损坏。IPv4 字符串约 95 个字符。它只是编码而非加密，拥有它的人可以申请配对，必须像密码一样保密。

客户端首先校验固定的证书指纹，再通过 TLS 1.3 连接。服务端为每个客户端分配独立的 ID 和密钥，不把邀请密钥作为所有客户端永久共享的身份。控制请求使用完整 JSON 的 HMAC-SHA256、时间戳和随机 nonce 防止篡改和重放。身份、隧道和待处理连接数量有上限。

外部 TCP 请求到达 public 端口后，服务端通过控制通道通知所属客户端，客户端再建立独立的 TLS 数据连接，并连接自己的 127.0.0.1:local 端口。双方双向复制数据并保留 TCP 半关闭语义。隧道加密不代替应用端 HTTPS，public 应用协议是否加密由被转发的应用决定。

服务端统一管理所有隧道编号和映射。修改 public 前先绑定新端口，修改配置时要求在线客户端保存并 ACK。失败保留旧映射；成功后释放旧监听。服务端允许删除离线隧道，重新连接时以服务端状态为准。客户端卸载时会请求删除自己的服务端隧道和身份；服务端不可达时会明确提示未确认的远端清理。

bbolt 保存映射、身份、编号计数和累计流量，正常重启后恢复。流量统计使用原子计数器，每秒采样速率。关闭服务时等待所属连接和工作协程退出，再关闭存储。客户端断线重连复用身份和隧道，不重新创建隧道；缓存累计值不会被离线的零值覆盖。

网页由 Go embed 内嵌 HTML/CSS/JavaScript 和 net/http 提供，不依赖 CDN、Node.js 或 Nginx。浏览器定时读取唯一的只读状态接口，写方法被拒绝。网页只绑定回环地址，不要求网页密码，也不能改配置。Windows/macOS 状态窗口与后台转发进程独立，结束状态查看进程后转发仍应工作。

安装收据记录所属二进制、角色、PATH 修改和服务账户归属。自启文件带有安装标识，进程记录包含 PID、启动标识和可执行路径；Unix 端还核对命令摘要。Windows 用原生 API 核对同一进程句柄的身份，并等待进程完全退出后才清理文件。卸载不会通过同名进程或过期 PID 误删其它软件。Windows 使用延迟自删除，macOS 不删除共享的 .local/bin 目录，Linux 包管理器只负责删除自己的包文件。

## 9. 依赖与构建

运行依赖是编译进入二进制的 Go 库，不需要用户额外安装 Python、Node.js、Docker 或数据库。Cobra 1.8.1 负责命令解析；go-toml/v2 2.2.2 负责结构化 TOML；bbolt 1.3.11 负责持久化和进程锁；x/sys 0.44.0 负责平台系统调用和 Windows 终端模式。Cobra 还使用 pflag 和 Windows mousetrap。完整许可证随包提供在 THIRD_PARTY_NOTICES.md。

TLS、Ed25519、HMAC、JSON、TCP、HTTP 和随机数使用 Go 标准库。旧版仅用来读取版本号的上游 FRP 依赖及密码/Argon2 辅助代码已经移除，本项目不与 frpc/frps 线协议互通，也没有旧协议兼容开关。源码最低 Go 1.25，发布构建使用 Go 1.27.1。GoReleaser v2/nFPM 生成 DEB/RPM、ZIP、TAR.GZ 和校验文件。ReportLab、FontTools、Poppler 与嵌入的 Noto Sans SC 字体只用于生成和检查本 PDF，不是软件运行依赖。字体来自 Google Fonts，采用 SIL Open Font License。

Linux 需要 systemd 和 root/sudo 安装权限；Windows 使用内置 PowerShell 5.1、WScript.Shell 和当前用户启动目录；macOS 使用 launchd 与 Terminal。Windows/macOS 的自启是用户登录自启，不是无人登录时的系统服务。此次用户要求排除重启/登录自启验收，但安装仍应创建对应配置。

## 10. 备份、恢复与版本边界

GitHub 保存源码、提交、标签和 Release 安装包。git bundle 可作离线源码备份，并通过 git bundle verify 检查。拉取前先检查本地修改，使用 git pull --ff-only，不强制覆盖工作区。恢复源码不等于恢复服务端身份，后者还需要私有的配置、证书和数据库备份。

对运行数据备份时只停止 SimpleFRP 自己的服务，再归档它的配置和 bbolt 文件；不要将正在写入的数据库当成一致性快照。备份目录权限 700、归档权限 600，禁止把私钥、连接字符串、客户端密钥、PID 文件或日志提交到公开仓库。恢复到停止的同协议版本安装，丢弃旧 PID/锁记录，确认权限后重新安装自启元数据并启动。

v0.1 与 v0.2 不兼容，也不自动迁移旧密码配置。需要先保存旧版本二进制/安装包、服务定义和私有运行状态作为回退备份，再明确卸载旧版，安装新版并重新配对。v2 的普通更新保留配置。安装、卸载和测试不得为了本项目修改其它网站、数据库、服务、防火墙规则或终端应用中的其它会话。

仓库： https://github.com/QiaoxiuLi/SimpleFRP

发布下载： https://github.com/QiaoxiuLi/SimpleFRP/releases

详细操作、源码恢复与实际验收证据分别见 README.md、BACKUP.md 和 docs/RELEASE_ACCEPTANCE.md。
"""
add_markdown(implementation)

def footer(canvas, doc):
    canvas.saveState()
    canvas.setStrokeColor(colors.HexColor("#D3DBD7"))
    canvas.line(48, 40, A4[0]-48, 40)
    canvas.setFont("Helvetica", 8)
    canvas.setFillColor(colors.HexColor("#59665F"))
    canvas.drawString(48, 27, "SimpleFRP v0.2.0 | Project and Operations Guide")
    canvas.drawRightString(A4[0]-48, 27, str(doc.page))
    canvas.restoreState()

output = pathlib.Path("output/pdf/SimpleFRP_Project_Guide_v0.2.0.pdf")
output.parent.mkdir(parents=True, exist_ok=True)
document = SimpleDocTemplate(str(output), pagesize=A4, rightMargin=48, leftMargin=48, topMargin=45, bottomMargin=55, title="SimpleFRP v0.2.0 Project and Operations Guide", author="QiaoxiuLi / SimpleFRP")
document.build(story, onFirstPage=footer, onLaterPages=footer)
pathlib.Path("docs/SimpleFRP_Project_Guide_v0.2.0.pdf").write_bytes(output.read_bytes())
print(output)
