# Integration Tests

Run `go test -race ./...` from the repository root. `TestIsolatedCLIWorkflow`
builds the actual CLI and executes it in three temporary installation homes.
It does not install a system service, change firewall rules, use the default
configuration, or open a desktop window.

The workflow covers initialization idempotency, pairing idempotency, globally
unique tunnel IDs, client ownership, real TCP payload forwarding, live traffic
counters, tunnel creation, server-side port changes, occupied-port rollback,
read-only HTTP status, heartbeat migration, and scoped uninstall. A second
client keeps forwarding while the first client is uninstalled. The final
checks confirm that owned runtime directories and listeners have been removed.

For native acceptance without installing Go on the target, cross-compile this
test package with `go test -c`, transfer its test executable and the matching
CLI, and set `SIMPLEFRP_TEST_BINARY` to the CLI's absolute path. Run the test
executable with `-test.v -test.run TestIsolatedCLIWorkflow`. Test connection
strings and private keys must not be printed or included in public reports.

On an authorized Windows test host, place the two executables and
`acceptance-windows.ps1` in a new dedicated temporary directory. Run:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\acceptance-windows.ps1 -Root .
```

The wrapper records the actual OS build, PowerShell version, artifact hashes,
test exit code, and whether the existing SimpleFRP processes, startup files,
user/system PATH, and SSH service stayed unchanged. It does not change the
execution policy persistently. Check `result.json` and `test.log`, then remove
only the dedicated test directory after confirming no test process remains.

The isolated tests deliberately set `SIMPLEFRP_TEST_SERVER=1`. This enables a
server only for the test process on macOS or Windows, so the protocol can be
checked natively. It is not a supported server deployment on those systems.

Execution on a verified native host proves the isolated CLI workflow on that
specific OS build. It does not prove real Linux-to-Windows networking, desktop
window visibility, package installation, startup at login, or reboot behavior.
Those remain separate release gates requiring the authorized native targets.
Do not restart or alter unrelated production services to satisfy a test gate.
Replacing SimpleFRP itself requires explicit authorization and a private rollback backup.

## Native Package And Cross-Platform Acceptance

`TestLiveClientWorkflow` is opt-in and tests an explicitly authorized disposable
native client installation against a real Linux server. Set
`SIMPLEFRP_TEST_BINARY` and `SIMPLEFRP_LIVE_INVITE_FILE`; the latter points at a
private file, never a public test fixture. It covers real TCP payloads, repeat
pairing, next/delete, both mapping directions, traffic totals, independent
viewer shutdown, online unpair and native uninstall.

On Windows 10/11 use `acceptance-windows-native.ps1` with the release ZIP and
matching cross-compiled test executable in a dedicated temporary directory.
The wrapper installs and reinstalls the client, uses a uniquely named temporary
InteractiveToken task only to launch this software on the logged-in desktop,
checks the visible status window, runs the live workflow, and removes its own
task and installation. It compares unrelated startup files, user/machine PATH
and the SSH service before and after. It does not reboot, log in, or operate Codex.

When a cloud firewall already permits one dedicated SimpleFRP public port,
`-AllowedPublicPort PORT` reuses that port sequentially without widening any
firewall rule. `SIMPLEFRP_LIVE_PUBLIC_PORT=PORT` is the equivalent for the native
Go harness. Automatic allocation is still tested separately on native loopback.
All public ports otherwise require explicit network allowance by the operator.

See `docs/RELEASE_ACCEPTANCE.md` for the recorded platform matrix and exclusions.
