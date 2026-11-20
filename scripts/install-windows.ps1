#Requires -Version 5.1
<#
.SYNOPSIS
  Install Parla Viva v0.2.0 on Windows, back up & reuse existing Parla accounts
  and deltachat-rpc-server.exe, and launch parla-viva.exe.
#>
$ErrorActionPreference = "Stop"

$Version    = "v0.2.0"
$ZipUrl     = "https://github.com/acam-infra/parla-viva/releases/download/$Version/parla-viva-windows-amd64.zip"
$InstallDir = Join-Path $env:LOCALAPPDATA "ParlaViva"
$ZipPath    = Join-Path $env:TEMP "parla-viva-$Version.zip"

Write-Host "[1/4] Downloading Parla Viva $Version..." -ForegroundColor Cyan
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
Invoke-WebRequest -Uri $ZipUrl -OutFile $ZipPath
Expand-Archive -Path $ZipPath -DestinationPath $InstallDir -Force
Remove-Item -Force $ZipPath -ErrorAction SilentlyContinue

Write-Host "[2/4] Locating existing Parla accounts & deltachat-rpc-server.exe..." -ForegroundColor Cyan
$AccountCandidates = @(
    $env:DC_ACCOUNTS_PATH,
    (Join-Path $env:LOCALAPPDATA "parla\accounts"),
    (Join-Path $env:APPDATA "parla\accounts"),
    (Join-Path $env:USERPROFILE ".local\share\parla\accounts")
) | Where-Object { $_ -and (Test-Path $_) }

$AccountsDir = $AccountCandidates | Select-Object -First 1
if ($AccountsDir) {
    $Stamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $BackupDir = "$AccountsDir-backup-$Stamp"
    Write-Host "  Found existing Parla accounts: $AccountsDir" -ForegroundColor Green
    Write-Host "  Creating safety copy at: $BackupDir" -ForegroundColor DarkGray
    Copy-Item -Path $AccountsDir -Destination $BackupDir -Recurse -Force
} else {
    $AccountsDir = Join-Path $env:LOCALAPPDATA "parla\accounts"
    New-Item -ItemType Directory -Force -Path $AccountsDir | Out-Null
    Write-Host "  No existing Parla accounts found; initialized fresh store: $AccountsDir" -ForegroundColor Yellow
}

$RpcCandidates = @(
    (Get-Command "deltachat-rpc-server.exe" -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Source),
    (Join-Path $env:LOCALAPPDATA "Programs\Parla\deltachat-rpc-server.exe"),
    (Join-Path $env:LOCALAPPDATA "Parla\deltachat-rpc-server.exe"),
    (Join-Path $env:USERPROFILE ".local\bin\deltachat-rpc-server.exe"),
    (Join-Path $env:USERPROFILE ".cargo\bin\deltachat-rpc-server.exe")
) | Where-Object { $_ -and (Test-Path $_) }

$RpcBin = $RpcCandidates | Select-Object -First 1
if (-not $RpcBin) {
    # Search common Parla extract folders under Downloads/Desktop/Program Files
    $Found = Get-ChildItem -Path $env:USERPROFILE\Downloads, $env:USERPROFILE\Desktop, "C:\Program Files" -Filter "deltachat-rpc-server.exe" -Recurse -Depth 3 -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($Found) { $RpcBin = $Found.FullName }
}

Write-Host "[3/4] Checking running Parla instance (avoiding DB lock)..." -ForegroundColor Cyan
Get-Process -Name "parla" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue

Write-Host "[4/4] Launching Parla Viva..." -ForegroundColor Green
$Exe = Join-Path $InstallDir "parla-viva.exe"
$ArgsList = @("--accounts", $AccountsDir)
if ($RpcBin) {
    Write-Host "  Reusing deltachat-rpc-server: $RpcBin" -ForegroundColor Green
    $ArgsList += @("--rpc", $RpcBin)
} else {
    Write-Host "  deltachat-rpc-server.exe not found; launching in offline preview mode (install via: pip install deltachat-rpc-server)" -ForegroundColor Yellow
}

Start-Process -FilePath $Exe -ArgumentList $ArgsList -WorkingDirectory $InstallDir
Write-Host "Done! Parla Viva launched from $Exe" -ForegroundColor Green
