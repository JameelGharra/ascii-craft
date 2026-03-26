@echo off
:: Launches the ASCII-Craft development environment using the PowerShell orchestrator script.
:: This is provided for convenient double-clicking from Windows File Explorer.
powershell.exe -ExecutionPolicy Bypass -NoProfile -File "%~dp0launch.ps1"
