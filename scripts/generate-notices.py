import json
import os
import pathlib
import subprocess

modules = [
    "github.com/spf13/cobra",
    "github.com/spf13/pflag",
    "github.com/inconshreveable/mousetrap",
    "github.com/pelletier/go-toml/v2",
    "go.etcd.io/bbolt",
    "golang.org/x/sys",
]
go = os.environ.get("GO", "go")
parts = ["# Third-Party Runtime Notices\n\nGenerated from the pinned Go module licenses.\n"]
for module in modules:
    info = json.loads(subprocess.check_output([go, "list", "-m", "-json", module]))
    root = pathlib.Path(info["Dir"])
    license_file = next(path for path in [root / "LICENSE", root / "LICENSE.txt"] if path.is_file())
    parts.append(f"\n## {module} {info['Version']}\n\n```text\n{license_file.read_text().rstrip()}\n```\n")
font_license = pathlib.Path(".tools/fonts/OFL.txt")
if font_license.is_file():
    parts.append("\n## Noto Sans SC (embedded documentation font)\n\nSource: https://github.com/google/fonts/tree/main/ofl/notosanssc\n\n```text\n" + font_license.read_text().rstrip() + "\n```\n")
pathlib.Path("THIRD_PARTY_NOTICES.md").write_text("\n".join(line.rstrip() for line in "".join(parts).splitlines()) + "\n")
