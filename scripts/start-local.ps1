[CmdletBinding()]
param(
    [string]$GoPath
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$projectRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$envFile = Join-Path $projectRoot '.env'
if (-not (Test-Path -LiteralPath $envFile)) {
    throw 'Missing .env. Copy sample.env to .env and configure it first.'
}

$goExecutable = $null
if ($GoPath) {
    $resolvedGoPath = Resolve-Path -LiteralPath $GoPath -ErrorAction SilentlyContinue
    if (-not $resolvedGoPath -or -not (Test-Path -LiteralPath $resolvedGoPath.Path -PathType Leaf)) {
        throw "Go executable was not found at: $GoPath"
    }
    $goExecutable = $resolvedGoPath.Path
}
else {
    $go = Get-Command go -ErrorAction SilentlyContinue
    if (-not $go) {
        throw 'Go was not found in PATH. Install Go 1.21 or newer, or pass -GoPath to go.exe.'
    }
    $goExecutable = $go.Source
}

Push-Location $projectRoot
try {
    & $goExecutable build -o bond.exe .
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }

    Write-Host 'Server runs in this window. Stop it with Ctrl+C.'
    & (Join-Path $projectRoot 'bond.exe')
    exit $LASTEXITCODE
}
finally {
    Pop-Location
}
