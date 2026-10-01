param([Parameter(Mandatory=$true)][string]$Root)

$ErrorActionPreference = 'Stop'
$Root = (Resolve-Path -LiteralPath $Root).Path
$binary = Join-Path $Root 'simplefrp-windows-x64.exe'
$runner = Join-Path $Root 'integration-tests-windows-x64.exe'
$log = Join-Path $Root 'test.log'
$report = Join-Path $Root 'result.json'

function Get-ProtectedState {
    $startup = [Environment]::GetFolderPath('Startup')
    $files = @()
    if (Test-Path -LiteralPath $startup) {
        $files = @(Get-ChildItem -LiteralPath $startup -File | Sort-Object Name | ForEach-Object {
            [ordered]@{name=$_.Name; sha256=(Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash}
        })
    }
    $processes = @(Get-CimInstance Win32_Process -Filter "Name='simplefrp.exe'" | ForEach-Object {
        [ordered]@{pid=$_.ProcessId; executable=$_.ExecutablePath; started=$_.CreationDate.ToUniversalTime().ToString('O')}
    })
    [ordered]@{
        user_path=[Environment]::GetEnvironmentVariable('Path','User')
        machine_path=[Environment]::GetEnvironmentVariable('Path','Machine')
        startup_files=$files
        existing_simplefrp=$processes
        sshd=(Get-Service -Name sshd).Status.ToString()
    } | ConvertTo-Json -Depth 6 -Compress
}

$before = Get-ProtectedState
$os = Get-CimInstance Win32_OperatingSystem
$oldBinary = $env:SIMPLEFRP_TEST_BINARY
$env:SIMPLEFRP_TEST_BINARY = $binary
try {
    & $runner '-test.v' '-test.run' '^TestIsolatedCLIWorkflow$' *> $log
    $code = $LASTEXITCODE
} finally {
    $env:SIMPLEFRP_TEST_BINARY = $oldBinary
}
$after = Get-ProtectedState
$result = [ordered]@{
    os=$os.Caption
    version=$os.Version
    build=$os.BuildNumber
    architecture=$os.OSArchitecture
    powershell=$PSVersionTable.PSVersion.ToString()
    cli_sha256=(Get-FileHash -LiteralPath $binary -Algorithm SHA256).Hash
    test_sha256=(Get-FileHash -LiteralPath $runner -Algorithm SHA256).Hash
    integration_exit_code=$code
    protected_state_unchanged=($before -ceq $after)
    startup_tested=$false
    desktop_window_tested=$false
    cross_platform_network_tested=$false
    windows_codex_used=$false
}
$result | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $report -Encoding UTF8
Get-Content -LiteralPath $report
Get-Content -LiteralPath $log
if ($code -ne 0) { exit $code }
if ($before -cne $after) { throw 'Protected state changed during isolated acceptance' }
