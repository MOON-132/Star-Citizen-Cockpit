@echo off
chcp 65001 >nul
cd /d "%~dp0"
title SC Cockpit
echo.
echo  SC Cockpit
echo  ----------
echo  Fenster offen lassen, waehrend du spielst.
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
