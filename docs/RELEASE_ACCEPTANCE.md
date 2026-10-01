# SimpleFRP v0.2.0 Release Acceptance

Date: 2026-09-30 (America/Los_Angeles).

## Scope And Safety

Only SimpleFRP installations and explicitly owned temporary acceptance files
are modified. Qingdao's legacy SimpleFRP is privately backed up before removal.
Existing Nginx sites, Node application, x-ui, SSH service and firewall rules are
protected by process, restart-counter, configuration-hash and HTTP baselines.
Windows is accessed only through authorized SSH. No Windows Codex task is used.

Reboot/login autostart execution is excluded at the user's explicit request.
Installers still configure systemd, LaunchAgents or the current-user Startup
shortcut. Native uninstall must remove those owned entries. Neither the host
nor a VM is restarted or logged out for acceptance.

## Native Candidate Matrix

| Native host | Architecture | Verified workflow |
| --- | --- | --- |
| Qingdao Linux, systemd/RPM | x86_64 | Native client install, real payload forwarding, repeat pairing, tunnel creation and deletion, mapping changes, counters, complete uninstall; native Linux server install and operation |
| Windows 10 Pro, build 19045, PowerShell 5.1 | x64 | ZIP install/reinstall, visible desktop status window, Linux-server public-network forwarding, tunnel operations, counters, independent viewer shutdown, native uninstall, unchanged unrelated PATH/startup/SSH |
| Windows 11 Pro, build 26200, PowerShell 5.1 | x64 | Same workflow; a reproduced status-viewer process-registration issue was fixed using native Windows process APIs and successfully retested |
| macOS 26.6.2, build 25G83 | arm64 | TAR install/reinstall, Terminal-attached status process, Linux-server public-network forwarding, tunnel operations, counters, independent viewer shutdown, native uninstall and removal of both LaunchAgents |

The Linux server is the only supported server deployment. Cross-platform tests
therefore mean Linux server to each of the three supported client platforms,
not an unsupported Windows/macOS server.

The cloud firewall did not allow the newly generated random port. Public-network
acceptance reused existing SimpleFRP-specific allowed ports through SimpleFRP's
own commands; no firewall/security-group rule or unrelated listening service
was changed. Automatic three-port allocation and additional tunnel allocation
were verified on the native isolated workflows.

## Logic And Failure Cases

`go test -race ./...` exercises initialization/pairing idempotency, unique IDs,
client ownership, multiple clients, tunnel creation/deletion, both peers'
mapping changes, occupied-port rollback, heartbeat migration and reconnect,
dashboard rebinding and read-only HTTP rejection, restart persistence of
mappings/totals, scoped uninstall and online unpair cleanup.

Focused protocol tests cover TLS pin rejection, modified signed requests,
replay rejection, oversized authenticated control-frame rejection, TCP
half-close and data coalesced with a handshake. A repeated pairing cannot
silently replace the established certificate fingerprint. Invalid unspecified
or multicast endpoints are rejected; explicit IPv6 initialization chooses an
IPv6 listener.

Native Windows regression tests also prove stable process identity, rejection
of a changed identity, and file-handle release before termination returns.
Windows clients use TerminateProcess plus WaitForSingleObject; the unsupported
test-only Windows server is allowed to finish its periodic durable sample
before an abrupt-stop restart test. Linux's production server graceful-drain
and persistence assertions do not use that delay.

`go vet ./...`, shell parsing, GoReleaser configuration validation and native
PowerShell parsing are release gates. `govulncheck` was run for Linux, macOS and
Windows; x/sys was updated to 0.44.0 to include the reported Windows-module fix.
The four-page PDF uses an embedded Chinese font and was rendered and visually
checked on every page. Desktop/mobile status-page layout was also checked.

## Final Artifact Gate

Publication requires installing and testing the final non-snapshot artifacts,
not only the candidate builds. The release's `acceptance-report.json` records
the final package hashes and successful native gates. All downloadable
packages, install scripts, guide and report are included in `checksums.txt`.
The published assets are downloaded again and checked for hashes and archive
integrity. A recoverable Git bundle and private runtime backups stay outside Git.

Linux arm64 and macOS amd64 are cross-compiled release artifacts, not claims of
native hardware execution. DEB artifacts receive package-structure verification;
this host's native Linux package-manager acceptance is RPM. Code signing and
notarization are not claimed. This test report is not a promise that every
possible environment or future failure is covered.
