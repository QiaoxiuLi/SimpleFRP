param([Parameter(Mandatory=$true)][string]$Root,[switch]$Desktop,[int]$AllowedPublicPort=0)
$ErrorActionPreference='Stop'
$Root=(Resolve-Path -LiteralPath $Root).Path
$binary=Join-Path $env:LOCALAPPDATA 'Programs\SimpleFRP\simplefrp.exe'
$windowReport=Join-Path $Root 'window.json'

if ($Desktop) {
    Start-Transcript -LiteralPath (Join-Path $Root 'desktop.log') -Force | Out-Null
    & $binary desktop
    if ($LASTEXITCODE -ne 0) { throw 'Desktop launcher failed' }
    Add-Type @'
using System;
using System.Runtime.InteropServices;
using System.Text;
public static class SimpleFRPWindowProbe {
    private delegate bool Callback(IntPtr hwnd, IntPtr parameter);
    [DllImport("user32.dll")] private static extern bool EnumWindows(Callback callback, IntPtr parameter);
    [DllImport("user32.dll")] private static extern bool IsWindowVisible(IntPtr hwnd);
    [DllImport("user32.dll", CharSet=CharSet.Unicode)] private static extern int GetWindowText(IntPtr hwnd, StringBuilder value, int count);
    public static int Count(string title) {
        int count=0;
        EnumWindows((hwnd,unused)=>{var text=new StringBuilder(512);GetWindowText(hwnd,text,text.Capacity);if(IsWindowVisible(hwnd)&&text.ToString().Contains(title))count++;return true;},IntPtr.Zero);
        return count;
    }
}
'@
    $cfg=Join-Path $env:LOCALAPPDATA 'SimpleFRP'
    $hash=[Security.Cryptography.SHA256]::Create().ComputeHash([Text.Encoding]::UTF8.GetBytes($cfg))
    $instance=([BitConverter]::ToString($hash).Replace('-','').ToLower()).Substring(0,16)
    $deadline=(Get-Date).AddSeconds(25)
    do {
        Start-Sleep -Milliseconds 300
        $visible=[SimpleFRPWindowProbe]::Count('SimpleFRP Status - '+$instance)
    } while ($visible -eq 0 -and (Get-Date) -lt $deadline)
    $record=Get-Content -LiteralPath (Join-Path $cfg 'data\monitor.pid.json') -Raw | ConvertFrom-Json
    $monitor=Get-CimInstance Win32_Process -Filter ('ProcessId='+$record.PID)
    [ordered]@{visible_status_windows=$visible;monitor_pid=$monitor.ProcessId;monitor_session=$monitor.SessionId;interactive_session=[Diagnostics.Process]::GetCurrentProcess().SessionId} | ConvertTo-Json | Set-Content -LiteralPath $windowReport -Encoding UTF8
    Stop-Transcript | Out-Null
    exit 0
}

function Get-Baseline {
    $startup=[Environment]::GetFolderPath('Startup')
    $files=@(Get-ChildItem -LiteralPath $startup -File | Sort-Object Name | ForEach-Object { [ordered]@{name=$_.Name;hash=(Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash} })
    [ordered]@{user_path=[Environment]::GetEnvironmentVariable('Path','User');machine_path=[Environment]::GetEnvironmentVariable('Path','Machine');startup=$files;sshd=(Get-Service sshd).Status.ToString()} | ConvertTo-Json -Depth 5 -Compress
}

if (Test-Path -LiteralPath $binary) { throw 'A native installation already exists; refusing to replace it for a test' }
if (Test-Path -LiteralPath (Join-Path $env:LOCALAPPDATA 'SimpleFRP')) { throw 'Existing runtime files were preserved' }
$before=Get-Baseline
$archive=Join-Path $Root 'simplefrp-client-windows-amd64.zip'
$unpacked=Join-Path $Root 'package'
Expand-Archive -LiteralPath $archive -DestinationPath $unpacked -Force
Remove-Item -LiteralPath $windowReport -ErrorAction SilentlyContinue
Get-ChildItem -LiteralPath $unpacked -Filter '*.ps1' | ForEach-Object {
    $tokens=$null;$parseErrors=$null
    [System.Management.Automation.Language.Parser]::ParseFile($_.FullName,[ref]$tokens,[ref]$parseErrors) | Out-Null
    if($parseErrors.Count){throw ($parseErrors | Out-String)}
}
& (Join-Path $unpacked 'install.ps1') -NoStart
# Reinstallation must reuse startup/PATH/configuration before the desktop probe.
& (Join-Path $unpacked 'install.ps1') -NoStart
$task='SimpleFRP-Acceptance-'+[Guid]::NewGuid().ToString('N')
$code=1
try {
    $user=[Security.Principal.WindowsIdentity]::GetCurrent().Name
    $principal=New-ScheduledTaskPrincipal -UserId $user -LogonType Interactive -RunLevel Limited
    $arguments='-NoProfile -ExecutionPolicy Bypass -File "'+$PSCommandPath+'" -Root "'+$Root+'" -Desktop'
    $action=New-ScheduledTaskAction -Execute 'powershell.exe' -Argument $arguments
    $settings=New-ScheduledTaskSettingsSet -ExecutionTimeLimit (New-TimeSpan -Minutes 2)
    Register-ScheduledTask -TaskName $task -Action $action -Principal $principal -Settings $settings | Out-Null
    Start-ScheduledTask -TaskName $task
    $deadline=(Get-Date).AddSeconds(90)
    while (!(Test-Path -LiteralPath $windowReport) -and (Get-Date) -lt $deadline) { Start-Sleep -Milliseconds 500 }
    if (!(Test-Path -LiteralPath $windowReport)) { throw 'Interactive desktop probe did not produce a result; no reboot/login was attempted' }
    $window=Get-Content -LiteralPath $windowReport -Raw | ConvertFrom-Json
    if ($window.visible_status_windows -lt 1 -or $window.monitor_session -eq 0) { throw 'Status window was not visible in the authorized user desktop' }
    $env:SIMPLEFRP_TEST_BINARY=$binary
    $env:SIMPLEFRP_LIVE_INVITE_FILE=Join-Path $Root 'invitation.txt'
    if($AllowedPublicPort -gt 0){$env:SIMPLEFRP_LIVE_PUBLIC_PORT=$AllowedPublicPort.ToString()}
    & (Join-Path $Root 'integration-tests-windows-x64.exe') '-test.v' '-test.run' '^TestLiveClientWorkflow$' *> (Join-Path $Root 'test.log')
    $code=$LASTEXITCODE
} finally {
    Get-ScheduledTaskInfo -TaskName $task -ErrorAction SilentlyContinue | Select-Object LastRunTime,LastTaskResult | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $Root 'task-result.json') -Encoding UTF8
    Unregister-ScheduledTask -TaskName $task -Confirm:$false -ErrorAction SilentlyContinue
    if((Test-Path -LiteralPath $binary) -and (Test-Path -LiteralPath (Join-Path $env:LOCALAPPDATA 'SimpleFRP\installation.json'))){ & $binary uninstall }
    Start-Sleep -Seconds 4
    Remove-Item Env:SIMPLEFRP_TEST_BINARY,Env:SIMPLEFRP_LIVE_INVITE_FILE,Env:SIMPLEFRP_LIVE_PUBLIC_PORT -ErrorAction SilentlyContinue
}
$after=Get-Baseline
$os=Get-CimInstance Win32_OperatingSystem
$remaining=@(Get-CimInstance Win32_Process -Filter "Name='simplefrp.exe'" | Where-Object {$_.ExecutablePath -eq $binary})
$result=[ordered]@{os=$os.Caption;version=$os.Version;powershell=$PSVersionTable.PSVersion.ToString();native_install_and_update_tested=$true;visible_status_window=$window.visible_status_windows;interactive_session=$window.monitor_session;integration_exit_code=$code;protected_state_unchanged=($before -ceq $after);installed_binary_removed=!(Test-Path -LiteralPath $binary);runtime_removed=!(Test-Path -LiteralPath (Join-Path $env:LOCALAPPDATA 'SimpleFRP'));remaining_owned_processes=$remaining.Count;boot_tested=$false;windows_codex_used=$false}
$result | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $Root 'result.json') -Encoding UTF8
Get-Content -LiteralPath (Join-Path $Root 'result.json')
Get-Content -LiteralPath (Join-Path $Root 'test.log')
if($code -ne 0 -or $before -cne $after -or $remaining.Count -ne 0 -or (Test-Path -LiteralPath $binary)){throw 'Native acceptance failed; see scoped result/log files'}
