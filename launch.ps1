<#
.SYNOPSIS
    Launches the complete ASCII-Craft environment.
.DESCRIPTION
    This script orchestrates the startup of the Relay Server, Game Server,
    and the Vite Frontend. It uses Windows Terminal (wt.exe) with split panes
    if available, otherwise it falls back to launching separate console windows.
#>

$ErrorActionPreference = "Stop"

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "   Launching ASCII-Craft Environment     " -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan

$baseDir = $PSScriptRoot
if (-not $baseDir) {
    $baseDir = (Get-Location).Path
}

$serverDir = Join-Path $baseDir "server"
$frontendDir = Join-Path $baseDir "frontend"

# Check dependencies
$missingDeps = @()
if (-not (Get-Command "go" -ErrorAction SilentlyContinue)) { $missingDeps += "Go" }
if (-not (Get-Command "npm" -ErrorAction SilentlyContinue)) { $missingDeps += "Node.js (npm)" }

if ($missingDeps.Count -gt 0) {
    Write-Host "Missing required dependencies: $($missingDeps -join ', ')" -ForegroundColor Red
    Write-Host "Please install them and ensure they are in your PATH." -ForegroundColor Yellow
    exit 1
}

# Ensure npm packages are installed
Write-Host "Checking frontend dependencies..." -ForegroundColor Green
if (-not (Test-Path (Join-Path $frontendDir "node_modules"))) {
    Write-Host "Running 'npm install' in frontend folder (this might take a moment)..." -ForegroundColor Yellow
    Start-Process -FilePath "npm.cmd" -ArgumentList "install" -WorkingDirectory $frontendDir -Wait -NoNewWindow
}

$useWT = $false
if (Get-Command "wt.exe" -ErrorAction SilentlyContinue) {
    $useWT = $true
}

if ($useWT) {
    Write-Host "Windows Terminal detected. Launching in split panes..." -ForegroundColor Green
    
    # We will launch wt.exe with:
    # 1. New tab for the Relay
    # 2. Split pane for Game Server
    # 3. Split pane for Vite Frontend
    $wtArgs = @(
        "new-tab", "--title", "`"Relay Server`"", "-d", "`"$serverDir`"", "cmd", "/k", "go run .\relay"
        ";", "split-pane", "--title", "`"Game Server`"", "-d", "`"$serverDir`"", "cmd", "/k", "go run ."
        ";", "split-pane", "--title", "`"Vite Frontend`"", "--horizontal", "-d", "`"$frontendDir`"", "cmd", "/k", "npm run dev"
    )
    
    Start-Process -FilePath "wt.exe" -ArgumentList $wtArgs
}
else {
    Write-Host "Windows Terminal not found. Launching separate windows..." -ForegroundColor Yellow
    
    # Fallback to separate cmd windows
    Start-Process -FilePath "cmd.exe" -ArgumentList "/k title Relay Server & go run .\relay" -WorkingDirectory $serverDir
    Start-Sleep -Seconds 2
    Start-Process -FilePath "cmd.exe" -ArgumentList "/k title Game Server & go run ." -WorkingDirectory $serverDir
    Start-Process -FilePath "cmd.exe" -ArgumentList "/k title Vite Frontend & npm run dev" -WorkingDirectory $frontendDir
}

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "All services have been launched!" -ForegroundColor Green
Write-Host "Frontend will be available at: http://localhost:5173" -ForegroundColor Yellow
Write-Host "To stop the services, simply close the opened terminal(s)." -ForegroundColor Yellow
Write-Host "=========================================" -ForegroundColor Cyan
