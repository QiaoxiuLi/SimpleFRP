param([switch]$NoStart)
$ErrorActionPreference='Stop'
if (![Environment]::Is64BitOperatingSystem) { throw '64-bit Windows is required.' }
[Net.ServicePointManager]::SecurityProtocol=[Net.SecurityProtocolType]::Tls12
$work=Join-Path ([IO.Path]::GetTempPath()) ('simplefrp-'+[Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory $work | Out-Null
try {
    $base='https://github.com/QiaoxiuLi/SimpleFRP/releases/latest/download'
    $name='simplefrp-client-windows-amd64.zip'
    $archive=Join-Path $work $name
    Invoke-WebRequest "$base/$name" -OutFile $archive -UseBasicParsing
    $checks=(Invoke-WebRequest "$base/checksums.txt" -UseBasicParsing).Content
    $line=@($checks -split "`n" | Where-Object { $_ -match ('^[a-fA-F0-9]{64}\s+'+[regex]::Escape($name)+'\s*$') })
    if ($line.Count -ne 1) { throw 'Release checksum entry is missing or ambiguous.' }
    $expected=($line[0] -split '\s+')[0]
    if ((Get-FileHash $archive -Algorithm SHA256).Hash -ne $expected) { throw 'Release checksum verification failed.' }
    Expand-Archive $archive -DestinationPath $work
    $arguments=@('-NoProfile','-ExecutionPolicy','Bypass','-File',(Join-Path $work 'install.ps1'))
    if ($NoStart) { $arguments+='-NoStart' }
    & powershell.exe @arguments
    if ($LASTEXITCODE -ne 0) { throw 'Installer did not complete successfully.' }
    $env:Path=[Environment]::GetEnvironmentVariable('Path','Machine')+';'+[Environment]::GetEnvironmentVariable('Path','User')
} finally { Remove-Item -LiteralPath $work -Recurse -Force }
