@echo off
chcp 65001 >nul
where.exe pwsh.exe >nul 2>&1
if errorlevel 1 (
    echo [ERROR] PowerShell 7 is required. Install it and add pwsh.exe to PATH.
    exit /b 1
)
pwsh.exe -NoLogo -NoProfile -File "%~dp0scripts\deploy.ps1" %*
exit /b %ERRORLEVEL%
