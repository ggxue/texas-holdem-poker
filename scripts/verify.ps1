[CmdletBinding()]
param(
    [ValidateSet('Harness', 'Go', 'All')]
    [string]$Stage = 'All',
    [switch]$Race
)

$ErrorActionPreference = 'Stop'
$taskRoot = Split-Path -Parent $PSScriptRoot
$taskFailures = [System.Collections.Generic.List[string]]::new()

function Invoke-Check {
    param([string]$Name, [scriptblock]$Action)
    Write-Output ('Checking: ' + $Name)
    try {
        & $Action
        Write-Output ('PASS: ' + $Name)
    } catch {
        $taskFailures.Add($Name)
        Write-Output ('FAIL: ' + $Name + ' - ' + $_.Exception.Message)
    }
}

function Assert-NativeExit {
    param([string]$Name)
    if ($LASTEXITCODE -ne 0) { throw ($Name + ' exited with ' + $LASTEXITCODE) }
}

function Test-Harness {
    $taskRequired = @('AGENTS.md', 'CODING_STANDARDS.md', '.gitignore', '.gitattributes', '.agents/skills.lock.json',
        '.agents/skills/LICENSE', 'docs/agents/issue-tracker.md', 'docs/agents/domain.md',
        'docs/agents/skills-source.md', 'docs/agents/harness.md', 'scripts/go-toolchain.json',
        'scripts/bootstrap-go.ps1', 'scripts/verify.ps1', '.scratch/harness-init/issues/01-initialize.md')
    foreach ($taskRelative in $taskRequired) {
        if (-not (Test-Path -LiteralPath (Join-Path $taskRoot $taskRelative) -PathType Leaf)) {
            throw ('Missing harness file: ' + $taskRelative)
        }
    }
    $taskLock = Get-Content -LiteralPath (Join-Path $taskRoot '.agents/skills.lock.json') -Raw | ConvertFrom-Json
    if ($taskLock.commit -notmatch '^[a-f0-9]{40}$') { throw 'Upstream commit must be a full SHA.' }
    $taskNames = @($taskLock.skills | ForEach-Object { $_.name })
    if (@($taskNames | Select-Object -Unique).Count -ne $taskNames.Count) { throw 'Duplicate skill names.' }
    foreach ($taskSkill in $taskLock.skills) {
        $taskFolder = Join-Path $taskRoot ('.agents/skills/' + $taskSkill.name)
        $taskManifest = Join-Path $taskFolder 'SKILL.md'
        if (-not (Test-Path -LiteralPath $taskManifest)) { throw ('Missing skill: ' + $taskSkill.name) }
        $taskText = Get-Content -LiteralPath $taskManifest -Raw -Encoding UTF8
        if ($taskText -notmatch '(?s)^---\r?\n(.+?)\r?\n---') { throw ('Invalid frontmatter: ' + $taskSkill.name) }
        $taskFrontmatter = $Matches[1]
        if ($taskFrontmatter -notmatch ('(?m)^name: ' + [regex]::Escape($taskSkill.name) + '\s*$') -or
            $taskFrontmatter -notmatch '(?m)^description: .+') {
            throw ('Missing/mismatched name or description: ' + $taskSkill.name)
        }
        if ($taskSkill.explicitOnly) {
            if ($taskFrontmatter -notmatch '(?m)^disable-model-invocation: true\s*$') {
                throw ('Upstream explicit invocation flag missing: ' + $taskSkill.name)
            }
            $taskPolicy = Get-Content -LiteralPath (Join-Path $taskFolder 'agents/openai.yaml') -Raw
            if ($taskPolicy -notmatch 'allow_implicit_invocation:\s*false') {
                throw ('Explicit invocation policy missing: ' + $taskSkill.name)
            }
        }
        foreach ($taskDependency in $taskSkill.dependencies) {
            if ($taskNames -notcontains $taskDependency) {
                throw ($taskSkill.name + ' depends on missing skill ' + $taskDependency)
            }
        }
        foreach ($taskFile in $taskSkill.files) {
            $taskFilePath = Join-Path $taskFolder $taskFile.path
            if (-not (Test-Path -LiteralPath $taskFilePath -PathType Leaf)) { throw ('Missing resource: ' + $taskFilePath) }
            $taskHash = (Get-FileHash -LiteralPath $taskFilePath -Algorithm SHA256).Hash.ToLowerInvariant()
            if ($taskHash -ne $taskFile.installedSha256) { throw ('Unrecorded skill change: ' + $taskFilePath) }
        }
    }
    $taskMarkdown = @(Get-ChildItem -LiteralPath (Join-Path $taskRoot '.agents/skills') -Filter '*.md' -Recurse -File)
    $taskMarkdown += @(Get-ChildItem -LiteralPath (Join-Path $taskRoot 'docs/agents') -Filter '*.md' -File)
    $taskMarkdown += @(Get-Item -LiteralPath (Join-Path $taskRoot 'AGENTS.md'))
    foreach ($taskDoc in $taskMarkdown) {
        $taskText = Get-Content -LiteralPath $taskDoc.FullName -Raw -Encoding UTF8
        # Sample markdown inside fenced code blocks is not a resource reference.
        $taskText = [regex]::Replace($taskText, '(?ms)^[ \t]*(`{3,}|~{3,})[^\r\n]*\r?\n.*?^[ \t]*\1[ \t]*$', '')
        foreach ($taskMatch in [regex]::Matches($taskText, '\]\(([^)]+)\)')) {
            $taskTarget = $taskMatch.Groups[1].Value.Trim('<', '>')
            if ($taskTarget -match '^[a-zA-Z][a-zA-Z0-9+.-]*:' -or $taskTarget.StartsWith('#')) { continue }
            $taskTarget = ($taskTarget -split '#', 2)[0]
            if ($taskTarget -and -not (Test-Path -LiteralPath (Join-Path $taskDoc.DirectoryName $taskTarget))) {
                throw ('Broken local link in ' + $taskDoc.FullName + ': ' + $taskTarget)
            }
        }
    }
    foreach ($taskScript in Get-ChildItem -LiteralPath $PSScriptRoot -Filter '*.ps1' -File) {
        $taskTokens = $null
        $taskParseErrors = $null
        [System.Management.Automation.Language.Parser]::ParseFile($taskScript.FullName, [ref]$taskTokens, [ref]$taskParseErrors) | Out-Null
        if ($taskParseErrors.Count) { throw ('PowerShell parse error in ' + $taskScript.Name + ': ' + $taskParseErrors[0].Message) }
    }
    Write-Output ('Validated ' + $taskNames.Count + ' pinned skills, dependencies, resources, links, and scripts.')
}

Push-Location -LiteralPath $taskRoot
try {
    if ($Stage -ne 'Go') { Invoke-Check 'harness' { Test-Harness } }
    if ($Stage -ne 'Harness') {
        $taskConfig = Get-Content -LiteralPath (Join-Path $PSScriptRoot 'go-toolchain.json') -Raw | ConvertFrom-Json
        $taskGoExe = Join-Path $taskRoot ('.tools/' + $taskConfig.version + '/go/bin/go.exe')
        if (-not (Test-Path -LiteralPath $taskGoExe)) {
            throw 'Pinned Go SDK missing. Run scripts/bootstrap-go.ps1 first.'
        }
        $taskGoFmt = Join-Path (Split-Path -Parent $taskGoExe) 'gofmt.exe'
        # Process-local caches keep sandboxed runs reproducible and avoid user caches.
        $env:GOCACHE = Join-Path $taskRoot '.tools/gocache'
        $env:GOPATH = Join-Path $taskRoot '.tools/gopath'
        $env:GOTOOLCHAIN = 'local'
        $env:GOWORK = 'off'
        New-Item -ItemType Directory -Path (Join-Path $taskRoot 'artifacts/build') -Force | Out-Null
        Invoke-Check 'Go toolchain' {
            $taskVersion = & $taskGoExe version
            Assert-NativeExit 'go version'
            if ($taskVersion -notmatch ('^go version ' + [regex]::Escape($taskConfig.version) + ' windows/amd64$')) {
                throw 'Go version differs from scripts/go-toolchain.json.'
            }
            Write-Output $taskVersion
        }
        Invoke-Check 'Go formatting' {
            $taskFiles = @(git ls-files --cached --others --exclude-standard -- '*.go' | Select-Object -Unique)
            Assert-NativeExit 'git ls-files'
            if ($taskFiles.Count) {
                $taskUnformatted = @(& $taskGoFmt -l @taskFiles)
                Assert-NativeExit 'gofmt'
                if ($taskUnformatted.Count) { throw ('Run gofmt on: ' + ($taskUnformatted -join ', ')) }
            }
        }
        Invoke-Check 'Go vet' { & $taskGoExe vet ./...; Assert-NativeExit 'go vet' }
        Invoke-Check 'Go tests' { & $taskGoExe test ./...; Assert-NativeExit 'go test' }
        Invoke-Check 'Go build' { & $taskGoExe build -o artifacts/build/texas-poker.exe .; Assert-NativeExit 'go build' }
        if ($Race) { Invoke-Check 'Go race tests' { & $taskGoExe test -race ./...; Assert-NativeExit 'go test -race' } }
    }
} catch {
    $taskFailures.Add('verification setup')
    Write-Output ('FAIL: verification setup - ' + $_.Exception.Message)
} finally {
    Pop-Location
}
if ($taskFailures.Count) {
    Write-Output ('Verification failed: ' + ($taskFailures -join ', '))
    exit 1
}
Write-Output 'Verification passed.'
exit 0
