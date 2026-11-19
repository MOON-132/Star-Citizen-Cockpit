# SC Cockpit fuer den aktuellen Windows-Benutzer installieren.
# Kein Admin noetig. Python 3 muss vorhanden sein oder wird angeboten.
$ErrorActionPreference = "Stop"
$Source = Split-Path -Parent $PSCommandPath
if (-not (Test-Path (Join-Path $Source "server.py"))) {
  Write-Error "server.py liegt nicht neben install.ps1."
}

$Root = Join-Path $env:LOCALAPPDATA "SC Cockpit"
$App = Join-Path $Root "app"
$UserDash = Join-Path $Root "user\dashboards"
New-Item -ItemType Directory -Force -Path $UserDash | Out-Null
if (Test-Path $App) { Remove-Item -LiteralPath $App -Recurse -Force }
New-Item -ItemType Directory -Force -Path $App | Out-Null

$names = @(
  "server.py", "index.html", "LICENSE", "README.md",
  "config.local.json.example", "install.ps1", "install.bat",
  "uninstall.ps1", "uninstall.bat"
)
foreach ($name in $names) {
  Copy-Item -LiteralPath (Join-Path $Source $name) -Destination (Join-Path $App $name) -Force
}
Copy-Item -LiteralPath (Join-Path $Source "dashboards") -Destination (Join-Path $App "dashboards") -Recurse -Force
Copy-Item -LiteralPath (Join-Path $Source "docs") -Destination (Join-Path $App "docs") -Recurse -Force
$copiedLocal = Join-Path $App "dashboards\local"
if (Test-Path $copiedLocal) { Remove-Item -LiteralPath $copiedLocal -Recurse -Force }

$launcher = @"
@echo off
chcp 65001 >nul
cd /d "%~dp0"
set "SC_COCKPIT_HOME=%LOCALAPPDATA%\SC Cockpit\user"
title SC Cockpit
where py >nul 2>&1 && (py -3 server.py) || (python server.py)
echo.
echo Server beendet.
pause
"@
$admin = @"
@echo off
powershell -NoProfile -ExecutionPolicy Bypass -Command "Start-Process -FilePath '%~dp0Starten.bat' -Verb RunAs"
"@
Set-Content -LiteralPath (Join-Path $App "Starten.bat") -Value $launcher -Encoding ASCII
Set-Content -LiteralPath (Join-Path $App "Als-Administrator.bat") -Value $admin -Encoding ASCII

$wsh = New-Object -ComObject WScript.Shell
function Make-Link($path, $target, $desc) {
  $lnk = $wsh.CreateShortcut($path)
  $lnk.TargetPath = $target
  $lnk.WorkingDirectory = $App
  $lnk.Description = $desc
  $lnk.Save()
}
$folder = Join-Path ([Environment]::GetFolderPath("Programs")) "SC Cockpit"
New-Item -ItemType Directory -Force -Path $folder | Out-Null
Make-Link (Join-Path $folder "SC Cockpit.lnk") (Join-Path $App "Starten.bat") "Star Citizen Dashboard starten"
Make-Link (Join-Path $folder "SC Cockpit (Administrator).lnk") (Join-Path $App "Als-Administrator.bat") "SC Cockpit mit Administratorrechten"
Make-Link (Join-Path $folder "Deinstallieren.lnk") (Join-Path $App "uninstall.bat") "SC Cockpit entfernen"
Make-Link (Join-Path ([Environment]::GetFolderPath("Desktop")) "SC Cockpit.lnk") (Join-Path $App "Starten.bat") "Star Citizen Dashboard starten"

Write-Host ""
Write-Host "Installiert: $App"
Write-Host "Eigene Dashboards: $UserDash"
Write-Host "Start: Desktop oder Startmenue, Ordner SC Cockpit."
Write-Host "Fenster offen lassen, URL im Browser von Tablet, Handy oder Laptop oeffnen."
$py = Get-Command py -ErrorAction SilentlyContinue
$python = Get-Command python -ErrorAction SilentlyContinue
if (-not $py -and -not $python) {
  Write-Host ""
  Write-Host "Python 3 fehlt. Installer von python.org, dabei 'Add python.exe to PATH' ankreuzen."
  if (Get-Command winget -ErrorAction SilentlyContinue) {
    $answer = Read-Host "Python jetzt mit winget installieren? (j/n)"
    if ($answer -eq "j") {
      winget install -e --id Python.Python.3.12 --scope user --accept-package-agreements --accept-source-agreements
    }
  }
}
