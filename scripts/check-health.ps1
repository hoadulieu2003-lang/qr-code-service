[CmdletBinding()]
param(
    [ValidateRange(1, 65535)]
    [int]$Port = 18080
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

try {
    $baseUrl = "http://localhost:$Port"
    $health = Invoke-WebRequest -UseBasicParsing "$baseUrl/health" -TimeoutSec 5
    $admin = Invoke-WebRequest -UseBasicParsing "$baseUrl/admin" -TimeoutSec 5

    if ($health.StatusCode -ne 200 -or $admin.StatusCode -ne 200) {
        throw 'Service health or local admin check failed.'
    }
    if ($admin.Content -notmatch 'name="trace_code"') {
        throw 'Local admin form is missing the trace_code input.'
    }
    if ($admin.Content -match 'name="api_key"') {
        throw 'Local admin form unexpectedly asks for an API key.'
    }

    Write-Host "PASS: health and local admin are ready on port $Port."
}
catch {
    Write-Error "FAIL: $($_.Exception.Message)"
    exit 1
}
