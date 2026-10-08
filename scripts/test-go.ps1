[CmdletBinding()]
param(
    [string]$Package = './...',
    [string]$Run = '.',
    [switch]$Verify,
    [switch]$Race
)

$ErrorActionPreference = 'Stop'
$testRoot = Split-Path -Parent $PSScriptRoot
Push-Location -LiteralPath $testRoot
try {
    if (-not $env:TEST_DATABASE_URL) { & (Join-Path $PSScriptRoot 'start-test-postgres.ps1') }
    $env:GOCACHE = Join-Path $testRoot '.tools/gocache'
    $env:GOPATH = Join-Path $testRoot '.tools/gopath'
    $env:GOTOOLCHAIN = 'local'
    $env:GOWORK = 'off'
    if ($Verify) {
        & (Join-Path $PSScriptRoot 'verify.ps1') -Race:$Race
    } else {
        $testGo = Join-Path $testRoot '.tools/go1.27.1/go/bin/go.exe'
        if ($Race) { & $testGo test -race $Package -run $Run -count=1 }
        else { & $testGo test $Package -run $Run -count=1 }
    }
    $testExit = $LASTEXITCODE
} finally {
    Pop-Location
}
exit $testExit
