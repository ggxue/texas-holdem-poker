[CmdletBinding()]
param([switch]$Stop)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$pgRoot = Split-Path -Parent $PSScriptRoot
$pgArchive = Join-Path $pgRoot '.tools/postgresql-18.6-5-windows-x64-binaries.zip'
$pgExpected = 'e2246ba91d22345bc3d017586c09ede52d9df180b1eeb480f050445f1cad84e2'
$pgDirectory = Join-Path $pgRoot '.tools/postgresql-18.6-5'
$pgBin = Join-Path $pgDirectory 'pgsql/bin'
$pgData = Join-Path $pgRoot '.tools/pg-test-data'
$pgPasswordFile = Join-Path $pgRoot '.tools/pg-test-password.txt'
$pgCtl = Join-Path $pgBin 'pg_ctl.exe'

function Assert-PgExit([string]$Name) {
    if ($LASTEXITCODE -ne 0) { throw "$Name exited with $LASTEXITCODE" }
}

if ($Stop) {
    if (-not (Test-Path -LiteralPath $pgCtl)) { throw 'Local PostgreSQL binary missing.' }
    & $pgCtl stop -D $pgData -m fast -w -t 30
    Assert-PgExit 'pg_ctl stop'
    return
}

if (-not (Test-Path -LiteralPath $pgArchive)) {
    New-Item -ItemType Directory -Path (Join-Path $pgRoot '.tools') -Force | Out-Null
    & curl.exe --fail --location --retry 2 --output $pgArchive 'https://get.enterprisedb.com/postgresql/postgresql-18.6-5-windows-x64-binaries.zip'
    Assert-PgExit 'PostgreSQL download'
}
if ((Get-FileHash -LiteralPath $pgArchive -Algorithm SHA256).Hash.ToLowerInvariant() -ne $pgExpected) {
    throw 'PostgreSQL archive checksum mismatch; do not execute it.'
}
if (-not (Test-Path -LiteralPath (Join-Path $pgDirectory '.extracted'))) {
    New-Item -ItemType Directory -Path $pgDirectory -Force | Out-Null
    & tar.exe -xf $pgArchive -C $pgDirectory pgsql/bin pgsql/lib pgsql/share pgsql/server_license.txt
    Assert-PgExit 'PostgreSQL extraction'
    New-Item -ItemType File -Path (Join-Path $pgDirectory '.extracted') -Force | Out-Null
}
& (Join-Path $pgBin 'postgres.exe') --version
Assert-PgExit 'postgres version'
if (-not (Test-Path -LiteralPath (Join-Path $pgData 'PG_VERSION'))) {
    if (Test-Path -LiteralPath $pgData) { throw 'Uninitialized data directory exists; inspect it before retrying.' }
    $pgBytes = New-Object byte[] 32
    $pgRandom = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    try { $pgRandom.GetBytes($pgBytes) } finally { $pgRandom.Dispose() }
    [System.IO.File]::WriteAllText($pgPasswordFile, [Convert]::ToBase64String($pgBytes))
    & (Join-Path $pgBin 'initdb.exe') -D $pgData -U poker_test -E UTF8 --locale=C -A scram-sha-256 --pwfile=$pgPasswordFile
    Assert-PgExit 'initdb'
}
if (-not (Test-Path -LiteralPath $pgPasswordFile)) { throw 'Test database password file missing.' }
& $pgCtl status -D $pgData
if ($LASTEXITCODE -eq 3) {
    New-Item -ItemType Directory -Path (Join-Path $pgRoot 'artifacts') -Force | Out-Null
    & $pgCtl start -D $pgData -l (Join-Path $pgRoot 'artifacts/postgres-test.log') -o '-h 127.0.0.1 -p 55432' -w -t 30
    Assert-PgExit 'pg_ctl start'
} elseif ($LASTEXITCODE -ne 0) {
    throw "pg_ctl status exited with $LASTEXITCODE"
}
$pgPassword = [System.IO.File]::ReadAllText($pgPasswordFile).Trim()
# Tests create and drop their own randomly named databases; no production DSN is used.
$env:TEST_DATABASE_URL = 'postgres://poker_test:' + [uri]::EscapeDataString($pgPassword) + '@127.0.0.1:55432/postgres?sslmode=disable'
Write-Output 'Local test PostgreSQL ready on 127.0.0.1:55432; TEST_DATABASE_URL set for this PowerShell process.'
