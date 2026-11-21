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

//go:embed ships_catalog.json
var embeddedShipsCatalog []byte

var defaultHangarFavorites = []string{
	"aegis-gladius",
	"misc-prospector",
	"drake-vulture",
	"crusader-c2-hercules",
	"drake-cutlass-black",
	"anvil-carrack",
	"rsi-constellation-andromeda",
	"aegis-avenger-titan",
}

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
		hasLocalServer := false
		if exeDir != "" {
			if _, err := os.Stat(filepath.Join(exeDir, "server.py")); err == nil {
				hasLocalServer = true
			}
		}

		// Open directly in Live Cockpit (ModeLauncher) whenever:
		// 1. Running from the git repo / folder where server.py is right next to the .exe
		// 2. SC Cockpit is already installed in %LOCALAPPDATA%\SC Cockpit\app (auto-syncs latest embedded files!)
		// 3. --launcher was passed explicitly
		if forceLauncher || hasLocalServer || app.ExistingInst {
			app.Mode = ModeLauncher
			enablePortableOrInstalledMode()
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

	app.LanIPs = detectLANIPs()
	loadConfigPort()
}

func extractEmbeddedPayloadSilent(targetDir string) {
	_ = os.MkdirAll(targetDir, 0755)
	zr, err := zip.NewReader(bytes.NewReader(embeddedPayload), int64(len(embeddedPayload)))
	if err != nil {
		return
	}
	for _, f := range zr.File {
		relPath := filepath.FromSlash(f.Name)
		if strings.Contains(relPath, "..") {
			continue
		}
		targetPath := filepath.Join(targetDir, relPath)
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(targetPath, 0755)
			continue
		}
		_ = os.MkdirAll(filepath.Dir(targetPath), 0755)
		rc, err := f.Open()
		if err != nil {
			continue
		}
		outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			rc.Close()
			continue
		}
		_, _ = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
	}
}

func enablePortableOrInstalledMode() {
	selfExe, err := os.Executable()
	if err == nil && selfExe != "" {
		exeDir := filepath.Dir(selfExe)
		// Case 1: Running directly inside the git repo / portable folder where server.py sits next to the exe
		if !strings.EqualFold(filepath.Clean(exeDir), filepath.Clean(app.AppDir)) {
			if _, err := os.Stat(filepath.Join(exeDir, "server.py")); err == nil {
				app.AppDir = exeDir
				app.UserDir = filepath.Join(exeDir, "dashboards", "local")
				app.UserDashDir = filepath.Join(exeDir, "dashboards", "local")
				_ = os.MkdirAll(app.UserDashDir, 0755)
				return
			}
		}
	}
	// Case 2: Running standalone .exe (e.g. from Downloads) or installed app -> silently unpack latest embedded files into AppDir so it works immediately!
	_ = os.MkdirAll(app.UserDashDir, 0755)
	extractEmbeddedPayloadSilent(app.AppDir)
}

// LanAddr is one IPv4 address a phone might open. Virtual is VPN, Hyper-V, WSL, …
// 169.254.0.0/16 (APIPA) is never included.
type LanAddr struct {
	IP      string
	Name    string
	Virtual bool
}

func plausibleIPv4(ip net.IP) bool {
	ip = ip.To4()
	if ip == nil || ip.IsLoopback() || ip.IsMulticast() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() {
		return false
	}
	// Explicit APIPA filter. Idle adapters often hold 169.254 and are not reachable.
	if ip[0] == 169 && ip[1] == 254 {
		return false
	}
	return true
}

func virtualAdapter(name string) bool {
	low := strings.ToLower(name)
	for _, mark := range []string{
		"vethernet", "hyper-v", "wsl", "virtualbox", "vmware", "vbox",
		"docker", "tailscale", "nordlynx", "wireguard", "zerotier",
		"hamachi", "bluetooth", "loopback", "isatap", "teredo",
		"vpn", "openvpn", "npcap", "veth", "virbr", "tunnel", "virtual",
	} {
		if strings.Contains(low, mark) {
			return true
		}
	}
	return false
}

func ipv4Band(ip net.IP) int {
	ip = ip.To4()
	if ip == nil {
		return 3
	}
	switch {
	case ip[0] == 192 && ip[1] == 168:
		return 0
	case ip[0] == 10:
		return 1
	case ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31:
		return 2
	default:
		return 3
	}
}

func lanRank(a LanAddr) int {
	rank := ipv4Band(net.ParseIP(a.IP))
	if a.Virtual {
		rank += 10
	}
	return rank
}

func detectLANIPs() []LanAddr {
	ifaces, err := net.Interfaces()
	if err != nil {
		return []LanAddr{{IP: "127.0.0.1", Name: "localhost"}}
	}
	seen := map[string]bool{}
	var out []LanAddr
	for _, ifi := range ifaces {
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := ifi.Addrs()
		if err != nil {
			continue
		}
		virt := virtualAdapter(ifi.Name)
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok || !plausibleIPv4(ipnet.IP) {
				continue
			}
			ip := ipnet.IP.To4().String()
			if seen[ip] {
				continue
			}
			seen[ip] = true
			out = append(out, LanAddr{IP: ip, Name: ifi.Name, Virtual: virt})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		ri, rj := lanRank(out[i]), lanRank(out[j])
		if ri != rj {
			return ri < rj
		}
		return out[i].IP < out[j].IP
	})
	if len(out) == 0 {
		return []LanAddr{{IP: "127.0.0.1", Name: "localhost"}}
	}
	return out
}

func refreshLANIPs() {
	found := detectLANIPs()
	app.mu.Lock()
	app.LanIPs = found
	app.mu.Unlock()
	if app.Hwnd != 0 {
		pPostMessageW.Call(app.Hwnd, WM_APP_REFRESH, 0, 0)
	}
}

func snapshotLAN() (running bool, starting bool, ips []LanAddr) {
	app.mu.Lock()
	defer app.mu.Unlock()
	running = app.ServerRunning
	starting = app.ServerStarting
	ips = append([]LanAddr(nil), app.LanIPs...)
	return
}

func isLoopbackIP(ip string) bool {
	return ip == "127.0.0.1" || strings.HasPrefix(ip, "127.")
}

func primaryLan(ips []LanAddr) LanAddr {
	for _, a := range ips {
		if !a.Virtual && a.IP != "" && !isLoopbackIP(a.IP) {
			return a
		}
	}
	for _, a := range ips {
		if a.IP != "" && !isLoopbackIP(a.IP) {
			return a
		}
	}
	if len(ips) > 0 && ips[0].IP != "" {
		return ips[0]
	}
	return LanAddr{IP: "127.0.0.1", Name: "localhost"}
}

func formatLANLine(a LanAddr, port int) string {
	line := fmt.Sprintf("http://%s:%d/", a.IP, port)
	if a.Name != "" && !isLoopbackIP(a.IP) {
		line += "   " + a.Name
	}
	if a.Virtual {
		line += "   virtuell"
	}
	return line
}

func serverBusy() bool {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.ServerRunning || app.ServerStarting
}

func serverIsUp() bool {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.ServerRunning
}

func serverStillOurs(cmd *exec.Cmd) bool {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.ServerCmd != nil && cmd != nil && cmd.Process != nil && app.ServerCmd.Pid == cmd.Process.Pid
}

func portOpen(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 200*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
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

func markServerListening(port int) {
	found := detectLANIPs()
	app.mu.Lock()
	if app.ServerCmd == nil || app.ServerRunning {
		app.mu.Unlock()
		return
	}
	app.LanIPs = found
	app.ServerRunning = true
	app.ServerStarting = false
	ips := append([]LanAddr(nil), app.LanIPs...)
	app.mu.Unlock()
	appendServerLog(fmt.Sprintf("Server läuft auf Port %d.", port))
	usable := false
	for _, a := range ips {
		if !isLoopbackIP(a.IP) {
			usable = true
		}
		appendServerLog(formatLANLine(a, port))
	}
	if !usable {
		appendServerLog("Keine LAN-Adresse gefunden. WLAN oder Kabel prüfen.")
	}
}

// startServer only reports "Server läuft" after server.py binds the port.
// cmd.Start success is not enough: Python can exit at once (port in use).
func startServer() {
	if serverBusy() {
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
	cmd.Env = append(os.Environ(), "SC_COCKPIT_HOME="+app.UserDir, "PYTHONIOENCODING=utf-8", "PYTHONUNBUFFERED=1")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		appendServerLog("Fehler beim Starten: " + err.Error())
		return
	}
	cmd.Stderr = cmd.Stdout

	// If something else already holds the port, a later probe would look like success.
	busyBefore := portOpen(app.Port)
	if err := cmd.Start(); err != nil {
		appendServerLog("Python-Server konnte nicht gestartet werden: " + err.Error())
		return
	}

	app.mu.Lock()
	app.ServerCmd = cmd.Process
	app.ServerStarting = true
	app.ServerRunning = false
	port := app.Port
	app.mu.Unlock()
	refreshLANIPs()
	appendServerLog("Python startet, warte auf den offenen Port...")

	go func(r io.Reader, c *exec.Cmd, port int) {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			text := strings.TrimSpace(scanner.Text())
			if text == "" {
				continue
			}
			if strings.Contains(text, "Server lauscht auf ") {
				markServerListening(port)
				continue
			}
			if strings.HasPrefix(text, "Dieses Fenster") || strings.HasPrefix(text, "Strg+C") || strings.Contains(text, "Gerät:") {
				continue
			}
			appendServerLog(text)
		}
		_ = c.Wait()
		app.mu.Lock()
		same := app.ServerCmd != nil && c.Process != nil && app.ServerCmd.Pid == c.Process.Pid
		wasUp := app.ServerRunning
		if same {
			app.ServerRunning = false
			app.ServerStarting = false
			app.ServerCmd = nil
		}
		app.mu.Unlock()
		if !same {
			return
		}
		if wasUp {
			appendServerLog("Server gestoppt.")
		} else {
			appendServerLog("Server lauscht nicht. Port belegt oder Python abgestürzt — Meldungen oben prüfen.")
		}
	}(stdout, cmd, port)

	if busyBefore {
		return
	}
	go func(c *exec.Cmd, port int) {
		deadline := time.Now().Add(8 * time.Second)
		for time.Now().Before(deadline) {
			if serverIsUp() || !serverStillOurs(c) {
				return
			}
			if portOpen(port) && serverStillOurs(c) {
				markServerListening(port)
				return
			}
			time.Sleep(200 * time.Millisecond)
		}
		if !serverIsUp() && serverStillOurs(c) {
			appendServerLog(fmt.Sprintf("Python antwortet nicht auf Port %d.", port))
		}
	}(cmd, port)
}

func stopServer() {
	app.mu.Lock()
	proc := app.ServerCmd
	app.ServerCmd = nil
	app.ServerRunning = false
	app.ServerStarting = false
	app.mu.Unlock()

	if proc != nil {
		_ = proc.Kill()
		appendServerLog("Server gestoppt.")
	}
}

// ============================================================================
// DASHBOARD CATALOG, 100+ STAR CITIZEN SHIPS & HANGAR FAVORITES
// ============================================================================
type stateJSON struct {
	Active    string   `json:"active"`
	Favorites []string `json:"favorites"`
}

func loadStateJSON() stateJSON {
	st := stateJSON{
		Active:    "aegis-gladius",
		Favorites: append([]string(nil), defaultHangarFavorites...),
	}
	stateFile := filepath.Join(app.UserDir, "state.json")
	if data, err := os.ReadFile(stateFile); err == nil {
		var raw map[string]interface{}
		if json.Unmarshal(data, &raw) == nil {
			if act, ok := raw["active"].(string); ok && act != "" {
				st.Active = act
			}
			if favArr, ok := raw["favorites"].([]interface{}); ok {
				var parsedFavs []string
				for _, f := range favArr {
					if fs, ok2 := f.(string); ok2 && fs != "" {
						parsedFavs = append(parsedFavs, fs)
					}
				}
				st.Favorites = parsedFavs
			}
		}
	}
	return st
}

func saveStateJSON(st stateJSON) {
	_ = os.MkdirAll(app.UserDir, 0755)
	stateFile := filepath.Join(app.UserDir, "state.json")
	if out, err := json.MarshalIndent(st, "", "  "); err == nil {
		_ = os.WriteFile(stateFile, append(out, '\n'), 0644)
	}
}

func toggleDashboardFavorite(id string) bool {
	st := loadStateJSON()
	isFav := false
	var next []string
	found := false
	for _, f := range st.Favorites {
		if f == id {
			found = true
		} else {
			next = append(next, f)
		}
	}
	if !found {
		next = append(next, id)
		isFav = true
	}
	st.Favorites = next
	saveStateJSON(st)
	for i := range app.Dashboards {
		if app.Dashboards[i].ID == id {
			app.Dashboards[i].Favorite = isFav || app.Dashboards[i].Source == "local"
		}
	}
	return isFav
}

func reloadDashboards() {
	index := make(map[string]int)
	var list []DashboardInfo

	registerParsed := func(id, title, ship, mfg, role, hullType string, subs []string, pages []DashPage, source, fullPath string) {
		if len(pages) == 0 || id == "" || id == "state" || id == "local" {
			return
		}
		if title == "" {
			title = id
		}
		if ship == "" {
			ship = title
		}
		if mfg == "" {
			mfg = "UEE"
		}
		if role == "" {
			role = "Multi-Role"
		}
		if hullType == "" {
			low := strings.ToLower(role + " " + id)
			if strings.Contains(low, "mining") {
				hullType = "mining_bubble"
			} else if strings.Contains(low, "salvage") {
				hullType = "twin_industrial"
			} else if strings.Contains(low, "hauler") || strings.Contains(low, "fracht") || strings.Contains(low, "cargo") {
				hullType = "crusader_wing"
			} else {
				hullType = "fighter_delta"
			}
		}
		if len(subs) == 0 {
			for _, p := range pages {
				if p.ID != "" {
					subs = append(subs, p.ID)
				}
			}
		}
		btnCount := 0
		for _, p := range pages {
			for _, g := range p.Groups {
				btnCount += len(g.Buttons)
			}
		}
		info := DashboardInfo{
			ID:           id,
			Title:        title,
			Ship:         ship,
			Manufacturer: mfg,
			Role:         role,
			HullType:     hullType,
			Subsystems:   subs,
			Source:       source,
			Path:         fullPath,
			PageCount:    len(pages),
			ButtonCount:  btnCount,
			Pages:        pages,
		}
		if existingIdx, ok := index[id]; ok {
			list[existingIdx] = info
		} else {
			index[id] = len(list)
			list = append(list, info)
		}
	}

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
			fullPath := filepath.Join(dir, e.Name())
			data, err := os.ReadFile(fullPath)
			if err != nil {
				continue
			}
			var parsed struct {
				ID           string     `json:"id"`
				Title        string     `json:"title"`
				Ship         string     `json:"ship"`
				Manufacturer string     `json:"manufacturer"`
				Role         string     `json:"role"`
				HullType     string     `json:"hull_type"`
				Subsystems   []string   `json:"subsystems"`
				Pages        []DashPage `json:"pages"`
			}
			if json.Unmarshal(data, &parsed) != nil || len(parsed.Pages) == 0 {
				continue
			}
			if parsed.ID != "" {
				id = parsed.ID
			}
			registerParsed(id, parsed.Title, parsed.Ship, parsed.Manufacturer, parsed.Role, parsed.HullType, parsed.Subsystems, parsed.Pages, source, fullPath)
		}
	}

	// Load all 100+ personalized Star Citizen ship dashboards from embeddedShipsCatalog!
	var embeddedShips []struct {
		ID           string     `json:"id"`
		Title        string     `json:"title"`
		Ship         string     `json:"ship"`
		Manufacturer string     `json:"manufacturer"`
		Role         string     `json:"role"`
		HullType     string     `json:"hull_type"`
		Subsystems   []string   `json:"subsystems"`
		Pages        []DashPage `json:"pages"`
	}
	if len(embeddedShipsCatalog) > 0 && json.Unmarshal(embeddedShipsCatalog, &embeddedShips) == nil {
		for _, s := range embeddedShips {
			registerParsed(s.ID, s.Title, s.Ship, s.Manufacturer, s.Role, s.HullType, s.Subsystems, s.Pages, "builtin", "")
		}
	}

	readDir(app.UserDashDir, "local")

	st := loadStateJSON()
	favSet := make(map[string]bool)
	for _, f := range st.Favorites {
		favSet[f] = true
	}
	for i := range list {
		list[i].Favorite = favSet[list[i].ID] || list[i].Source == "local"
	}

	app.Dashboards = list

	active := st.Active
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
	st := loadStateJSON()
	st.Active = id
	saveStateJSON(st)
	app.ActiveDash = id
}

func cycleActiveDashboard(dir int) {
	if len(app.Dashboards) == 0 {
		return
	}
	// Cycle primarily through the user's favorited Hangar ships!
	var favList []DashboardInfo
	for _, d := range app.Dashboards {
		if d.Favorite || d.ID == app.ActiveDash {
			favList = append(favList, d)
		}
	}
	if len(favList) == 0 {
		favList = app.Dashboards
	}
	curIdx := 0
	for i, d := range favList {
		if d.ID == app.ActiveDash {
			curIdx = i
			break
		}
	}
	nextIdx := (curIdx + dir + len(favList)) % len(favList)
	setActiveDashboard(favList[nextIdx].ID)
	app.LivePageIdx = 0
}

func triggerInAppButton(btn DashButton) {
	if len(btn.Keys) == 0 {
		return
	}
	combo := strings.ToUpper(strings.Join(btn.Keys, "+"))
	app.LastActionTxt = fmt.Sprintf("%s (%s)", btn.Label, combo)
	app.ToastMsg = fmt.Sprintf("SHARD AUSGELÖST:  %s  ►  [%s]", strings.ToUpper(btn.Label), combo)

	if !serverBusy() {
		startServer()
	}

	go func(b DashButton) {
		for i := 0; i < 40 && !serverIsUp(); i++ {
			time.Sleep(100 * time.Millisecond)
		}
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

func marshalDashboardToMap(d DashboardInfo) map[string]interface{} {
	if d.Path != "" {
		if raw, err := os.ReadFile(d.Path); err == nil {
			var m map[string]interface{}
			if json.Unmarshal(raw, &m) == nil {
				return m
			}
		}
	}
	pagesBytes, _ := json.Marshal(d.Pages)
	var pagesGeneric []interface{}
	_ = json.Unmarshal(pagesBytes, &pagesGeneric)
	return map[string]interface{}{
		"id":           d.ID,
		"title":        d.Title,
		"ship":         d.Ship,
		"manufacturer": d.Manufacturer,
		"role":         d.Role,
		"hull_type":    d.HullType,
		"subsystems":   d.Subsystems,
		"pages":        pagesGeneric,
	}
}

func duplicateDashboard(src DashboardInfo) (string, error) {
	_ = os.MkdirAll(app.UserDashDir, 0755)
	data := marshalDashboardToMap(src)

	newID := src.ID + "-custom"
	n := 2
	for {
		if _, err := os.Stat(filepath.Join(app.UserDashDir, newID+".json")); os.IsNotExist(err) {
			break
		}
		newID = fmt.Sprintf("%s-custom-%d", src.ID, n)
		n++
	}

	data["id"] = newID
	data["title"] = src.Title + " (Custom)"
	data["ship"] = src.Ship + " (Custom)"
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

	shipTitle := fmt.Sprintf("Eigenes Schiff %d", n-1)
	template := map[string]interface{}{
		"id":           newID,
		"title":        shipTitle,
		"ship":         shipTitle,
		"manufacturer": "RSI",
		"role":         "Multi-Role // Custom",
		"hull_type":    "fighter_delta",
		"bg_opacity":   0.48,
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

func toggleAlwaysOnTop() {
	app.AlwaysOnTop = !app.AlwaysOnTop
	pSetWindowPos := user32.NewProc("SetWindowPos")
	var zOrder uintptr
	if app.AlwaysOnTop {
		zOrder = ^uintptr(0) // -1 (HWND_TOPMOST)
		app.ToastMsg = "OVERLAY-MODUS AKTIV: Fenster bleibt über Star Citizen (Borderless) im Vordergrund."
	} else {
		zOrder = ^uintptr(1) // -2 (HWND_NOTOPMOST)
		app.ToastMsg = "Overlay-Modus deaktiviert."
	}
	pSetWindowPos.Call(app.Hwnd, zOrder, 0, 0, 0, 0, 0x0003)
}

func addSubsystemToActiveShip(subKind string) {
	var cur *DashboardInfo
	for i := range app.Dashboards {
		if app.Dashboards[i].ID == app.ActiveDash {
			cur = &app.Dashboards[i]
			break
		}
	}
	if cur == nil {
		app.ToastMsg = "Bitte zuerst ein Schiff wählen."
		return
	}

	data := marshalDashboardToMap(*cur)

	var newPage map[string]interface{}
	switch subKind {
	case "mining":
		newPage = map[string]interface{}{
			"id":    fmt.Sprintf("mining-%d", cur.PageCount+1),
			"label": "Mining & Laser",
			"groups": []map[string]interface{}{
				{
					"title": "Mining-Laser & Verbrauchsgüter",
					"buttons": []map[string]interface{}{
						{"id": "mine-mode", "sc": "v_toggle_mining_mode", "label": "Mining-Modus", "hint": "M", "keys": []string{"m"}, "mode": "toggle", "color": "amber", "wide": true},
						{"id": "mine-laser", "label": "Laser / Extraktion", "hint": "Maus Links halten", "keys": []string{"mouse1"}, "mode": "hold", "color": "red", "danger": true, "wide": true},
						{"id": "mine-up", "label": "Laser-Leistung +", "hint": "Mausrad Hoch", "keys": []string{"wheelup"}, "mode": "tap", "color": "amber"},
						{"id": "mine-dn", "label": "Laser-Leistung —", "hint": "Mausrad Runter", "keys": []string{"wheeldown"}, "mode": "tap", "color": "cyan"},
						{"id": "mine-m1", "sc": "v_mining_use_consumable1", "label": "Modul 1", "hint": "L-Alt + 1", "keys": []string{"lalt", "1"}, "mode": "tap", "color": "cyan"},
						{"id": "mine-m2", "sc": "v_mining_use_consumable2", "label": "Modul 2", "hint": "L-Alt + 2", "keys": []string{"lalt", "2"}, "mode": "tap", "color": "cyan"},
						{"id": "mine-m3", "sc": "v_mining_use_consumable3", "label": "Modul 3", "hint": "L-Alt + 3", "keys": []string{"lalt", "3"}, "mode": "tap", "color": "cyan"},
						{"id": "mine-jet", "sc": "v_jettison_volatile_cargo", "label": "Erz abwerfen", "hint": "L-Alt + J", "keys": []string{"lalt", "j"}, "mode": "tap", "color": "red", "danger": true},
					},
				},
			},
		}
	case "salvage":
		newPage = map[string]interface{}{
			"id":    fmt.Sprintf("salvage-%d", cur.PageCount+1),
			"label": "Salvage & Scraper",
			"groups": []map[string]interface{}{
				{
					"title": "Hüllen-Bergung & Struktur-Fraktur",
					"buttons": []map[string]interface{}{
						{"id": "slv-mode", "sc": "v_toggle_salvage_mode", "label": "Salvage-Modus", "hint": "M", "keys": []string{"m"}, "mode": "toggle", "color": "amber", "wide": true},
						{"id": "slv-beam", "label": "Scraper-Strahl", "hint": "Maus Links halten", "keys": []string{"mouse1"}, "mode": "hold", "color": "red", "danger": true, "wide": true},
						{"id": "slv-all", "sc": "v_salvage_focus_all_heads", "label": "Beide Köpfe", "hint": "R-Alt + S", "keys": []string{"ralt", "s"}, "mode": "tap", "color": "amber"},
						{"id": "slv-gimb", "sc": "v_salvage_toggle_gimbal_mode", "label": "Salvage Gimbal", "hint": "G", "keys": []string{"g"}, "mode": "toggle", "color": "cyan"},
						{"id": "slv-frac", "sc": "v_salvage_toggle_fire_fracture", "label": "Fraktur-Feld", "hint": "R-Alt + W", "keys": []string{"ralt", "w"}, "mode": "tap", "color": "red", "wide": true},
						{"id": "slv-cm", "sc": "v_salvage_toggle_fire_disintegrate", "label": "Desintegration (CM)", "hint": "R-Alt + X", "keys": []string{"ralt", "x"}, "mode": "tap", "color": "amber", "wide": true},
					},
				},
			},
		}
	case "kampf":
		newPage = map[string]interface{}{
			"id":    fmt.Sprintf("kampf-%d", cur.PageCount+1),
			"label": "Kampf & Waffen",
			"groups": []map[string]interface{}{
				{
					"title": "Gefechtssysteme & Gegenmaßnahmen",
					"buttons": []map[string]interface{}{
						{"id": "kmp-tgt", "sc": "v_target_nearest_hostile", "label": "Nächster Feind", "hint": "T", "keys": []string{"t"}, "mode": "tap", "color": "red"},
						{"id": "kmp-gmb", "sc": "v_weapon_gimbals_state_toggle", "label": "Gimbal Modus", "hint": "G", "keys": []string{"g"}, "mode": "tap", "color": "blue"},
						{"id": "kmp-flr", "sc": "v_weapon_countermeasure_decoy_launch", "label": "Flare / Köder", "hint": "H", "keys": []string{"h"}, "mode": "tap", "color": "amber"},
						{"id": "kmp-nse", "sc": "v_weapon_countermeasure_noise_launch", "label": "Noise / Störfeld", "hint": "J", "keys": []string{"j"}, "mode": "tap", "color": "cyan"},
						{"id": "kmp-f1", "sc": "v_attack1", "label": "Waffengruppe 1", "hint": "Maus Links", "keys": []string{"mouse1"}, "mode": "hold", "color": "red", "danger": true, "wide": true},
						{"id": "kmp-f2", "sc": "v_attack2", "label": "Waffengruppe 2", "hint": "Maus Rechts", "keys": []string{"mouse2"}, "mode": "hold", "color": "red", "danger": true, "wide": true},
					},
				},
			},
		}
	case "cargo":
		newPage = map[string]interface{}{
			"id":    fmt.Sprintf("cargo-%d", cur.PageCount+1),
			"label": "Fracht & Rampen",
			"groups": []map[string]interface{}{
				{
					"title": "Hangar, Rampen & Frachtgitter",
					"buttons": []map[string]interface{}{
						{"id": "crg-drs", "sc": "v_toggle_all_doors", "label": "Alle Rampen / Türen", "hint": "R-Alt + K", "keys": []string{"ralt", "k"}, "mode": "toggle", "color": "amber", "wide": true},
						{"id": "crg-lck", "sc": "v_lock_all_doors", "label": "Verriegeln", "hint": "R-Alt + L", "keys": []string{"ralt", "l"}, "mode": "toggle", "color": "cyan"},
						{"id": "crg-prt", "sc": "v_toggle_all_portlocks", "label": "Frachtgitter / Portlocks", "hint": "R-Alt + P", "keys": []string{"ralt", "p"}, "mode": "toggle", "color": "amber"},
						{"id": "crg-atc", "sc": "v_atc_loading_area_request", "label": "Ladebereich anfragen", "hint": "L-Alt + K", "keys": []string{"lalt", "k"}, "mode": "tap", "color": "green", "wide": true},
					},
				},
			},
		}
	default: // "energie"
		newPage = map[string]interface{}{
			"id":    fmt.Sprintf("energie-%d", cur.PageCount+1),
			"label": "Energie & Schilde",
			"groups": []map[string]interface{}{
				{
					"title": "Energie-Dreieck & Schilde",
					"buttons": []map[string]interface{}{
						{"id": "pwr-wep", "label": "Energie -> Waffen", "hint": "F5", "keys": []string{"f5"}, "mode": "tap", "color": "red"},
						{"id": "pwr-eng", "label": "Energie -> Antrieb", "hint": "F6", "keys": []string{"f6"}, "mode": "tap", "color": "amber"},
						{"id": "pwr-shd", "label": "Energie -> Schilde", "hint": "F7", "keys": []string{"f7"}, "mode": "tap", "color": "cyan"},
						{"id": "pwr-rst", "label": "Energie Reset", "hint": "F8", "keys": []string{"f8"}, "mode": "tap", "color": "slate"},
					},
				},
			},
		}
	}

	pagesArr, _ := data["pages"].([]interface{})
	pagesArr = append(pagesArr, newPage)
	data["pages"] = pagesArr

	_ = os.MkdirAll(app.UserDashDir, 0755)
	targetFile := filepath.Join(app.UserDashDir, cur.ID+".json")
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		app.ToastMsg = "Speicherfehler: " + err.Error()
		return
	}
	_ = os.WriteFile(targetFile, append(out, '\n'), 0644)
	reloadDashboards()
	app.ToastMsg = fmt.Sprintf("Unterpunkt '%s' wurde zum Schiff '%s' hinzugefügt!", newPage["label"], cur.Ship)
}

func runGitHubLivePull(userInitiated bool) {
	updatedCount := 0

	gitDir := filepath.Join(app.AppDir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		if gitExe, err2 := exec.LookPath("git"); err2 == nil && gitExe != "" {
			cmd := exec.Command(gitExe, "pull", "--ff-only")
			cmd.Dir = app.AppDir
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			if out, err3 := cmd.CombinedOutput(); err3 == nil {
				outStr := strings.TrimSpace(string(out))
				if !strings.Contains(outStr, "Already up to date") && !strings.Contains(outStr, "Bereits aktuell") {
					updatedCount++
				}
			}
		}
	}

	rawBase := "https://raw.githubusercontent.com/MOON-132/Star-Citizen-Cockpit/main/"
	filesToSync := []string{
		"server.py",
		"sc_ships.py",
		"index.html",
		"dashboards/standard.json",
		"dashboards/mining.json",
		"dashboards/salvage.json",
		"dashboards/hauler.json",
	}
	client := &http.Client{Timeout: 5 * time.Second}
	for _, rel := range filesToSync {
		resp, err := client.Get(rawBase + rel)
		if err != nil || resp == nil {
			continue
		}
		if resp.StatusCode == 200 {
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err == nil && len(body) > 32 {
				targetPath := filepath.Join(app.AppDir, filepath.FromSlash(rel))
				_ = os.MkdirAll(filepath.Dir(targetPath), 0755)
				existing, _ := os.ReadFile(targetPath)
				if !bytes.Equal(existing, body) {
					if os.WriteFile(targetPath, body, 0644) == nil {
						updatedCount++
					}
				}
			}
		} else {
			resp.Body.Close()
		}
	}

	reloadDashboards()
	if updatedCount > 0 {
		if serverBusy() {
			stopServer()
			startServer()
		}
		app.ToastMsg = "AUTO-UPDATE ERFOLGREICH: Neueste Version & Schiffe wurden von GitHub geladen!"
	} else if userInitiated {
		app.ToastMsg = "SC Cockpit ist bereits auf dem allerneuesten GitHub-Stand."
	}
	if app.Hwnd != 0 {
		pPostMessageW.Call(app.Hwnd, WM_APP_REFRESH, 0, 0)
	}
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
