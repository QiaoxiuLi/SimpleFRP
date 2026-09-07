#!/usr/bin/env bash
set -euo pipefail
role=${1:-server}
[[ $# -eq 0 ]] || shift
[[ "$role" == server || "$role" == client ]] || { echo 'Usage: linux.sh server|client [--package /path/to/package] [--no-configure] [setup flags]' >&2; exit 2; }
package=''; configure=1; setup_args=()
while [[ $# -gt 0 ]]; do
 case "$1" in
  --package) package=${2:?Missing package path}; shift 2 ;;
  --no-configure) configure=0; shift ;;
  *) setup_args+=("$1"); shift ;;
 esac
done
[[ $(uname -s) == Linux && $(uname -m) == x86_64 ]] || { echo 'This release supports Linux x86_64 packages.' >&2; exit 1; }
if command -v dpkg >/dev/null 2>&1; then format=deb; elif command -v rpm >/dev/null 2>&1; then format=rpm; else echo 'A DEB or RPM package manager is required.' >&2; exit 1; fi
privilege=()
if [[ $EUID -ne 0 ]]; then
 command -v sudo >/dev/null || { echo 'Run this Linux system-service installer as root or a sudo-enabled user.' >&2; exit 1; }
 sudo -v; privilege=(sudo)
fi
work=''
trap '[[ -z "$work" ]] || rm -rf -- "$work"' EXIT
if [[ -z "$package" ]]; then
 work=$(mktemp -d)
 base='https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download'
 name="simplefrp-$role-linux-amd64.$format"
 curl --fail --location --retry 2 --connect-timeout 15 "$base/$name" -o "$work/$name"
 curl --fail --location --retry 2 --connect-timeout 15 "$base/checksums.txt" -o "$work/checksums.txt"
 (cd "$work"; awk -v file="$name" '$2==file {print}' checksums.txt > selected.sha256; test -s selected.sha256; sha256sum --check selected.sha256)
 package="$work/$name"
fi
[[ -f "$package" && "$package" == *."$format" ]] || { echo 'Package does not exist or does not match this system.' >&2; exit 1; }
if [[ "$format" == deb ]]; then actual=$(dpkg-deb -f "$package" Package); else actual=$(rpm -qp --qf '%{NAME}' "$package"); fi
[[ "$actual" == "simplefrp-$role" ]] || { echo 'Package role does not match the requested role.' >&2; exit 1; }
backup="/var/backups/simplefrp-$(date +%Y%m%d-%H%M%S)"
"${privilege[@]}" install -d -m 700 "$backup"
# Back up only this program's existing files, including its live statistics database after stopping it.
if "${privilege[@]}" systemctl is-active --quiet "simplefrp-$role"; then "${privilege[@]}" systemctl stop "simplefrp-$role"; fi
for path in /etc/simplefrp /var/lib/simplefrp /usr/bin/simplefrp; do
 if "${privilege[@]}" test -e "$path"; then "${privilege[@]}" cp -a --parents "$path" "$backup/"; fi
done
if [[ "$format" == deb ]]; then
 # v0.1.0's old postrm runs during upgrades and lacks an upgrade guard.
 installed=$(dpkg-query -W -f='${Version}' "simplefrp-$role" 2>/dev/null || true)
 if [[ "$installed" == 0.1.0* ]]; then
  for script in prerm postrm; do
   path="/var/lib/dpkg/info/simplefrp-$role.$script"
   if "${privilege[@]}" test -f "$path"; then
    "${privilege[@]}" cp -p "$path" "$backup/$script"
    "${privilege[@]}" sed -i '2i case "${1:-}" in upgrade|failed-upgrade) exit 0 ;; esac' "$path"
   fi
  done
 fi
 "${privilege[@]}" dpkg -i "$package"
else
 installed=$(rpm -q --qf '%{VERSION}' "simplefrp-$role" 2>/dev/null || true)
 if [[ "$installed" == 0.1.0 ]]; then
  # Skip only the destructive old removal hooks; new install hooks still run.
  "${privilege[@]}" rpm -U --nopreun --nopostun "$package"
 else
  "${privilege[@]}" rpm -U "$package"
 fi
fi
if [[ $configure -eq 1 ]]; then
 if "${privilege[@]}" test -f "/etc/simplefrp/$role.toml"; then
  "${privilege[@]}" systemctl enable --now "simplefrp-$role"
  echo 'Existing configuration preserved. Use simplefrp setup explicitly to change it.'
 else
  "${privilege[@]}" /usr/bin/simplefrp setup --role "$role" "${setup_args[@]}"
 fi
fi
printf 'Installed %s. Backup: %s\n' "$role" "$backup"
