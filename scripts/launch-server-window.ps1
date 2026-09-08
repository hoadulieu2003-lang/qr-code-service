[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string]$StartScript,
    [Parameter(Mandatory)]
    [string]$LockDir,
    [string]$GoPath
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

if (-not (Test-Path -LiteralPath $StartScript -PathType Leaf)) {
    exit 1
}
if (-not (Test-Path -LiteralPath $LockDir -PathType Container)) {
    exit 1
}
if ($GoPath -and -not (Test-Path -LiteralPath $GoPath -PathType Leaf)) {
    exit 1
}

$escapedStartScript = $StartScript.Replace("'", "''")
$command = '$Host.UI.RawUI.WindowTitle = ''QR Server''; & ''' + $escapedStartScript + ''''
if ($GoPath) {
    $command += ' -GoPath ''' + $GoPath.Replace("'", "''") + ''''
}
$encodedCommand = [Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes($command))
$process = Start-Process -FilePath 'powershell.exe' `
    -ArgumentList "-NoExit -NoProfile -ExecutionPolicy Bypass -EncodedCommand $encodedCommand" `
    -PassThru

Set-Content -LiteralPath (Join-Path $LockDir 'server.pid') -Value $process.Id -NoNewline
