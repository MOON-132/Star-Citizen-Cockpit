@echo off
chcp 65001 >nul
cd /d "%~dp0"
title SC Cockpit

if /i not "%~1"=="--console" (
  if exist "%~dp0SC-Cockpit-Setup.exe" (
    powershell -NoProfile -NonInteractive -ExecutionPolicy Bypass -Command "Unblock-File -LiteralPath '%~dp0SC-Cockpit-Setup.exe' -ErrorAction SilentlyContinue" >nul 2>&1
    start "" "%~dp0SC-Cockpit-Setup.exe" --launcher
    exit /b 0
  )
)

echo.
echo  SC Cockpit (Portabler Konsolen-Modus)
echo  -------------------------------------
echo  Portabler Start direkt aus diesem Ordner ohne Installation.
echo  Browser: http://127.0.0.1:8765/
echo  Beenden mit Strg+C.
echo.
where py >nul 2>&1 && (
  py -3 server.py
) else (
  python server.py
)
echo.
echo Server beendet.
pause

