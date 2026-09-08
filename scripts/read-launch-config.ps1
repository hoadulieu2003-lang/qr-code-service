[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string]$EnvPath
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

if (-not (Test-Path -LiteralPath $EnvPath -PathType Leaf)) {
    exit 1
}

$settings = @{}
foreach ($line in Get-Content -LiteralPath $EnvPath) {
    if ($line -match '^\s*(?<key>[A-Z_]+)\s*=\s*(?<value>.*?)\s*$') {
        $settings[$matches.key] = $matches.value.Trim().Trim('"')
    }
}

$portText = if ($settings.ContainsKey('PORT')) { $settings['PORT'] } else { '18080' }
$port = 0
if (-not [int]::TryParse($portText, [ref]$port) -or $port -lt 1 -or $port -gt 65535) {
    exit 1
}

$sslText = if ($settings.ContainsKey('SSL')) { $settings['SSL'] } else { 'FALSE' }
$sslEnabled = $sslText -match '^(?i:1|on|true)$'

Write-Output "VALID|$port|$sslEnabled"
