[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$raceRoot = Split-Path -Parent $PSScriptRoot
if ($env:OS -ne 'Windows_NT' -or -not [Environment]::Is64BitOperatingSystem) {
    throw 'This compiler bootstrap supports Windows amd64 only.'
}
$raceArchive = Join-Path $raceRoot '.tools/w64devkit-x64-2.10.0.7z.exe'
$raceToolRoot = Join-Path $raceRoot '.tools/w64devkit-2.10.0'
$raceCompiler = Join-Path $raceToolRoot 'w64devkit/bin/gcc.exe'
New-Item -ItemType Directory -Path (Join-Path $raceRoot '.tools') -Force | Out-Null
if (-not (Test-Path -LiteralPath $raceArchive)) {
    $raceProgress = $ProgressPreference
    try {
        $ProgressPreference = 'SilentlyContinue'
        Invoke-WebRequest -UseBasicParsing -Uri 'https://github.com/skeeto/w64devkit/releases/download/v2.10.0/w64devkit-x64-2.10.0.7z.exe' -OutFile $raceArchive
    } finally {
        $ProgressPreference = $raceProgress
    }
}
$raceHash = (Get-FileHash -LiteralPath $raceArchive -Algorithm SHA256).Hash.ToLowerInvariant()
if ($raceHash -ne '18d0a4c71a166f8401ab6305781bec5882b40b5e06ba9807c61cb5f3b3c6325e') {
    throw 'Compiler archive checksum mismatch; do not execute it.'
}
if (-not (Test-Path -LiteralPath $raceCompiler)) {
    $raceExtraction = Start-Process -WindowStyle Hidden -Wait -PassThru -FilePath $raceArchive -ArgumentList @('-y', ('-o' + $raceToolRoot))
    if ($raceExtraction.ExitCode -ne 0) { throw 'Compiler extraction failed.' }
}
$raceVersion = @(& $raceCompiler --version)
if ($LASTEXITCODE -ne 0 -or $raceVersion[0] -notmatch '\(GCC\) 16\.2\.0$') { throw 'Portable compiler does not match pinned GCC 16.2.0.' }
$raceLibrary = & $raceCompiler --print-file-name libsynchronization.a
if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $raceLibrary)) { throw 'MinGW synchronization library required for race is missing.' }
Write-Output $raceVersion[0]
Write-Output ('Race compiler ready: ' + $raceCompiler)
