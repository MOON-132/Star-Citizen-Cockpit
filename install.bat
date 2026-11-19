@echo off
cd /d "%~dp0"
if exist "%~dp0SC-Cockpit-Setup.exe" (
  powershell -NoProfile -NonInteractive -ExecutionPolicy Bypass -Command "Unblock-File -LiteralPath '%~dp0SC-Cockpit-Setup.exe' -ErrorAction SilentlyContinue" >nul 2>&1
  start "" "%~dp0SC-Cockpit-Setup.exe"
  exit /b 0
)
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0install.ps1"
echo.
pause


