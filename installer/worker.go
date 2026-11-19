package main

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

//go:embed payload.zip
var embeddedPayload []byte

const launcherBat = `@echo off
chcp 65001 >nul
cd /d "%~dp0"
set "SC_COCKPIT_HOME=%LOCALAPPDATA%\SC Cockpit\user"
title SC Cockpit
where py >nul 2>&1 && (py -3 server.py) || (python server.py)
echo.
echo Server beendet.
pause
`

const adminBat = `@echo off
powershell -NoProfile -ExecutionPolicy Bypass -Command "Start-Process -FilePath '%~dp0Starten.bat' -Verb RunAs"
`

const uninstallRegKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\SCCockpit`

func runHiddenPowerShell(script string) error {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func runInstallWorker() {
	defer func() {
		pPostMessageW.Call(app.Hwnd, WM_APP_DONE, 0, 0)
	}()

	app.mu.Lock()
	app.Logs = nil
	app.InstallErr = ""
	app.InstallOK = false
	app.mu.Unlock()

	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		home, _ := os.UserHomeDir()
		localAppData = filepath.Join(home, "AppData", "Local")
	}
	rootDir := filepath.Join(localAppData, "SC Cockpit")
	appDir := filepath.Join(rootDir, "app")
	userDashDir := filepath.Join(rootDir, "user", "dashboards")

	app.setProgress(8, "Initialisiere Benutzerverzeichnis und Telemetrie...")
	time.Sleep(150 * time.Millisecond)

	if err := os.MkdirAll(userDashDir, 0755); err != nil {
		app.InstallErr = fmt.Sprintf("Konnte Benutzerordner nicht anlegen: %v", err)
		return
	}

	app.setProgress(16, "Bereinige vorherige Programmversion in %LOCALAPPDATA%\\SC Cockpit\\app...")
	_ = os.RemoveAll(appDir)
	if err := os.MkdirAll(appDir, 0755); err != nil {
		app.InstallErr = fmt.Sprintf("Konnte Programmordner nicht anlegen: %v", err)
		return
	}
	time.Sleep(150 * time.Millisecond)

	// Extract embedded payload.zip
	zr, err := zip.NewReader(bytes.NewReader(embeddedPayload), int64(len(embeddedPayload)))
	if err != nil {
		app.InstallErr = fmt.Sprintf("Archivfehler im Installer-Payload: %v", err)
		return
	}

	totalFiles := len(zr.File)
	for i, f := range zr.File {
		relPath := filepath.FromSlash(f.Name)
		if strings.Contains(relPath, "..") {
			continue
		}
		targetPath := filepath.Join(appDir, relPath)
		pct := 20 + ((i+1)*40)/maxInt(totalFiles, 1)
		app.setProgress(pct, fmt.Sprintf("Entpacke Modul: %s", f.Name))

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(targetPath, 0755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			app.InstallErr = fmt.Sprintf("Ordnerfehler bei %s: %v", f.Name, err)
			return
		}
		rc, err := f.Open()
		if err != nil {
			app.InstallErr = fmt.Sprintf("Lesefehler bei %s: %v", f.Name, err)
			return
		}
		outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			rc.Close()
			app.InstallErr = fmt.Sprintf("Schreibfehler bei %s: %v", f.Name, err)
			return
		}
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			app.InstallErr = fmt.Sprintf("Kopierfehler bei %s: %v", f.Name, err)
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	app.setProgress(65, "Generiere Start-Skripte (Starten.bat & Als-Administrator.bat)...")
	launcherWin := strings.ReplaceAll(launcherBat, "\n", "\r\n")
	adminWin := strings.ReplaceAll(adminBat, "\n", "\r\n")
	_ = os.WriteFile(filepath.Join(appDir, "Starten.bat"), []byte(launcherWin), 0644)
	_ = os.WriteFile(filepath.Join(appDir, "Als-Administrator.bat"), []byte(adminWin), 0644)

	// Copy running installer exe into appDir as SC-Cockpit-Setup.exe so it acts as graphical uninstaller
	app.setProgress(72, "Registriere grafischen Uninstaller & System-Icon...")
	selfExe, err := os.Executable()
	uninstallerExe := filepath.Join(appDir, "SC-Cockpit-Setup.exe")
	if err == nil && selfExe != "" {
		if !strings.EqualFold(filepath.Clean(selfExe), filepath.Clean(uninstallerExe)) {
			if data, err := os.ReadFile(selfExe); err == nil {
				_ = os.WriteFile(uninstallerExe, data, 0755)
			}
		}
	}
	time.Sleep(150 * time.Millisecond)

	// Create shortcuts and Windows Uninstall registry entry via PowerShell
	app.setProgress(82, "Erstelle Verknüpfungen & Windows-App-Registrierung...")
	iconPath := filepath.Join(appDir, "sc-cockpit.ico")
	if _, err := os.Stat(iconPath); err != nil {
		iconPath = uninstallerExe + ",0"
	}

	var psBuilder strings.Builder
	psBuilder.WriteString("$ErrorActionPreference = 'Stop'\n")
	psBuilder.WriteString(fmt.Sprintf("$App = '%s'\n", escapePS(appDir)))
	psBuilder.WriteString(fmt.Sprintf("$Icon = '%s'\n", escapePS(iconPath)))
	psBuilder.WriteString("$wsh = New-Object -ComObject WScript.Shell\n")
	psBuilder.WriteString("function Make-Link($path, $target, $argsStr, $desc) {\n")
	psBuilder.WriteString("  $lnk = $wsh.CreateShortcut($path)\n")
	psBuilder.WriteString("  $lnk.TargetPath = $target\n")
	psBuilder.WriteString("  if ($argsStr) { $lnk.Arguments = $argsStr }\n")
	psBuilder.WriteString("  $lnk.WorkingDirectory = $App\n")
	psBuilder.WriteString("  $lnk.Description = $desc\n")
	psBuilder.WriteString("  $lnk.IconLocation = $Icon\n")
	psBuilder.WriteString("  $lnk.Save()\n")
	psBuilder.WriteString("}\n")

	if app.OptStartMenu {
		psBuilder.WriteString("$folder = Join-Path ([Environment]::GetFolderPath('Programs')) 'SC Cockpit'\n")
		psBuilder.WriteString("New-Item -ItemType Directory -Force -Path $folder | Out-Null\n")
		psBuilder.WriteString("Make-Link (Join-Path $folder 'SC Cockpit.lnk') (Join-Path $App 'Starten.bat') '' 'Star Citizen Dashboard starten'\n")
		psBuilder.WriteString("Make-Link (Join-Path $folder 'SC Cockpit (Administrator).lnk') (Join-Path $App 'Als-Administrator.bat') '' 'SC Cockpit mit Administratorrechten'\n")
		psBuilder.WriteString("Make-Link (Join-Path $folder 'Deinstallieren.lnk') (Join-Path $App 'SC-Cockpit-Setup.exe') '--uninstall' 'SC Cockpit entfernen'\n")
	}

	if app.OptDesktop {
		psBuilder.WriteString("Make-Link (Join-Path ([Environment]::GetFolderPath('Desktop')) 'SC Cockpit.lnk') (Join-Path $App 'Starten.bat') '' 'Star Citizen Dashboard starten'\n")
	}

	// Register in HKCU Uninstall so it shows up in Windows Installed Apps
	psBuilder.WriteString(fmt.Sprintf("$regPath = 'HKCU:\\%s'\n", uninstallRegKey))
	psBuilder.WriteString("New-Item -Path $regPath -Force | Out-Null\n")
	psBuilder.WriteString("Set-ItemProperty -Path $regPath -Name 'DisplayName' -Value 'SC Cockpit'\n")
	psBuilder.WriteString(fmt.Sprintf("Set-ItemProperty -Path $regPath -Name 'DisplayVersion' -Value '%s'\n", AppVersion))
	psBuilder.WriteString("Set-ItemProperty -Path $regPath -Name 'Publisher' -Value 'MOON-132 // SC Cockpit'\n")
	psBuilder.WriteString("Set-ItemProperty -Path $regPath -Name 'InstallLocation' -Value $App\n")
	psBuilder.WriteString("Set-ItemProperty -Path $regPath -Name 'DisplayIcon' -Value $Icon\n")
	psBuilder.WriteString("Set-ItemProperty -Path $regPath -Name 'UninstallString' -Value ('\"' + (Join-Path $App 'SC-Cockpit-Setup.exe') + '\" --uninstall')\n")
	psBuilder.WriteString("Set-ItemProperty -Path $regPath -Name 'NoModify' -Value 1 -Type DWord\n")
	psBuilder.WriteString("Set-ItemProperty -Path $regPath -Name 'NoRepair' -Value 1 -Type DWord\n")

	if err := runHiddenPowerShell(psBuilder.String()); err != nil {
		app.addLog("Hinweis bei Verknüpfungen: " + err.Error())
	}

	if !app.PythonFound && app.WingetFound && app.OptWingetPy {
		app.setProgress(90, "Installiere Python 3.12 via winget (bitte warten)...")
		cmd := exec.Command("winget", "install", "-e", "--id", "Python.Python.3.12", "--scope", "user", "--accept-package-agreements", "--accept-source-agreements")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := cmd.Run(); err != nil {
			app.addLog("Winget Python-Installation konnte nicht automatisch abgeschlossen werden.")
		} else {
			app.PythonFound = true
			app.PythonVer = "Python 3.12"
			app.addLog("Python 3.12 erfolgreich installiert.")
		}
	}

	app.setProgress(100, "Deployment abgeschlossen. Alle Systeme online.")
	app.InstallOK = true
	app.ExistingInst = true
	time.Sleep(250 * time.Millisecond)
}

func runUninstallWorker() {
	defer func() {
		pPostMessageW.Call(app.Hwnd, WM_APP_DONE, 0, 0)
	}()

	app.mu.Lock()
	app.Logs = nil
	app.InstallErr = ""
	app.mu.Unlock()

	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		home, _ := os.UserHomeDir()
		localAppData = filepath.Join(home, "AppData", "Local")
	}
	rootDir := filepath.Join(localAppData, "SC Cockpit")
	appDir := filepath.Join(rootDir, "app")

	app.setProgress(25, "Entferne Desktop- und Startmenü-Verknüpfungen sowie Windows-Registrierung...")
	var ps strings.Builder
	ps.WriteString("$ErrorActionPreference = 'SilentlyContinue'\n")
	ps.WriteString("$folder = Join-Path ([Environment]::GetFolderPath('Programs')) 'SC Cockpit'\n")
	ps.WriteString("$desktop = Join-Path ([Environment]::GetFolderPath('Desktop')) 'SC Cockpit.lnk'\n")
	ps.WriteString("if (Test-Path $folder) { Remove-Item -LiteralPath $folder -Recurse -Force }\n")
	ps.WriteString("if (Test-Path $desktop) { Remove-Item -LiteralPath $desktop -Force }\n")
	ps.WriteString(fmt.Sprintf("$regPath = 'HKCU:\\%s'\n", uninstallRegKey))
	ps.WriteString("if (Test-Path $regPath) { Remove-Item -LiteralPath $regPath -Recurse -Force }\n")
	_ = runHiddenPowerShell(ps.String())
	time.Sleep(250 * time.Millisecond)

	target := appDir
	if app.OptPurgeData {
		target = rootDir
		app.setProgress(65, "Entferne Programmordner und alle eigenen Dashboards (-Purge)...")
	} else {
		app.setProgress(65, "Entferne Programmordner (eigene Dashboards unter user bleiben erhalten)...")
	}

	// Attempt direct remove first; also schedule background cmd cleanup in case we are running from inside appDir
	_ = os.RemoveAll(target)
	if _, err := os.Stat(target); err == nil {
		cleanupCmd := fmt.Sprintf("timeout /t 2 /nobreak >nul & rmdir /s /q \"%s\"", target)
		cmd := exec.Command("cmd.exe", "/c", cleanupCmd)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = cmd.Start()
	}

	time.Sleep(300 * time.Millisecond)
	app.setProgress(100, "Deinstallation vollständig abgeschlossen.")
	app.ExistingInst = false
	time.Sleep(200 * time.Millisecond)
}

func escapePS(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
