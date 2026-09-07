param([int]$LocalPort=0,[string]$Server="",[switch]$NoConfigure)
$ErrorActionPreference='Stop'
if (![Environment]::Is64BitOperatingSystem) { throw 'This package requires 64-bit Windows.' }
$work=Join-Path ([IO.Path]::GetTempPath()) ('simplefrp-'+[Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory $work | Out-Null
try {
    $base='https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download'
    $name='simplefrp-client-windows-amd64.zip'
    $archive=Join-Path $work $name
    Invoke-WebRequest "$base/$name" -OutFile $archive -UseBasicParsing
    $checks=(Invoke-WebRequest "$base/checksums.txt" -UseBasicParsing).Content
    $expected=($checks -split "`n" | Where-Object { $_ -match ('\s+'+[regex]::Escape($name)+'\s*$') } | Select-Object -First 1) -split '\s+'
    if (!$expected -or (Get-FileHash $archive -Algorithm SHA256).Hash -ne $expected[0]) { throw 'Release checksum verification failed.' }
    Expand-Archive $archive -DestinationPath $work
    $arguments=@('-NoProfile','-ExecutionPolicy','Bypass','-File',(Join-Path $work 'install.ps1'))
    if ($LocalPort -gt 0) { $arguments+=@('-LocalPort',[string]$LocalPort) }
    if ($Server) { $arguments+=@('-Server',$Server) }
    if ($NoConfigure) { $arguments+='-NoConfigure' }
    & powershell.exe @arguments
    if ($LASTEXITCODE -ne 0) { throw 'Installer did not complete successfully.' }
} finally { Remove-Item -LiteralPath $work -Recurse -Force }
