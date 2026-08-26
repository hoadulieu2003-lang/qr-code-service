[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [ValidatePattern('^[A-Za-z0-9_-]+$')]
    [string]$TraceCode,
    [string]$ProductCode = 'API-DEMO-001',
    [string]$Name = 'Sản phẩm API demo',
    [string]$BatchCode = 'LO-API-001',
    [string]$ManufacturedAt = (Get-Date -Format 'yyyy-MM-dd'),
    [string]$ExpiresAt = ((Get-Date).AddYears(1).ToString('yyyy-MM-dd')),
    [string]$Origin = 'Đắk Lắk, Việt Nam',
    [ValidateSet('verified', 'unverified')]
    [string]$VerificationStatus = 'verified'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Get-DotEnvSettings([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path)) {
        throw 'Missing .env. Copy sample.env to .env and configure it first.'
    }

    $settings = @{}
    foreach ($line in Get-Content -LiteralPath $Path) {
        if ($line -match '^\s*(?<key>[A-Z_]+)\s*=\s*"?(?<value>.*?)"?\s*$') {
            $settings[$matches.key] = $matches.value.Trim('"')
        }
    }
    return $settings
}

function Get-HTTPStatusFromError($ErrorRecord) {
    if ($ErrorRecord.Exception.Response) {
        return [int]$ErrorRecord.Exception.Response.StatusCode
    }
    return 0
}

try {
    $projectRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
    $settings = Get-DotEnvSettings (Join-Path $projectRoot '.env')
    $apiKey = $settings['API_KEY']
    $publicBaseUrl = $settings['PUBLIC_BASE_URL']
    if ([string]::IsNullOrWhiteSpace($apiKey)) {
        throw 'API_KEY is missing from .env.'
    }
    if ([string]::IsNullOrWhiteSpace($publicBaseUrl)) {
        throw 'PUBLIC_BASE_URL is missing from .env.'
    }
    $publicBaseUrl = $publicBaseUrl.TrimEnd('/')

    $product = @{
        trace_code = $TraceCode; product_code = $ProductCode; name = $Name
        batch_code = $BatchCode; manufactured_at = $ManufacturedAt; expires_at = $ExpiresAt
        origin = $Origin; verification_status = $VerificationStatus
    }
    $response = Invoke-RestMethod -Method Post -Uri "$publicBaseUrl/api/products" `
        -Headers @{ 'X-API-Key' = $apiKey } -ContentType 'application/json; charset=utf-8' `
        -Body ($product | ConvertTo-Json)
    $trace = Invoke-WebRequest -UseBasicParsing $response.trace_url -TimeoutSec 10
    if ($trace.StatusCode -ne 200) {
        throw 'Public trace page was not reachable after product creation.'
    }

    $response | Select-Object trace_code, trace_url, qr_url
}
catch {
    $status = Get-HTTPStatusFromError $_
    if ($status -eq 409) {
        Write-Error "Trace code '$TraceCode' already exists. Choose a new TraceCode; the existing product was not overwritten."
    }
    elseif ($status -eq 401) {
        Write-Error 'API authentication failed. Check API_KEY in .env without printing it.'
    }
    else {
        Write-Error "API test failed: $($_.Exception.Message)"
    }
    exit 1
}
