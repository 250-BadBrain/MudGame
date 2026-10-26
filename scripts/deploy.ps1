[CmdletBinding()]
param(
    [ValidatePattern('^[A-Za-z0-9][A-Za-z0-9_.@-]*$')]
    [string]$SshHost = 'oracle',
    [ValidatePattern('^/[A-Za-z0-9_./-]+$')]
    [string]$RemoteRoot = '/home/ubuntu/MudGame',
    [ValidatePattern('^[A-Za-z0-9][A-Za-z0-9_.@-]*$')]
    [string]$ServiceName = 'mud',
    [ValidatePattern('^[A-Za-z0-9][A-Za-z0-9_.-]*$')]
    [string]$ExeName = 'mud',
    [ValidateRange(1, 65535)]
    [int]$HealthPort = 4009,
    [string]$OutputDirectory,
    [switch]$BuildOnly,
    [switch]$StageOnly
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
if ($PSVersionTable.PSVersion.Major -lt 7) {
    throw 'PowerShell 7 or later is required.'
}
if ($BuildOnly -and $StageOnly) {
    throw 'BuildOnly and StageOnly cannot be combined.'
}
$RemoteRoot = $RemoteRoot.TrimEnd('/')
if (!$RemoteRoot -or ($RemoteRoot -split '/') -contains '..' -or ($RemoteRoot -split '/') -contains '.') {
    throw 'RemoteRoot must be an absolute project path without dot segments.'
}

function Invoke-CheckedNative {
    param([string]$Program, [string[]]$Arguments)
    & $Program @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$Program failed with exit code $LASTEXITCODE."
    }
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$serverRoot = Join-Path $repoRoot 'MudGame-server'
$go = (Get-Command go -CommandType Application -ErrorAction Stop | Select-Object -First 1).Source
$git = (Get-Command git -CommandType Application -ErrorAction Stop | Select-Object -First 1).Source
$tar = (Get-Command tar -CommandType Application -ErrorAction Stop | Select-Object -First 1).Source
if (!$BuildOnly) {
    $ssh = (Get-Command ssh -CommandType Application -ErrorAction Stop | Select-Object -First 1).Source
    $scp = (Get-Command scp -CommandType Application -ErrorAction Stop | Select-Object -First 1).Source
}
if (!$OutputDirectory) {
    $OutputDirectory = Join-Path $repoRoot 'artifacts/backend'
}
$releaseId = [DateTime]::UtcNow.ToString('yyyyMMddTHHmmssZ') + '-' + [Guid]::NewGuid().ToString('N').Substring(0, 8)
$releaseDir = Join-Path ([IO.Path]::GetFullPath($OutputDirectory)) $releaseId
$payloadDir = Join-Path $releaseDir 'payload'
New-Item -ItemType Directory -Path $payloadDir | Out-Null

$originalEnvironment = @{}
foreach ($name in @('GOOS', 'GOARCH', 'CGO_ENABLED', 'GOAMD64')) {
    $originalEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}
Push-Location -LiteralPath $serverRoot
try {
    $hostOS = (& $go env GOHOSTOS).Trim()
    if ($LASTEXITCODE -ne 0) { throw 'Cannot detect Go host OS.' }
    $hostArchitecture = (& $go env GOHOSTARCH).Trim()
    if ($LASTEXITCODE -ne 0) { throw 'Cannot detect Go host architecture.' }
    $env:GOOS = $hostOS
    $env:GOARCH = $hostArchitecture
    Write-Host 'Checking backend tests and static analysis...'
    Invoke-CheckedNative $go @('test', './...')
    Invoke-CheckedNative $go @('vet', './...')

    $env:GOOS = 'linux'
    $env:GOARCH = 'amd64'
    $env:GOAMD64 = 'v1'
    $env:CGO_ENABLED = '0'
    Write-Host 'Building current source for Linux amd64...'
    Invoke-CheckedNative $go @('build', '-trimpath', '-buildvcs=true', '-o', (Join-Path $payloadDir $ExeName), './cmd/mudserver')
}
finally {
    Pop-Location
    foreach ($name in $originalEnvironment.Keys) {
        [Environment]::SetEnvironmentVariable($name, $originalEnvironment[$name], 'Process')
    }
}

Copy-Item -LiteralPath (Join-Path $serverRoot 'resources') -Destination (Join-Path $payloadDir 'resources') -Recurse
$revision = (& $git -C $repoRoot rev-parse HEAD).Trim()
if ($LASTEXITCODE -ne 0) { throw 'Cannot determine Git revision.' }
$status = & $git -C $repoRoot status --porcelain
if ($LASTEXITCODE -ne 0) { throw 'Cannot determine Git working tree status.' }
$files = [ordered]@{}
foreach ($file in (Get-ChildItem -LiteralPath $payloadDir -File -Recurse | Sort-Object FullName)) {
    $name = [IO.Path]::GetRelativePath($payloadDir, $file.FullName).Replace('\', '/')
    $files[$name] = (Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
}
$manifest = [ordered]@{
    releaseId = $releaseId
    revision = $revision
    workingTreeDirty = [bool]$status
    builtAtUtc = [DateTime]::UtcNow.ToString('o')
    target = 'linux/amd64'
    files = $files
}
$manifest | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $payloadDir 'release.json') -Encoding utf8
$archive = Join-Path $releaseDir 'package.tar.gz'
Invoke-CheckedNative $tar @('-czf', $archive, '-C', $payloadDir, $ExeName, 'resources', 'release.json')
$archiveHash = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant()
Write-Host "Release: $releaseId; revision: $revision; working tree modified: $([bool]$status)"
Write-Host "Package: $archive"
Write-Host "SHA256: $archiveHash"
if ($BuildOnly) {
    Write-Host 'BuildOnly complete. No remote connection was made.'
    return
}

$sshOptions = @('-T', '-o', 'BatchMode=yes', '-o', 'ConnectTimeout=15', '-o', 'ServerAliveInterval=15', '-o', 'ServerAliveCountMax=3')
$prepare = "from pathlib import Path; import sys; Path(sys.argv[1], '.deploy', 'incoming').mkdir(parents=True, exist_ok=True)"
$prepare | & $ssh @sshOptions $SshHost python3 - $RemoteRoot
if ($LASTEXITCODE -ne 0) { throw 'Remote staging preparation failed; production was not stopped.' }
$destination = "${SshHost}:${RemoteRoot}/.deploy/incoming/$releaseId.tar.gz"
Write-Host 'Uploading complete release before stopping production...'
Invoke-CheckedNative $scp @('-q', '-o', 'BatchMode=yes', '-o', 'ConnectTimeout=15', $archive, $destination)
$remoteArguments = @('python3', '-', '--root', $RemoteRoot, '--release', $releaseId, '--service', $ServiceName,
    '--executable', $ExeName, '--port', "$HealthPort", '--sha256', $archiveHash)
if ($StageOnly) { $remoteArguments += '--stage-only' }
Get-Content -LiteralPath (Join-Path $PSScriptRoot 'deploy_remote.py') -Raw -Encoding utf8 |
    & $ssh @sshOptions $SshHost @remoteArguments
if ($LASTEXITCODE -ne 0) {
    throw 'Remote deployment failed. Review the remote error and backup path above.'
}
