# SC Cockpit entfernen. Eigene Dashboards bleiben, ausser mit -Purge.
$ErrorActionPreference = "Stop"
$Root = Join-Path $env:LOCALAPPDATA "SC Cockpit"
$App = Join-Path $Root "app"
$folder = Join-Path ([Environment]::GetFolderPath("Programs")) "SC Cockpit"
$desktop = Join-Path ([Environment]::GetFolderPath("Desktop")) "SC Cockpit.lnk"
if (Test-Path $folder) { Remove-Item -LiteralPath $folder -Recurse -Force }
if (Test-Path $desktop) { Remove-Item -LiteralPath $desktop -Force }

$purge = $args -contains "-Purge"
if ($purge) {
  $target = $Root
  $message = "Programm und eigene Dashboards werden entfernt."
} else {
  $target = $App
  $message = "Programm entfernt. Eigene Dashboards bleiben in:`n$Root\user"
}
if (Test-Path $target) {
  $cmd = "timeout /t 1 /nobreak >nul & rmdir /s /q `"$target`""
  Start-Process -FilePath "cmd.exe" -ArgumentList "/c $cmd" -WindowStyle Hidden
}
Write-Host $message
if (-not $purge) {
  Write-Host "Alles inklusive Dashboards loeschen: uninstall.bat -Purge"
}
