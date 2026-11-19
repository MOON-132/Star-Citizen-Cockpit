package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

//go:embed payload.zip
var embeddedPayload []byte

const uninstallRegKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\SCCockpit`

func initEnvironment() {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		home, _ := os.UserHomeDir()
		localAppData = filepath.Join(home, "AppData", "Local")
	}
	app.RootDir = filepath.Join(localAppData, "SC Cockpit")
	app.AppDir = filepath.Join(app.RootDir, "app")
	app.UserDir = filepath.Join(app.RootDir, "user")
	app.UserDashDir = filepath.Join(app.UserDir, "dashboards")

	if _, err := os.Stat(filepath.Join(app.AppDir, "server.py")); err == nil {
		app.ExistingInst = true
	}

	// Detect if running from inside the installed app directory or with --launcher
	selfExe, _ := os.Executable()
	forceInstaller := false
	forceLauncher := false
	for _, arg := range os.Args[1:] {
		lower := strings.ToLower(arg)
		if lower == "--uninstall" || lower == "/uninstall" || lower == "-uninstall" {
			app.Mode = ModeInstaller
			app.IsUninstall = true
			app.InstPage = InstPageUninstall
			forceInstaller = true
		} else if lower == "-purge" || lower == "--purge" {
			app.OptPurgeData = true
		} else if lower == "--launcher" {
			forceLauncher = true
		} else if lower == "--install" {
			forceInstaller = true
		}
	}

	if !forceInstaller {
		exeDir := ""
		if selfExe != "" {
			exeDir = filepath.Dir(selfExe)
		}
		if forceLauncher {
			app.Mode = ModeLauncher
			enablePortableOrInstalledMode()
		} else if selfExe != "" && strings.EqualFold(filepath.Clean(exeDir), filepath.Clean(app.AppDir)) {
			app.Mode = ModeLauncher
		} else {
			app.Mode = ModeInstaller
		}
	}

	// Check Admin rights
	pIsUserAnAdmin := shell32.NewProc("IsUserAnAdmin")
	if r, _, _ := pIsUserAnAdmin.Call(); r != 0 {
		app.IsAdmin = true
	}

	// Check Python 3
	for _, cmdName := range []string{"py", "python"} {
		if path, err := exec.LookPath(cmdName); err == nil && path != "" {
			cmd := exec.Command(cmdName, "--version")
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			if out, err := cmd.CombinedOutput(); err == nil {
				ver := strings.TrimSpace(string(out))
				if strings.HasPrefix(ver, "Python 3") {
					app.PythonFound = true
					app.PythonCmd = cmdName
					app.PythonVer = ver
					break
				}
			}
		}
	}

	if _, err := exec.LookPath("winget"); err == nil {
		app.WingetFound = true
		if !app.PythonFound {
			app.OptWingetPy = true
		}
	}

	app.LanIP = detectPrimaryLANIP()
	loadConfigPort()
}

func enablePortableOrInstalledMode() {
	selfExe, err := os.Executable()
	if err != nil || selfExe == "" {
		return
	}
	exeDir := filepath.Dir(selfExe)
	if !strings.EqualFold(filepath.Clean(exeDir), filepath.Clean(app.AppDir)) {
		if _, err := os.Stat(filepath.Join(exeDir, "server.py")); err == nil {
			app.AppDir = exeDir
			app.UserDir = filepath.Join(exeDir, "dashboards", "local")
			app.UserDashDir = filepath.Join(exeDir, "dashboards", "local")
		}
	}
}

func detectPrimaryLANIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
			ip := addr.IP.String()
			if !strings.HasPrefix(ip, "127.") && !strings.HasPrefix(ip, "169.254.") {
				return ip
			}
		}
	}
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
				ip := ipnet.IP.String()
				if !strings.HasPrefix(ip, "169.254.") {
					return ip
				}
			}
		}
	}
	return "127.0.0.1"
}

func loadConfigPort() {
	cfgPath := filepath.Join(app.UserDir, "config.local.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return
	}
	var raw map[string]interface{}
	if json.Unmarshal(data, &raw) == nil {
		if p, ok := raw["port"].(float64); ok && p >= 1024 && p <= 65535 {
			app.Port = int(p)
		}
	}
}

func saveConfigPort(port int) {
	_ = os.MkdirAll(app.UserDir, 0755)
	cfgPath := filepath.Join(app.UserDir, "config.local.json")
	raw := map[string]interface{}{
		"port":           port,
		"token":          "",
		"hold_timeout_s": 4,
	}
	if data, err := os.ReadFile(cfgPath); err == nil {
		_ = json.Unmarshal(data, &raw)
	}
	raw["port"] = port
	if out, err := json.MarshalIndent(raw, "", "  "); err == nil {
		_ = os.WriteFile(cfgPath, append(out, '\n'), 0644)
	}
}

// ============================================================================
// SERVER PROCESS MANAGEMENT (NO BLACK CMD WINDOW!)
// ============================================================================
func appendServerLog(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	app.mu.Lock()
	ts := time.Now().Format("15:04:05")
	app.ServerLogs = append(app.ServerLogs, fmt.Sprintf("[%s]  %s", ts, line))
	if len(app.ServerLogs) > 30 {
		app.ServerLogs = app.ServerLogs[len(app.ServerLogs)-30:]
	}
	app.mu.Unlock()
	if app.Hwnd != 0 {
		pPostMessageW.Call(app.Hwnd, WM_APP_REFRESH, 0, 0)
	}
}

func startServer() {
	if app.ServerRunning {
		return
	}
	serverPy := filepath.Join(app.AppDir, "server.py")
	if _, err := os.Stat(serverPy); err != nil {
		appendServerLog("Fehler: server.py nicht gefunden in " + app.AppDir)
		return
	}

	pyExe := app.PythonCmd
	var args []string
	if pyExe == "py" {
		args = []string{"-3", "-u", serverPy}
	} else if pyExe != "" {
		args = []string{"-u", serverPy}
	} else {
		pyExe = "python"
		args = []string{"-u", serverPy}
	}

	cmd := exec.Command(pyExe, args...)
	cmd.Dir = app.AppDir
	cmd.Env = append(os.Environ(), "SC_COCKPIT_HOME="+app.UserDir, "PYTHONIOENCODING=utf-8")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		appendServerLog("Fehler beim Starten: " + err.Error())
		return
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		appendServerLog("Python-Server konnte nicht gestartet werden: " + err.Error())
		return
	}

	app.ServerCmd = cmd.Process
	app.ServerRunning = true
	app.LanIP = detectPrimaryLANIP()
	appendServerLog(fmt.Sprintf("Server gestartet auf http://%s:%d/", app.LanIP, app.Port))

	go func(r io.Reader, c *exec.Cmd) {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			text := scanner.Text()
			// Filter noisy console startup banner lines
			if strings.HasPrefix(text, "Dieses Fenster") || strings.HasPrefix(text, "Strg+C") || strings.HasPrefix(text, "Gerät:") {
				continue
			}
			appendServerLog(text)
		}
		_ = c.Wait()
		app.mu.Lock()
		if app.ServerCmd != nil && c.Process != nil && app.ServerCmd.Pid == c.Process.Pid {
			app.ServerRunning = false
			app.ServerCmd = nil
		}
		app.mu.Unlock()
		appendServerLog("Server gestoppt.")
	}(stdout, cmd)
}

func stopServer() {
	app.mu.Lock()
	proc := app.ServerCmd
	app.ServerCmd = nil
	app.ServerRunning = false
	app.mu.Unlock()

	if proc != nil {
		_ = proc.Kill()
	}
}

// ============================================================================
// DASHBOARD CATALOG & MANAGEMENT
// ============================================================================
func reloadDashboards() {
	index := make(map[string]int)
	var list []DashboardInfo

	readDir := func(dir, source string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
				continue
			}
			id := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
			if id == "state" || id == "local" {
				continue
			}
			fullPath := filepath.Join(dir, e.Name())
			data, err := os.ReadFile(fullPath)
			if err != nil {
				continue
			}
			var parsed struct {
				ID    string     `json:"id"`
				Title string     `json:"title"`
				Pages []DashPage `json:"pages"`
			}
			if json.Unmarshal(data, &parsed) != nil || len(parsed.Pages) == 0 {
				continue
			}
			if parsed.ID != "" {
				id = parsed.ID
			}
			title := parsed.Title
			if title == "" {
				title = id
			}
			btnCount := 0
			for _, p := range parsed.Pages {
				for _, g := range p.Groups {
					btnCount += len(g.Buttons)
				}
			}
			info := DashboardInfo{
				ID:          id,
				Title:       title,
				Source:      source,
				Path:        fullPath,
				PageCount:   len(parsed.Pages),
				ButtonCount: btnCount,
				Pages:       parsed.Pages,
			}
			if existingIdx, ok := index[id]; ok {
				list[existingIdx] = info
			} else {
				index[id] = len(list)
				list = append(list, info)
			}
		}
	}

	appDash := filepath.Join(app.AppDir, "dashboards")
	if _, err := os.Stat(appDash); err != nil {
		if selfExe, err2 := os.Executable(); err2 == nil {
			cand := filepath.Join(filepath.Dir(selfExe), "dashboards")
			if _, err3 := os.Stat(cand); err3 == nil {
				appDash = cand
			}
		}
	}
	readDir(appDash, "builtin")
	readDir(app.UserDashDir, "local")

	app.Dashboards = list

	// Read active dashboard from state.json
	active := "standard"
	stateFile := filepath.Join(app.UserDir, "state.json")
	if data, err := os.ReadFile(stateFile); err == nil {
		var st struct {
			Active string `json:"active"`
		}
		if json.Unmarshal(data, &st) == nil && st.Active != "" {
			active = st.Active
		}
	}
	found := false
	for _, d := range list {
		if d.ID == active {
			found = true
			break
		}
	}
	if !found && len(list) > 0 {
		active = list[0].ID
	}
	app.ActiveDash = active
}

func setActiveDashboard(id string) {
	_ = os.MkdirAll(app.UserDir, 0755)
	stateFile := filepath.Join(app.UserDir, "state.json")
	out, _ := json.MarshalIndent(map[string]string{"active": id}, "", "  ")
	_ = os.WriteFile(stateFile, append(out, '\n'), 0644)
	app.ActiveDash = id
}

func cycleActiveDashboard(dir int) {
	if len(app.Dashboards) == 0 {
		return
	}
	curIdx := 0
	for i, d := range app.Dashboards {
		if d.ID == app.ActiveDash {
			curIdx = i
			break
		}
	}
	nextIdx := (curIdx + dir + len(app.Dashboards)) % len(app.Dashboards)
	setActiveDashboard(app.Dashboards[nextIdx].ID)
	app.LivePageIdx = 0
}

func triggerInAppButton(btn DashButton) {
	if len(btn.Keys) == 0 {
		return
	}
	combo := strings.ToUpper(strings.Join(btn.Keys, "+"))
	app.LastActionTxt = fmt.Sprintf("%s (%s)", btn.Label, combo)
	app.ToastMsg = fmt.Sprintf("SHARD AUSGELÖST:  %s  ►  [%s]", strings.ToUpper(btn.Label), combo)

	if !app.ServerRunning {
		startServer()
	}

	go func(b DashButton) {
		// Find Star Citizen window if running so keybind goes directly to the game even when clicking the desktop app!
		pFindWindowW := user32.NewProc("FindWindowW")
		pSetForegroundWindow := user32.NewProc("SetForegroundWindow")
		for _, title := range []string{"Star Citizen", "StarCitizen"} {
			if hwndSC, _, _ := pFindWindowW.Call(0, u16ptr(title)); hwndSC != 0 {
				pSetForegroundWindow.Call(hwndSC)
				time.Sleep(35 * time.Millisecond)
				break
			}
		}

		mode := b.Mode
		if mode == "" || mode == "toggle" {
			mode = "tap"
		}
		payload, _ := json.Marshal(map[string]interface{}{
			"id":   b.ID,
			"keys": b.Keys,
			"mode": mode,
		})
		url := fmt.Sprintf("http://127.0.0.1:%d/api/press", app.Port)
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Post(url, "application/json", bytes.NewReader(payload))
		if err == nil && resp != nil {
			resp.Body.Close()
		}
	}(btn)
}

func duplicateDashboard(src DashboardInfo) (string, error) {
	_ = os.MkdirAll(app.UserDashDir, 0755)
	raw, err := os.ReadFile(src.Path)
	if err != nil {
		return "", err
	}
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return "", err
	}

	newID := src.ID + "-eigen"
	n := 2
	for {
		if _, err := os.Stat(filepath.Join(app.UserDashDir, newID+".json")); os.IsNotExist(err) {
			break
		}
		newID = fmt.Sprintf("%s-eigen-%d", src.ID, n)
		n++
	}

	data["id"] = newID
	data["title"] = src.Title + " (Kopie)"
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(app.UserDashDir, newID+".json"), append(out, '\n'), 0644); err != nil {
		return "", err
	}
	setActiveDashboard(newID)
	reloadDashboards()
	return newID, nil
}

func createNewDashboard() (string, error) {
	_ = os.MkdirAll(app.UserDashDir, 0755)
	newID := "eigenes-schiff"
	n := 2
	for {
		if _, err := os.Stat(filepath.Join(app.UserDashDir, newID+".json")); os.IsNotExist(err) {
			break
		}
		newID = fmt.Sprintf("eigenes-schiff-%d", n)
		n++
	}

	template := map[string]interface{}{
		"id":         newID,
		"title":      fmt.Sprintf("Eigenes Schiff %d", n-1),
		"background": "preset:aegis",
		"bg_opacity": 0.40,
		"pages": []map[string]interface{}{
			{
				"id":    "flug",
				"label": "Cockpit & Systeme",
				"groups": []map[string]interface{}{
					{
						"title": "Hauptsysteme (Baukasten)",
						"buttons": []map[string]interface{}{
							{
								"id":    "flight-ready",
								"sc":    "v_flightready",
								"label": "Flight Ready",
								"hint":  "R-Alt + R",
								"keys":  []string{"ralt", "r"},
								"mode":  "tap",
								"color": "green",
								"wide":  true,
							},
							{
								"id":    "gear",
								"sc":    "v_toggle_landing_system",
								"label": "Fahrwerk",
								"hint":  "N",
								"keys":  []string{"n"},
								"mode":  "toggle",
								"color": "cyan",
							},
							{
								"id":    "lights",
								"sc":    "v_lights",
								"label": "Licht",
								"hint":  "L",
								"keys":  []string{"l"},
								"mode":  "toggle",
								"color": "amber",
							},
							{
								"id":    "live-pwr",
								"kind":  "power",
								"label": "Energie-Dreieck",
								"hint":  "F5 / F6 / F7 / F8",
								"keys":  []string{"f8"},
								"mode":  "tap",
								"color": "amber",
								"wide":  true,
							},
							{
								"id":    "live-shd",
								"kind":  "shield",
								"label": "Schild-Kompass",
								"hint":  "Sektoren",
								"keys":  []string{"num5"},
								"mode":  "tap",
								"color": "cyan",
								"wide":  true,
							},
						},
					},
				},
			},
		},
	}
	out, _ := json.MarshalIndent(template, "", "  ")
	if err := os.WriteFile(filepath.Join(app.UserDashDir, newID+".json"), append(out, '\n'), 0644); err != nil {
		return "", err
	}
	setActiveDashboard(newID)
	reloadDashboards()
	return newID, nil
}

func syncStarCitizenActionmaps() string {
	candidates := []string{
		`C:\Program Files\Roberts Space Industries\StarCitizen\LIVE\user\client\0\Profiles\default\actionmaps.xml`,
		`C:\Program Files\Roberts Space Industries\StarCitizen\LIVE\USER\Client\0\Profiles\default\actionmaps.xml`,
		`D:\Program Files\Roberts Space Industries\StarCitizen\LIVE\user\client\0\Profiles\default\actionmaps.xml`,
		`D:\Roberts Space Industries\StarCitizen\LIVE\user\client\0\Profiles\default\actionmaps.xml`,
		`E:\Roberts Space Industries\StarCitizen\LIVE\user\client\0\Profiles\default\actionmaps.xml`,
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return "Star Citizen actionmaps.xml gefunden: " + p + " (Im Browser-Baukasten synchronisierbar)"
		}
	}
	return "Hinweis: Keine lokale actionmaps.xml im Standardpfad gefunden — Standard-Presets (SC 4.x) sind aktiv."
}

// ============================================================================
// INSTALLER & UNINSTALLER WORKER
// ============================================================================
func runHiddenPowerShell(script string) error {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (a *AppState) setProgress(pct int, msg string) {
	a.mu.Lock()
	a.ProgressPct = pct
	a.StatusLine = msg
	a.mu.Unlock()
	if a.Hwnd != 0 {
		pPostMessageW.Call(a.Hwnd, WM_APP_REFRESH, 0, 0)
	}
}

func createShortcutsAndRegistry() {
	exeTarget := filepath.Join(app.AppDir, "SC-Cockpit.exe")
	iconPath := filepath.Join(app.AppDir, "sc-cockpit.ico")
	if _, err := os.Stat(iconPath); err != nil {
		iconPath = exeTarget + ",0"
	}

	adminBat := `@echo off
powershell -NoProfile -ExecutionPolicy Bypass -Command "Start-Process -FilePath '%~dp0SC-Cockpit.exe' -ArgumentList '--launcher' -Verb RunAs"
`
	_ = os.WriteFile(filepath.Join(app.AppDir, "Als-Administrator.bat"), []byte(strings.ReplaceAll(adminBat, "\n", "\r\n")), 0644)

	launcherBat := `@echo off
start "" "%~dp0SC-Cockpit.exe" --launcher
`
	_ = os.WriteFile(filepath.Join(app.AppDir, "Starten.bat"), []byte(strings.ReplaceAll(launcherBat, "\n", "\r\n")), 0644)

	var ps strings.Builder
	ps.WriteString("$ErrorActionPreference = 'Stop'\n")
	ps.WriteString(fmt.Sprintf("$App = '%s'\n", escapePS(app.AppDir)))
	ps.WriteString(fmt.Sprintf("$Exe = '%s'\n", escapePS(exeTarget)))
	ps.WriteString(fmt.Sprintf("$Icon = '%s'\n", escapePS(iconPath)))
	ps.WriteString("$wsh = New-Object -ComObject WScript.Shell\n")
	ps.WriteString("function Make-Link($path, $target, $argsStr, $desc) {\n")
	ps.WriteString("  $lnk = $wsh.CreateShortcut($path)\n")
	ps.WriteString("  $lnk.TargetPath = $target\n")
	ps.WriteString("  if ($argsStr) { $lnk.Arguments = $argsStr }\n")
	ps.WriteString("  $lnk.WorkingDirectory = $App\n")
	ps.WriteString("  $lnk.Description = $desc\n")
	ps.WriteString("  $lnk.IconLocation = $Icon\n")
	ps.WriteString("  $lnk.Save()\n")
	ps.WriteString("}\n")

	if app.OptStartMenu {
		ps.WriteString("$folder = Join-Path ([Environment]::GetFolderPath('Programs')) 'SC Cockpit'\n")
		ps.WriteString("New-Item -ItemType Directory -Force -Path $folder | Out-Null\n")
		ps.WriteString("Make-Link (Join-Path $folder 'SC Cockpit.lnk') $Exe '--launcher' 'SC Cockpit Control Center starten'\n")
		ps.WriteString("Make-Link (Join-Path $folder 'SC Cockpit (Administrator).lnk') (Join-Path $App 'Als-Administrator.bat') '' 'SC Cockpit mit Administratorrechten'\n")
		ps.WriteString("Make-Link (Join-Path $folder 'Deinstallieren.lnk') $Exe '--uninstall' 'SC Cockpit entfernen'\n")
	}

	if app.OptDesktop {
		ps.WriteString("Make-Link (Join-Path ([Environment]::GetFolderPath('Desktop')) 'SC Cockpit.lnk') $Exe '--launcher' 'SC Cockpit Control Center starten'\n")
	}

	ps.WriteString(fmt.Sprintf("$regPath = 'HKCU:\\%s'\n", uninstallRegKey))
	ps.WriteString("New-Item -Path $regPath -Force | Out-Null\n")
	ps.WriteString("Set-ItemProperty -Path $regPath -Name 'DisplayName' -Value 'SC Cockpit'\n")
	ps.WriteString(fmt.Sprintf("Set-ItemProperty -Path $regPath -Name 'DisplayVersion' -Value '%s'\n", AppVersion))
	ps.WriteString("Set-ItemProperty -Path $regPath -Name 'Publisher' -Value 'MOON-132 // SC Cockpit'\n")
	ps.WriteString("Set-ItemProperty -Path $regPath -Name 'InstallLocation' -Value $App\n")
	ps.WriteString("Set-ItemProperty -Path $regPath -Name 'DisplayIcon' -Value $Icon\n")
	ps.WriteString("Set-ItemProperty -Path $regPath -Name 'UninstallString' -Value ('\"' + $Exe + '\" --uninstall')\n")
	ps.WriteString("Set-ItemProperty -Path $regPath -Name 'NoModify' -Value 1 -Type DWord\n")
	ps.WriteString("Set-ItemProperty -Path $regPath -Name 'NoRepair' -Value 1 -Type DWord\n")

	_ = runHiddenPowerShell(ps.String())
}

func runInstallWorker() {
	defer func() {
		pPostMessageW.Call(app.Hwnd, WM_APP_DONE, 0, 0)
	}()

	app.InstallErr = ""
	app.setProgress(10, "Bereite Zielverzeichnisse vor...")
	if err := os.MkdirAll(app.UserDashDir, 0755); err != nil {
		app.InstallErr = fmt.Sprintf("Konnte Benutzerordner nicht anlegen: %v", err)
		return
	}
	if err := os.MkdirAll(app.AppDir, 0755); err != nil {
		app.InstallErr = fmt.Sprintf("Konnte Programmordner nicht anlegen: %v", err)
		return
	}

	zr, err := zip.NewReader(bytes.NewReader(embeddedPayload), int64(len(embeddedPayload)))
	if err != nil {
		app.InstallErr = fmt.Sprintf("Archivfehler im Installer: %v", err)
		return
	}

	totalFiles := len(zr.File)
	for i, f := range zr.File {
		relPath := filepath.FromSlash(f.Name)
		if strings.Contains(relPath, "..") {
			continue
		}
		targetPath := filepath.Join(app.AppDir, relPath)
		pct := 15 + ((i+1)*55)/maxInt(totalFiles, 1)
		app.setProgress(pct, fmt.Sprintf("Installiere %s...", f.Name))

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(targetPath, 0755)
			continue
		}
		_ = os.MkdirAll(filepath.Dir(targetPath), 0755)
		rc, err := f.Open()
		if err != nil {
			app.InstallErr = err.Error()
			return
		}
		outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			rc.Close()
			app.InstallErr = err.Error()
			return
		}
		_, _ = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		time.Sleep(35 * time.Millisecond)
	}

	app.setProgress(75, "Installiere grafisches SC Cockpit Control Center...")
	selfExe, err := os.Executable()
	if err == nil && selfExe != "" {
		if data, err := os.ReadFile(selfExe); err == nil {
			_ = os.WriteFile(filepath.Join(app.AppDir, "SC-Cockpit.exe"), data, 0755)
			_ = os.WriteFile(filepath.Join(app.AppDir, "SC-Cockpit-Setup.exe"), data, 0755)
		}
	}

	app.setProgress(85, "Erstelle Desktop- & Startmenü-Verknüpfungen...")
	createShortcutsAndRegistry()

	if !app.PythonFound && app.WingetFound && app.OptWingetPy {
		app.setProgress(92, "Installiere Python 3.12 über Windows Package Manager (winget)...")
		cmd := exec.Command("winget", "install", "-e", "--id", "Python.Python.3.12", "--scope", "user", "--accept-package-agreements", "--accept-source-agreements")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if cmd.Run() == nil {
			app.PythonFound = true
			app.PythonCmd = "py"
			app.PythonVer = "Python 3.12"
		}
	}

	app.setProgress(100, "Fertig! Starte SC Cockpit Control Center...")
	app.ExistingInst = true
	time.Sleep(250 * time.Millisecond)
}

func runUninstallWorker() {
	defer func() {
		pPostMessageW.Call(app.Hwnd, WM_APP_DONE, 0, 0)
	}()

	stopServer()
	app.InstallErr = ""
	app.setProgress(30, "Entferne Desktop- und Startmenü-Verknüpfungen...")

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

	target := app.AppDir
	if app.OptPurgeData {
		target = app.RootDir
		app.setProgress(70, "Entferne Programm und eigene Dashboards...")
	} else {
		app.setProgress(70, "Entferne Programmordner (eigene Dashboards bleiben erhalten)...")
	}

	_ = os.RemoveAll(target)
	if _, err := os.Stat(target); err == nil {
		cleanupCmd := fmt.Sprintf("timeout /t 2 /nobreak >nul & rmdir /s /q \"%s\"", target)
		cmd := exec.Command("cmd.exe", "/c", cleanupCmd)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = cmd.Start()
	}

	time.Sleep(250 * time.Millisecond)
	app.setProgress(100, "Deinstallation abgeschlossen.")
	app.ExistingInst = false
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
