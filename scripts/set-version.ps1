param(
    [Parameter(Position = 0)]
    [string]$Version,

    [switch]$Show
)

$ErrorActionPreference = 'Stop'
Set-Location (Join-Path $PSScriptRoot '..')

function Get-ProjectVersion {
    $versionGo = Join-Path (Get-Location) 'version.go'
    $text = [System.IO.File]::ReadAllText($versionGo, [System.Text.UTF8Encoding]::new($false))
    if ($text -match 'AppVersion\s*=\s*"([^"]+)"') {
        return $Matches[1]
    }
    throw 'Cannot read current version from version.go'
}

function Normalize-Version([string]$value) {
    $value = $value.Trim()
    if ($value.StartsWith('v')) { $value = $value.Substring(1) }
    if ($value -notmatch '^\d+\.\d+\.\d+$') {
        throw '[error] Invalid version. Use major.minor.patch (e.g. 1.0.1).'
    }
    return $value
}

function Get-NextPatchVersion([string]$value) {
    $normalized = Normalize-Version $value
    $parts = $normalized.Split('.')
    $parts[2] = ([int]$parts[2] + 1).ToString()
    return ($parts -join '.')
}

function Set-TextFile([string]$path, [string]$content) {
    $utf8 = New-Object System.Text.UTF8Encoding $false
    [System.IO.File]::WriteAllText($path, $content, $utf8)
}

$current = Get-ProjectVersion
Write-Host ''
Write-Host '=== ZServerBackup set version ==='
Write-Host "Current: $current"

if ($Show) { exit 0 }

if (-not $Version) {
    $suggestedVersion = Get-NextPatchVersion $current
    $inputVersion = Read-Host "New version (e.g. $suggestedVersion)"
    if ([string]::IsNullOrWhiteSpace($inputVersion)) {
        $Version = $suggestedVersion
    } else {
        $Version = $inputVersion
    }
}

$newVersion = Normalize-Version $Version
if ($newVersion -eq $current) {
    Write-Host '[info] Version unchanged.'
    exit 0
}

$root = Get-Location
$versionGo = Join-Path $root 'version.go'
$versionText = [System.IO.File]::ReadAllText($versionGo, [System.Text.UTF8Encoding]::new($false))
$versionText = [regex]::Replace($versionText, 'AppVersion\s*=\s*"[^"]+"', "AppVersion = `"$newVersion`"")
Set-TextFile $versionGo $versionText

$configYml = Join-Path $root 'build\config.yml'
$configText = [System.IO.File]::ReadAllText($configYml, [System.Text.UTF8Encoding]::new($false))
$configText = [regex]::Replace($configText, '(?m)^  version: "[^"]+"', "  version: `"$newVersion`"")
Set-TextFile $configYml $configText

$gopath = & go env GOPATH 2>$null
if ($gopath) { $env:PATH = (Join-Path $gopath 'bin') + ';' + $env:PATH }

& wails3 task common:update:build-assets
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$nfpmPath = Join-Path $root 'build\linux\nfpm\nfpm.yaml'
if (Test-Path $nfpmPath) {
    $nfpmText = [System.IO.File]::ReadAllText($nfpmPath, [System.Text.UTF8Encoding]::new($false))
    $nfpmText = $nfpmText.Replace('./bin/ZServerBackup', './dist/ZServerBackup')
    Set-TextFile $nfpmPath $nfpmText
}

Write-Host "Done: $current -> $newVersion"
