[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$taskRoot = Split-Path -Parent $PSScriptRoot
$taskConfig = Get-Content -LiteralPath (Join-Path $PSScriptRoot 'go-toolchain.json') -Raw | ConvertFrom-Json
if ($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitOperatingSystem) {
    throw 'This bootstrap supports Windows amd64. Use a matching official Go SDK on other platforms.'
}
$taskToolRoot = Join-Path $taskRoot ('.tools/' + $taskConfig.version)
$taskGoExe = Join-Path $taskToolRoot 'go/bin/go.exe'
if (Test-Path -LiteralPath $taskGoExe) {
    $taskVersion = & $taskGoExe version
    if ($LASTEXITCODE -ne 0 -or $taskVersion -notmatch ('^go version ' + [regex]::Escape($taskConfig.version) + ' windows/amd64$')) {
        throw 'Existing local Go SDK does not match the pinned toolchain.'
    }
    Write-Output $taskVersion
    return
}
$taskUri = [Uri]$taskConfig.url
if ($taskUri.Scheme -ne 'https' -or $taskUri.Host -ne 'go.dev') {
    throw 'The toolchain URL must be an HTTPS download from go.dev.'
}
New-Item -ItemType Directory -Path $taskToolRoot -Force | Out-Null
$taskArchive = Join-Path $taskToolRoot ($taskConfig.version + '.zip')
if (-not (Test-Path -LiteralPath $taskArchive)) {
    Write-Output ('Downloading ' + $taskConfig.version + ' from go.dev ...')
    $taskProgressPreference = $ProgressPreference
    try {
        $ProgressPreference = 'SilentlyContinue'
        Invoke-WebRequest -Uri $taskConfig.url -OutFile $taskArchive -UseBasicParsing
    } finally {
        $ProgressPreference = $taskProgressPreference
    }
}
$taskHash = (Get-FileHash -LiteralPath $taskArchive -Algorithm SHA256).Hash.ToLowerInvariant()
if ($taskHash -ne $taskConfig.sha256) {
    throw ('Go archive checksum mismatch. Inspect and remove the archive before retrying: ' + $taskArchive)
}
Write-Output 'SHA256 verified. Extracting the local SDK ...'
Expand-Archive -LiteralPath $taskArchive -DestinationPath $taskToolRoot -Force
$taskVersion = & $taskGoExe version
if ($LASTEXITCODE -ne 0 -or $taskVersion -notmatch ('^go version ' + [regex]::Escape($taskConfig.version) + ' windows/amd64$')) {
    throw 'Extracted Go SDK does not match the pinned toolchain.'
}
Write-Output $taskVersion
Write-Output ('GoLand SDK directory: ' + (Join-Path $taskToolRoot 'go'))
