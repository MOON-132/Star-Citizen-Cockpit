package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

const (
	AppVersion = "0.2.0"
	AppName    = "SC Cockpit"
)

func rgb(r, g, b uint8) uint32 {
	return uint32(r) | (uint32(g) << 8) | (uint32(b) << 16)
}

// GameGlass Holographic Sci-Fi Palette
var (
	colBg       = rgb(0x05, 0x08, 0x0d) // #05080d Deep Space Obsidian
	colSidebar  = rgb(0x08, 0x0d, 0x15) // #080d15 GameGlass Rail
	colPanel    = rgb(0x0d, 0x15, 0x22) // #0d1522 Shard Panel
	colPanelHov = rgb(0x15, 0x22, 0x34) // #152234 Shard Hover
	colPanelAct = rgb(0x18, 0x29, 0x3e) // #18293e Shard Active
	colLine     = rgb(0x20, 0x31, 0x46) // #203146 Frame Border
	colLineHi   = rgb(0x30, 0x4a, 0x69) // #304a69 Highlight Border
	colText     = rgb(0xee, 0xf5, 0xfc) // #eef5fc Crisp White
	colMuted    = rgb(0x7c, 0x93, 0xad) // #7c93ad Muted Label
	colDim      = rgb(0x4a, 0x5d, 0x73) // #4a5d73 Dim Subtext
	colCyan     = rgb(0x00, 0xe5, 0xff) // #00e5ff GameGlass Electric Cyan
	colCyanBg   = rgb(0x08, 0x26, 0x32) // #082632 Cyan Glass Fill
	colAmber    = rgb(0xff, 0xab, 0x00) // #ffab00 Quantum Amber
	colAmberHov = rgb(0xff, 0xc0, 0x33) // #ffc033 Amber Hover
	colAmberBg  = rgb(0x28, 0x1d, 0x08) // #281d08 Amber Glass Fill
	colGreen    = rgb(0x00, 0xe6, 0x76) // #00e676 Online Green
	colGreenBg  = rgb(0x09, 0x26, 0x18) // #092618 Green Glass Fill
	colRed      = rgb(0xff, 0x3d, 0x57) // #ff3d57 Combat Crimson
	colRedBg    = rgb(0x2b, 0x0e, 0x14) // #2b0e14 Red Glass Fill
	colBlue     = rgb(0x3d, 0x8b, 0xff) // #3d8bff Navigation Blue
	colBlueBg   = rgb(0x0c, 0x1e, 0x38) // #0c1e38 Blue Glass Fill
	colSlate    = rgb(0x8d, 0xa2, 0xb8) // #8da2b8 Slate
)

// Win32 Constants
const (
	WS_POPUP        = 0x80000000
	WS_VISIBLE      = 0x10000000
	WS_SYSMENU      = 0x00080000
	WS_MINIMIZEBOX  = 0x00020000
	WS_CLIPCHILDREN = 0x02000000
	WS_EX_APPWINDOW = 0x00040000

	WM_DESTROY     = 0x0002
	WM_PAINT       = 0x000F
	WM_CLOSE       = 0x0010
	WM_ERASEBKGND  = 0x0014
	WM_SETCURSOR   = 0x0020
	WM_NCHITTEST   = 0x0084
	WM_KEYDOWN     = 0x0100
	WM_MOUSEMOVE   = 0x0200
	WM_LBUTTONDOWN = 0x0201
	WM_LBUTTONUP   = 0x0202
	WM_APP_REFRESH = 0x8001
	WM_APP_DONE    = 0x8002

	HTCLIENT  = 1
	HTCAPTION = 2

	IDC_ARROW = 32512
	IDC_HAND  = 32649

	DT_LEFT       = 0x00000000
	DT_CENTER     = 0x00000001
	DT_RIGHT      = 0x00000002
	DT_VCENTER    = 0x00000004
	DT_WORDBREAK  = 0x00000010
	DT_SINGLELINE = 0x00000020
	DT_NOPREFIX   = 0x00000800

	TRANSPARENT = 1
	PS_SOLID    = 0

	SW_SHOWNORMAL = 1
	SW_MINIMIZE   = 6

	CS_HREDRAW    = 0x0002
	CS_VREDRAW    = 0x0001
	CS_DROPSHADOW = 0x00020000

	VK_ESCAPE = 0x1B

	SRCCOPY           = 0x00CC0020
	CLEARTYPE_QUALITY = 5
	FW_NORMAL         = 400
	FW_SEMIBOLD       = 600
	FW_BOLD           = 700
)

type POINT struct {
	X, Y int32
}

type RECT struct {
	Left, Top, Right, Bottom int32
}

type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type WNDCLASSEXW struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type PAINTSTRUCT struct {
	Hdc         uintptr
	Erase       int32
	RcPaint     RECT
	Restore     int32
	IncUpdate   int32
	RgbReserved [32]byte
}

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")

	pSetProcessDPIAware = user32.NewProc("SetProcessDPIAware")
	pRegisterClassExW   = user32.NewProc("RegisterClassExW")
	pCreateWindowExW    = user32.NewProc("CreateWindowExW")
	pDefWindowProcW     = user32.NewProc("DefWindowProcW")
	pShowWindow         = user32.NewProc("ShowWindow")
	pUpdateWindow       = user32.NewProc("UpdateWindow")
	pGetMessageW        = user32.NewProc("GetMessageW")
	pTranslateMessage   = user32.NewProc("TranslateMessage")
	pDispatchMessageW   = user32.NewProc("DispatchMessageW")
	pPostQuitMessage    = user32.NewProc("PostQuitMessage")
	pPostMessageW       = user32.NewProc("PostMessageW")
	pBeginPaint         = user32.NewProc("BeginPaint")
	pEndPaint           = user32.NewProc("EndPaint")
	pInvalidateRect     = user32.NewProc("InvalidateRect")
	pGetClientRect      = user32.NewProc("GetClientRect")
	pFillRect           = user32.NewProc("FillRect")
	pDrawTextW          = user32.NewProc("DrawTextW")
	pLoadCursorW        = user32.NewProc("LoadCursorW")
	pLoadIconW          = user32.NewProc("LoadIconW")
	pSetCursor          = user32.NewProc("SetCursor")
	pGetSystemMetrics   = user32.NewProc("GetSystemMetrics")
	pSetCapture         = user32.NewProc("SetCapture")
	pReleaseCapture     = user32.NewProc("ReleaseCapture")
	pScreenToClient     = user32.NewProc("ScreenToClient")
	pOpenClipboard      = user32.NewProc("OpenClipboard")
	pEmptyClipboard     = user32.NewProc("EmptyClipboard")
	pSetClipboardData   = user32.NewProc("SetClipboardData")
	pCloseClipboard     = user32.NewProc("CloseClipboard")

	pCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	pCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	pSelectObject           = gdi32.NewProc("SelectObject")
	pDeleteObject           = gdi32.NewProc("DeleteObject")
	pDeleteDC               = gdi32.NewProc("DeleteDC")
	pBitBlt                 = gdi32.NewProc("BitBlt")
	pCreateSolidBrush       = gdi32.NewProc("CreateSolidBrush")
	pCreatePen              = gdi32.NewProc("CreatePen")
	pCreateFontW            = gdi32.NewProc("CreateFontW")
	pSetTextColor           = gdi32.NewProc("SetTextColor")
	pSetBkMode              = gdi32.NewProc("SetBkMode")
	pMoveToEx               = gdi32.NewProc("MoveToEx")
	pLineTo                 = gdi32.NewProc("LineTo")
	pPolygon                = gdi32.NewProc("Polygon")
	pGetDeviceCaps          = gdi32.NewProc("GetDeviceCaps")

	pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	pGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	pGlobalLock       = kernel32.NewProc("GlobalLock")
	pGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
	pShellExecuteW    = shell32.NewProc("ShellExecuteW")
)

const (
	ModeInstaller = iota
	ModeLauncher
)

// 4 Top-Level Tabs in the Graphical App
const (
	TabLiveCockpit = iota // 1. Use Dashboards LIVE inside the App!
	TabShards             // 2. Manage Shards / Dashboards
	TabHostLink           // 3. Server Toggle & Tablet Wi-Fi Link
	TabSettings           // 4. Settings & Uninstall
)

const (
	InstPageWelcome = iota
	InstPageProgress
	InstPageDone
	InstPageUninstall
)

// Control IDs
const (
	CtrlNone = iota
	CtrlWinClose
	CtrlWinMin

	// Sidebar Navigation
	CtrlNavLiveCockpit
	CtrlNavShards
	CtrlNavHostLink
	CtrlNavSettings

	// Live Cockpit Tab Controls
	CtrlLivePrevBoard
	CtrlLiveNextBoard
	CtrlLiveArmToggle
	CtrlLiveOpenEditor

	// Shards Manager Controls
	CtrlBtnNewDash
	CtrlBtnSyncSCBindings
	CtrlBtnOpenDashFolder
	CtrlBtnReloadDash

	// Host Link / Server Controls
	CtrlBtnToggleServer
	CtrlBtnCopyURL
	CtrlBtnOpenBrowser
	CtrlBtnOpenEditor
	CtrlBtnRestartAdmin

	// Settings Controls
	CtrlBtnPortMinus
	CtrlBtnPortPlus
	CtrlBtnOpenUserFolder
	CtrlBtnOpenAppFolder
	CtrlBtnUpdateRepair
	CtrlBtnShowUninstall
	CtrlChkPurgeData
	CtrlBtnConfirmUninstall
	CtrlBtnCancelUninstall

	// Installer Mode Controls
	CtrlInstChkDesktop
	CtrlInstChkStartMenu
	CtrlInstChkWingetPy
	CtrlInstPrimary
	CtrlInstSecondary
	CtrlInstSwitchUninstall

	// Dynamic Ranges
	CtrlDashUseLiveBase  = 1000 // + index (Opens dashboard directly in Live Cockpit tab)
	CtrlDashEditBase     = 2000 // + index
	CtrlDashDupBase      = 3000 // + index
	CtrlDashDelBase      = 4000 // + index
	CtrlLivePageTabBase  = 5000 // + page index
	CtrlLiveTileBtnBase  = 6000 // + flat button index in current page
)

type HotRegion struct {
	ID   int
	Rect RECT
}

type DashButton struct {
	ID     string   `json:"id"`
	Kind   string   `json:"kind"`
	Label  string   `json:"label"`
	Hint   string   `json:"hint"`
	Keys   []string `json:"keys"`
	Mode   string   `json:"mode"`
	Color  string   `json:"color"`
	Wide   bool     `json:"wide"`
	Span   int      `json:"span"`
	Danger bool     `json:"danger"`
}

type DashGroup struct {
	Title   string       `json:"title"`
	Buttons []DashButton `json:"buttons"`
}

type DashPage struct {
	ID     string      `json:"id"`
	Label  string      `json:"label"`
	Groups []DashGroup `json:"groups"`
}

type DashboardInfo struct {
	ID          string
	Title       string
	Source      string
	Path        string
	PageCount   int
	ButtonCount int
	Pages       []DashPage
}

type AppState struct {
	mu          sync.Mutex
	Hwnd        uintptr
	Scale       float64
	Width       int32
	Height      int32
	Mode        int
	Tab         int
	InstPage    int
	IsUninstall bool

	// Paths & Environment
	RootDir      string
	AppDir       string
	UserDir      string
	UserDashDir  string
	ExistingInst bool
	PythonFound  bool
	PythonCmd    string
	PythonVer    string
	WingetFound  bool
	IsAdmin      bool

	// Server Process & Network
	ServerRunning bool
	ServerCmd     *os.Process
	Port          int
	LanIP         string
	ServerLogs    []string
	ToastMsg      string

	// Dashboards & In-App Live Cockpit Deck State
	Dashboards    []DashboardInfo
	ActiveDash    string
	LivePageIdx   int
	LiveArmed     bool
	LiveToggled   map[string]bool
	LivePageBtns  []DashButton // Flattened buttons on current Live Cockpit page
	LastActionTxt string

	// Installer Options
	OptDesktop    bool
	OptStartMenu  bool
	OptWingetPy   bool
	OptPurgeData  bool
	ConfirmUninst bool
	ProgressPct   int
	StatusLine    string
	InstallErr    string

	// Mouse & UI
	HoverID int
	PressID int
	Regions []HotRegion

	// Fonts & Cursors
	CursorArrow uintptr
	CursorHand  uintptr
	FontHero    uintptr
	FontTitle   uintptr
	FontSub     uintptr
	FontBody    uintptr
	FontBold    uintptr
	FontMono    uintptr
	FontSmall   uintptr
}

var app = &AppState{
	OptDesktop:   true,
	OptStartMenu: true,
	Port:         8765,
	LanIP:        "127.0.0.1",
	LiveToggled:  make(map[string]bool),
}

func u16(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func u16ptr(s string) uintptr {
	p, _ := syscall.UTF16PtrFromString(s)
	return uintptr(unsafe.Pointer(p))
}

func sc(v int32) int32 {
	return int32(float64(v)*app.Scale + 0.5)
}

func ptInRect(x, y int32, r RECT) bool {
	return x >= r.Left && x < r.Right && y >= r.Top && y < r.Bottom
}

func copyToClipboard(text string) {
	u, err := syscall.UTF16FromString(text)
	if err != nil {
		return
	}
	r, _, _ := pOpenClipboard.Call(app.Hwnd)
	if r == 0 {
		return
	}
	defer pCloseClipboard.Call()
	pEmptyClipboard.Call()
	size := uintptr(len(u) * 2)
	hMem, _, _ := pGlobalAlloc.Call(0x0002, size)
	if hMem == 0 {
		return
	}
	ptr, _, _ := pGlobalLock.Call(hMem)
	if ptr == 0 {
		return
	}
	dst := unsafe.Slice((*uint16)(unsafe.Pointer(ptr)), len(u))
	copy(dst, u)
	pGlobalUnlock.Call(hMem)
	pSetClipboardData.Call(13, hMem)
}

// ============================================================================
// GAMEGLASS CHAMFERED SHARD DRAWING PRIMITIVES
// ============================================================================
func fillRect(hdc uintptr, r RECT, col uint32) {
	brush, _, _ := pCreateSolidBrush.Call(uintptr(col))
	pFillRect.Call(hdc, uintptr(unsafe.Pointer(&r)), brush)
	pDeleteObject.Call(brush)
}

// GameGlass Beveled Shard Box (diagonal top-right and bottom-left cut corners)
func drawShardBox(hdc uintptr, r RECT, cut int32, fillCol, borderCol uint32) {
	c := sc(cut)
	pts := []POINT{
		{r.Left, r.Top},
		{r.Right - c, r.Top},
		{r.Right, r.Top + c},
		{r.Right, r.Bottom},
		{r.Left + c, r.Bottom},
		{r.Left, r.Bottom - c},
	}
	brush, _, _ := pCreateSolidBrush.Call(uintptr(fillCol))
	pen, _, _ := pCreatePen.Call(PS_SOLID, uintptr(sc(1)), uintptr(borderCol))
	oldB, _, _ := pSelectObject.Call(hdc, brush)
	oldP, _, _ := pSelectObject.Call(hdc, pen)
	pPolygon.Call(hdc, uintptr(unsafe.Pointer(&pts[0])), uintptr(len(pts)))
	pSelectObject.Call(hdc, oldB)
	pSelectObject.Call(hdc, oldP)
	pDeleteObject.Call(brush)
	pDeleteObject.Call(pen)
}

func drawLine(hdc uintptr, x1, y1, x2, y2 int32, col uint32, width int32) {
	pen, _, _ := pCreatePen.Call(PS_SOLID, uintptr(width), uintptr(col))
	oldP, _, _ := pSelectObject.Call(hdc, pen)
	pMoveToEx.Call(hdc, uintptr(x1), uintptr(y1), 0)
	pLineTo.Call(hdc, uintptr(x2), uintptr(y2))
	pSelectObject.Call(hdc, oldP)
	pDeleteObject.Call(pen)
}

func drawText(hdc uintptr, text string, r RECT, font uintptr, col uint32, flags uint32) {
	pSelectObject.Call(hdc, font)
	pSetTextColor.Call(hdc, uintptr(col))
	pSetBkMode.Call(hdc, TRANSPARENT)
	u, _ := syscall.UTF16FromString(text)
	if len(u) <= 1 {
		return
	}
	rc := r
	pDrawTextW.Call(hdc, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&rc)), uintptr(flags|DT_NOPREFIX))
}

// GameGlass Holographic Button
func drawButton(hdc uintptr, id int, r RECT, label string, style string) {
	app.Regions = append(app.Regions, HotRegion{ID: id, Rect: r})
	hov := app.HoverID == id
	press := app.PressID == id

	var fill, border, textCol, barCol uint32
	switch style {
	case "amber":
		fill = colAmber
		border = colAmber
		textCol = colBg
		barCol = colText
		if hov {
			fill = colAmberHov
			border = colAmberHov
		}
	case "cyan":
		fill = colCyanBg
		border = colCyan
		textCol = colCyan
		barCol = colCyan
		if hov {
			fill = rgb(0x10, 0x3b, 0x4d)
			textCol = colText
		}
	case "danger":
		fill = colRedBg
		border = colRed
		textCol = colRed
		barCol = colRed
		if hov {
			fill = colRed
			textCol = colText
		}
	case "green":
		fill = colGreenBg
		border = colGreen
		textCol = colGreen
		barCol = colGreen
		if hov {
			fill = rgb(0x12, 0x3d, 0x26)
		}
	default:
		fill = colPanel
		border = colLineHi
		textCol = colText
		barCol = colCyan
		if hov {
			fill = colPanelHov
			border = colCyan
		}
	}

	if press {
		r.Top += sc(1)
		r.Bottom += sc(1)
	}

	drawShardBox(hdc, r, 8, fill, border)
	if style != "amber" {
		fillRect(hdc, RECT{r.Left, r.Top + sc(5), r.Left + sc(3), r.Bottom - sc(8)}, barCol)
	}
	drawText(hdc, label, r, app.FontBold, textCol, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
}

func drawCheckboxRow(hdc uintptr, id int, r RECT, title, subtitle string, checked bool) {
	app.Regions = append(app.Regions, HotRegion{ID: id, Rect: r})
	hov := app.HoverID == id

	fill := colPanel
	border := colLine
	if hov {
		fill = colPanelHov
		border = colLineHi
	}
	if checked {
		border = colCyan
	}
	drawShardBox(hdc, r, 10, fill, border)
	if checked {
		fillRect(hdc, RECT{r.Left, r.Top + sc(6), r.Left + sc(3), r.Bottom - sc(10)}, colCyan)
	}

	boxSize := sc(20)
	bx := r.Left + sc(18)
	by := r.Top + (r.Bottom-r.Top-boxSize)/2
	boxRect := RECT{bx, by, bx + boxSize, by + boxSize}

	if checked {
		drawShardBox(hdc, boxRect, 4, colCyan, colCyan)
		drawLine(hdc, bx+sc(5), by+sc(10), bx+sc(9), by+sc(14), colBg, sc(2))
		drawLine(hdc, bx+sc(9), by+sc(14), bx+sc(15), by+sc(6), colBg, sc(2))
	} else {
		drawShardBox(hdc, boxRect, 4, colBg, colMuted)
	}

	tx := bx + boxSize + sc(16)
	drawText(hdc, title, RECT{tx, r.Top + sc(10), r.Right - sc(16), r.Top + sc(30)}, app.FontBold, colText, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, subtitle, RECT{tx, r.Top + sc(30), r.Right - sc(16), r.Bottom - sc(8)}, app.FontSmall, colMuted, DT_LEFT|DT_SINGLELINE)
}

func paintWindow(hdc uintptr, w, h int32) {
	app.Regions = app.Regions[:0]
	fillRect(hdc, RECT{0, 0, w, h}, colBg)

	if app.Mode == ModeInstaller {
		paintInstallerMode(hdc, w, h)
	} else {
		paintLauncherMode(hdc, w, h)
	}

	// Top-Right Window Controls
	btnW := sc(36)
	btnH := sc(28)
	closeRect := RECT{w - sc(14) - btnW, sc(12), w - sc(14), sc(12) + btnH}
	minRect := RECT{closeRect.Left - sc(6) - btnW, sc(12), closeRect.Left - sc(6), sc(12) + btnH}

	app.Regions = append(app.Regions, HotRegion{ID: CtrlWinMin, Rect: minRect})
	app.Regions = append(app.Regions, HotRegion{ID: CtrlWinClose, Rect: closeRect})

	if app.HoverID == CtrlWinMin {
		drawShardBox(hdc, minRect, 5, colPanelHov, colLine)
	}
	drawText(hdc, "—", minRect, app.FontBold, colMuted, DT_CENTER|DT_VCENTER|DT_SINGLELINE)

	closeCol := colMuted
	if app.HoverID == CtrlWinClose {
		drawShardBox(hdc, closeRect, 5, colRed, colRed)
		closeCol = colText
	}
	drawText(hdc, "✕", closeRect, app.FontBold, closeCol, DT_CENTER|DT_VCENTER|DT_SINGLELINE)

	// Top glowing GameGlass Cyan accent line
	drawLine(hdc, 0, 0, sc(260), 0, colCyan, sc(2))
	drawLine(hdc, sc(260), 0, w, 0, colLine, 1)
	drawLine(hdc, 0, h-1, w, h-1, colLine, 1)
	drawLine(hdc, 0, 0, 0, h, colLine, 1)
	drawLine(hdc, w-1, 0, w-1, h, colLine, 1)
}

// ============================================================================
// 1. GAMEGLASS INSTALLER UI
// ============================================================================
func paintInstallerMode(hdc uintptr, w, h int32) {
	sideW := sc(240)
	fillRect(hdc, RECT{0, 0, sideW, h}, colSidebar)
	drawLine(hdc, sideW, 0, sideW, h, colLine, 1)

	// GameGlass Hex Badge + Brand
	hexRect := RECT{sc(24), sc(24), sc(56), sc(56)}
	drawShardBox(hdc, hexRect, 8, colCyanBg, colCyan)
	drawText(hdc, "SC", hexRect, app.FontBold, colCyan, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
	drawText(hdc, "SC COCKPIT", RECT{sc(68), sc(23), sideW - sc(12), sc(42)}, app.FontTitle, colText, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, "GAMEGLASS SHARD SETUP", RECT{sc(68), sc(43), sideW - sc(12), sc(58)}, app.FontSmall, colCyan, DT_LEFT|DT_SINGLELINE)

	steps := []string{"01 // KONFIGURATION", "02 // DEPLOYMENT", "03 // COCKPIT BEREIT"}
	sy := sc(105)
	for i, st := range steps {
		active := (app.InstPage == InstPageWelcome && i == 0) ||
			(app.InstPage == InstPageProgress && i == 1) ||
			(app.InstPage == InstPageDone && i == 2)
		r := RECT{sc(14), sy, sideW - sc(14), sy + sc(38)}
		col := colMuted
		if active {
			col = colCyan
			drawShardBox(hdc, r, 8, colCyanBg, colCyan)
			fillRect(hdc, RECT{r.Left, r.Top + sc(6), r.Left + sc(3), r.Bottom - sc(8)}, colCyan)
		}
		drawText(hdc, st, RECT{r.Left + sc(16), r.Top, r.Right - sc(8), r.Bottom}, app.FontBold, col, DT_LEFT|DT_VCENTER|DT_SINGLELINE)
		sy += sc(46)
	}

	drawText(hdc, fmt.Sprintf("BUILD %s\nIN-APP LIVE DECK & WI-FI\nKEIN ADMIN ERFORDERLICH", AppVersion),
		RECT{sc(24), h - sc(80), sideW - sc(16), h - sc(18)}, app.FontSmall, colDim, DT_LEFT|DT_WORDBREAK)

	cx := sideW + sc(36)
	cw := w - cx - sc(36)

	switch app.InstPage {
	case InstPageWelcome:
		title := "SC COCKPIT // GAMEGLASS EDITION"
		sub := "Installiert das interaktive Desktop- & Tablet-Shard-Deck für Star Citizen."
		if app.ExistingInst {
			title = "SC COCKPIT // UPDATE & VERWALTUNG"
			sub = "Bestehende Installation erkannt. Deine eigenen Shards & Dashboards bleiben erhalten."
		}
		drawText(hdc, title, RECT{cx, sc(30), cx + cw - sc(80), sc(58)}, app.FontHero, colText, DT_LEFT|DT_SINGLELINE)
		drawText(hdc, sub, RECT{cx, sc(60), cx + cw, sc(84)}, app.FontBody, colMuted, DT_LEFT|DT_SINGLELINE)

		sysCard := RECT{cx, sc(98), cx + cw, sc(168)}
		drawShardBox(hdc, sysCard, 10, colPanel, colLine)
		fillRect(hdc, RECT{sysCard.Left, sysCard.Top, sysCard.Left + sc(70), sysCard.Top + sc(2)}, colAmber)
		drawText(hdc, "ZIELVERZEICHNIS", RECT{sysCard.Left + sc(18), sysCard.Top + sc(12), sysCard.Left + sc(340), sysCard.Top + sc(28)}, app.FontSmall, colDim, DT_LEFT|DT_SINGLELINE)
		drawText(hdc, app.AppDir, RECT{sysCard.Left + sc(18), sysCard.Top + sc(30), sysCard.Left + sc(380), sysCard.Bottom - sc(12)}, app.FontMono, colText, DT_LEFT|DT_SINGLELINE)

		drawLine(hdc, sysCard.Left+sc(395), sysCard.Top+sc(12), sysCard.Left+sc(395), sysCard.Bottom-sc(12), colLine, 1)
		drawText(hdc, "TELEMETRIE & SERVER ENGINE", RECT{sysCard.Left + sc(415), sysCard.Top + sc(12), sysCard.Right - sc(16), sysCard.Top + sc(28)}, app.FontSmall, colDim, DT_LEFT|DT_SINGLELINE)
		if app.PythonFound {
			drawText(hdc, "●  "+app.PythonVer+" [ONLINE]", RECT{sysCard.Left + sc(415), sysCard.Top + sc(30), sysCard.Right - sc(16), sysCard.Bottom - sc(12)}, app.FontBold, colGreen, DT_LEFT|DT_SINGLELINE)
		} else {
			drawText(hdc, "●  In-App Deck bereit (Python optional via winget)", RECT{sysCard.Left + sc(415), sysCard.Top + sc(30), sysCard.Right - sc(16), sysCard.Bottom - sc(12)}, app.FontBold, colCyan, DT_LEFT|DT_SINGLELINE)
		}

		oy := sc(186)
		oh := sc(58)
		drawCheckboxRow(hdc, CtrlInstChkDesktop, RECT{cx, oy, cx + cw, oy + oh},
			"Desktop-Shard-Starter anlegen",
			"Erstellt 'SC Cockpit' auf deinem Desktop für direkten Zugriff auf das Live-Cockpit",
			app.OptDesktop)

		oy += oh + sc(12)
		drawCheckboxRow(hdc, CtrlInstChkStartMenu, RECT{cx, oy, cx + cw, oy + oh},
			"Startmenü-Ordner & Administrator-Modus erstellen",
			"Inklusive 'SC Cockpit (Administrator)' falls Star Citizen mit Admin-Rechten läuft",
			app.OptStartMenu)

		if !app.PythonFound && app.WingetFound {
			oy += oh + sc(12)
			drawCheckboxRow(hdc, CtrlInstChkWingetPy, RECT{cx, oy, cx + cw, oy + oh},
				"Python 3.12 automatisch via winget einrichten",
				"Wird benötigt, falls du die Shards zusätzlich auf einem Tablet/Handy im WLAN öffnen möchtest",
				app.OptWingetPy)
		}

		btnPrimary := RECT{w - sc(36) - sc(220), h - sc(68), w - sc(36), h - sc(22)}
		btnLabel := "JETZT INSTALLIEREN  ►"
		if app.ExistingInst {
			btnLabel = "AKTUALISIEREN  ►"
			btnUninst := RECT{cx, h - sc(68), cx + sc(155), h - sc(22)}
			drawButton(hdc, CtrlInstSwitchUninstall, btnUninst, "DEINSTALLIEREN", "danger")
			btnDirect := RECT{btnUninst.Right + sc(10), h - sc(68), btnUninst.Right + sc(195), h - sc(22)}
			drawButton(hdc, CtrlInstSecondary, btnDirect, "DIREKT ÖFFNEN", "cyan")
		} else {
			btnDirect := RECT{cx, h - sc(68), cx + sc(210), h - sc(22)}
			drawButton(hdc, CtrlInstSecondary, btnDirect, "PORTABEL TESTEN", "cyan")
		}
		drawButton(hdc, CtrlInstPrimary, btnPrimary, btnLabel, "amber")

	case InstPageUninstall:
		drawText(hdc, "SC COCKPIT ENTFERNEN", RECT{cx, sc(30), cx + cw - sc(80), sc(58)}, app.FontHero, colRed, DT_LEFT|DT_SINGLELINE)
		drawText(hdc, "Möchtest du SC Cockpit und alle Verknüpfungen von diesem PC entfernen?", RECT{cx, sc(60), cx + cw, sc(84)}, app.FontBody, colMuted, DT_LEFT|DT_SINGLELINE)

		warnCard := RECT{cx, sc(104), cx + cw, sc(180)}
		drawShardBox(hdc, warnCard, 10, colRedBg, colRed)
		drawText(hdc, "Der Programmordner sowie alle Desktop- und Startmenü-Einträge werden gelöscht.\nDeine eigenen Dashboards im Benutzerordner bleiben standardmäßig erhalten.",
			RECT{warnCard.Left + sc(18), warnCard.Top + sc(16), warnCard.Right - sc(18), warnCard.Bottom - sc(14)}, app.FontBody, colText, DT_LEFT|DT_WORDBREAK)

		drawCheckboxRow(hdc, CtrlChkPurgeData, RECT{cx, sc(200), cx + cw, sc(260)},
			"Auch alle eigenen Shards / Dashboards unwiderruflich löschen",
			fmt.Sprintf("Löscht zusätzlich %s", app.UserDir),
			app.OptPurgeData)

		btnBack := RECT{cx, h - sc(68), cx + sc(140), h - sc(22)}
		btnRemove := RECT{w - sc(36) - sc(215), h - sc(68), w - sc(36), h - sc(22)}
		drawButton(hdc, CtrlInstSwitchUninstall, btnBack, "◄ ZURÜCK", "secondary")
		drawButton(hdc, CtrlInstPrimary, btnRemove, "ENDGÜLTIG ENTFERNEN", "danger")

	case InstPageProgress:
		app.mu.Lock()
		pct := app.ProgressPct
		status := app.StatusLine
		app.mu.Unlock()

		title := "QUANTUM DEPLOYMENT LÄUFT..."
		if app.IsUninstall {
			title = "DEINSTALLATION LÄUFT..."
		}
		drawText(hdc, title, RECT{cx, sc(30), cx + cw, sc(58)}, app.FontHero, colCyan, DT_LEFT|DT_SINGLELINE)
		drawText(hdc, status, RECT{cx, sc(64), cx + cw, sc(90)}, app.FontBody, colText, DT_LEFT|DT_SINGLELINE)

		barBg := RECT{cx, sc(126), cx + cw, sc(148)}
		drawShardBox(hdc, barBg, 6, colPanel, colLineHi)
		fillW := ((barBg.Right - barBg.Left) * int32(pct)) / 100
		if fillW > sc(8) {
			drawShardBox(hdc, RECT{barBg.Left, barBg.Top, barBg.Left + fillW, barBg.Bottom}, 6, colCyan, colCyan)
		}
		drawText(hdc, fmt.Sprintf("%d %%", pct), RECT{cx, sc(158), cx + cw, sc(184)}, app.FontBold, colCyan, DT_RIGHT|DT_SINGLELINE)

	case InstPageDone:
		if app.InstallErr != "" {
			drawText(hdc, "FEHLER BEI DER AUSFÜHRUNG", RECT{cx, sc(30), cx + cw, sc(58)}, app.FontHero, colRed, DT_LEFT|DT_SINGLELINE)
			errCard := RECT{cx, sc(78), cx + cw, sc(178)}
			drawShardBox(hdc, errCard, 10, colRedBg, colRed)
			drawText(hdc, app.InstallErr, RECT{errCard.Left + sc(18), errCard.Top + sc(16), errCard.Right - sc(18), errCard.Bottom - sc(16)}, app.FontBody, colText, DT_LEFT|DT_WORDBREAK)
		} else if app.IsUninstall {
			drawText(hdc, "SYSTEM BEREINIGT", RECT{cx, sc(30), cx + cw, sc(58)}, app.FontHero, colCyan, DT_LEFT|DT_SINGLELINE)
			doneCard := RECT{cx, sc(78), cx + cw, sc(165)}
			drawShardBox(hdc, doneCard, 10, colPanel, colCyan)
			msg := "SC Cockpit wurde erfolgreich entfernt."
			if !app.OptPurgeData {
				msg += "\nDeine eigenen Dashboards unter %LOCALAPPDATA%\\SC Cockpit\\user wurden behalten."
			}
			drawText(hdc, msg, RECT{doneCard.Left + sc(18), doneCard.Top + sc(18), doneCard.Right - sc(18), doneCard.Bottom - sc(18)}, app.FontBody, colText, DT_LEFT|DT_WORDBREAK)
		}
		btnClose := RECT{w - sc(36) - sc(180), h - sc(68), w - sc(36), h - sc(22)}
		drawButton(hdc, CtrlInstPrimary, btnClose, "SCHLIESSEN", "amber")
	}
}

// ============================================================================
// 2. GAMEGLASS IN-APP COCKPIT & CONTROL CENTER
// ============================================================================
func paintLauncherMode(hdc uintptr, w, h int32) {
	// Left GameGlass Vertical Shard Rail
	sideW := sc(225)
	fillRect(hdc, RECT{0, 0, sideW, h}, colSidebar)
	drawLine(hdc, sideW, 0, sideW, h, colLine, 1)

	// Brand Header
	hexRect := RECT{sc(18), sc(18), sc(48), sc(48)}
	drawShardBox(hdc, hexRect, 7, colCyanBg, colCyan)
	drawText(hdc, "SC", hexRect, app.FontBold, colCyan, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
	drawText(hdc, "SC COCKPIT", RECT{sc(58), sc(17), sideW - sc(10), sc(36)}, app.FontTitle, colText, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, "GAMEGLASS SHARD HUB", RECT{sc(58), sc(36), sideW - sc(10), sc(51)}, app.FontSmall, colCyan, DT_LEFT|DT_SINGLELINE)

	navItems := []struct {
		id    int
		tab   int
		label string
		tag   string
	}{
		{CtrlNavLiveCockpit, TabLiveCockpit, "LIVE COCKPIT", "IN-APP"},
		{CtrlNavShards, TabShards, "SHARDS & DECKS", fmt.Sprintf("%d", len(app.Dashboards))},
		{CtrlNavHostLink, TabHostLink, "TABLET / SERVER", "WLAN"},
		{CtrlNavSettings, TabSettings, "EINSTELLUNGEN", "SYS"},
	}

	ny := sc(74)
	for _, item := range navItems {
		r := RECT{sc(12), ny, sideW - sc(12), ny + sc(44)}
		app.Regions = append(app.Regions, HotRegion{ID: item.id, Rect: r})
		active := app.Tab == item.tab
		hov := app.HoverID == item.id

		if active {
			drawShardBox(hdc, r, 8, colCyanBg, colCyan)
			fillRect(hdc, RECT{r.Left, r.Top + sc(6), r.Left + sc(3), r.Bottom - sc(8)}, colCyan)
		} else if hov {
			drawShardBox(hdc, r, 8, colPanelHov, colLineHi)
		} else {
			drawShardBox(hdc, r, 8, colPanel, colLine)
		}

		col := colMuted
		tagCol := colDim
		if active {
			col = colCyan
			tagCol = colAmber
		} else if hov {
			col = colText
		}
		drawText(hdc, item.label, RECT{r.Left + sc(14), r.Top, r.Right - sc(48), r.Bottom}, app.FontBold, col, DT_LEFT|DT_VCENTER|DT_SINGLELINE)
		drawText(hdc, item.tag, RECT{r.Right - sc(46), r.Top, r.Right - sc(10), r.Bottom}, app.FontMono, tagCol, DT_RIGHT|DT_VCENTER|DT_SINGLELINE)
		ny += sc(52)
	}

	// Bottom Sidebar Status Pod
	statusCard := RECT{sc(12), h - sc(108), sideW - sc(12), h - sc(14)}
	if app.ServerRunning {
		drawShardBox(hdc, statusCard, 8, colGreenBg, colGreen)
		fillRect(hdc, RECT{statusCard.Left, statusCard.Top + sc(6), statusCard.Left + sc(3), statusCard.Bottom - sc(8)}, colGreen)
		drawText(hdc, "●  HOST LINK ONLINE", RECT{statusCard.Left + sc(12), statusCard.Top + sc(8), statusCard.Right - sc(8), statusCard.Top + sc(26)}, app.FontBold, colGreen, DT_LEFT|DT_SINGLELINE)
	} else {
		drawShardBox(hdc, statusCard, 8, colPanel, colLine)
		drawText(hdc, "○  WLAN SERVER AUS", RECT{statusCard.Left + sc(12), statusCard.Top + sc(8), statusCard.Right - sc(8), statusCard.Top + sc(26)}, app.FontBold, colMuted, DT_LEFT|DT_SINGLELINE)
	}
	drawText(hdc, fmt.Sprintf("http://%s:%d/", app.LanIP, app.Port), RECT{statusCard.Left + sc(12), statusCard.Top + sc(28), statusCard.Right - sc(8), statusCard.Top + sc(46)}, app.FontMono, colCyan, DT_LEFT|DT_SINGLELINE)
	lastAct := app.LastActionTxt
	if lastAct == "" {
		lastAct = "Bereit für Eingaben"
	}
	drawText(hdc, "LETZTE: "+lastAct, RECT{statusCard.Left + sc(12), statusCard.Top + sc(50), statusCard.Right - sc(8), statusCard.Bottom - sc(8)}, app.FontSmall, colAmber, DT_LEFT|DT_SINGLELINE)

	// Main Stage Area
	cx := sideW + sc(24)
	cw := w - cx - sc(24)

	switch app.Tab {
	case TabLiveCockpit:
		paintTabLiveCockpit(hdc, cx, cw, h)
	case TabShards:
		paintTabShards(hdc, cx, cw, h)
	case TabHostLink:
		paintTabHostLink(hdc, cx, cw, h)
	case TabSettings:
		paintTabSettings(hdc, cx, cw, h)
	}

	if app.ToastMsg != "" {
		toastRect := RECT{cx, h - sc(42), cx + cw, h - sc(12)}
		drawShardBox(hdc, toastRect, 6, colCyanBg, colCyan)
		drawText(hdc, app.ToastMsg, toastRect, app.FontBold, colCyan, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
	}
}

// ============================================================================
// TAB 1: IN-APP INTERACTIVE LIVE COCKPIT (USE DASHBOARDS DIRECTLY IN THE APP!)
// ============================================================================
func paintTabLiveCockpit(hdc uintptr, cx, cw, h int32) {
	app.LivePageBtns = app.LivePageBtns[:0]

	// Find active dashboard
	var curDash *DashboardInfo
	for i := range app.Dashboards {
		if app.Dashboards[i].ID == app.ActiveDash {
			curDash = &app.Dashboards[i]
			break
		}
	}
	if curDash == nil && len(app.Dashboards) > 0 {
		curDash = &app.Dashboards[0]
		app.ActiveDash = curDash.ID
	}

	// Top MFD Deck Control Bar
	topBar := RECT{cx, sc(14), cx + cw - sc(84), sc(58)}
	drawShardBox(hdc, topBar, 8, colPanel, colLineHi)
	fillRect(hdc, RECT{topBar.Left, topBar.Top, topBar.Left + sc(65), topBar.Top + sc(2)}, colCyan)

	// Shard Selector Prev / Title / Next
	btnPrev := RECT{topBar.Left + sc(10), topBar.Top + sc(8), topBar.Left + sc(42), topBar.Bottom - sc(8)}
	drawButton(hdc, CtrlLivePrevBoard, btnPrev, "◄", "secondary")

	dashTitle := "KEIN SHARD"
	if curDash != nil {
		dashTitle = strings.ToUpper(curDash.Title) + " // LIVE SHARD"
	}
	drawText(hdc, dashTitle, RECT{btnPrev.Right + sc(8), topBar.Top, btnPrev.Right + sc(235), topBar.Bottom}, app.FontTitle, colAmber, DT_CENTER|DT_VCENTER|DT_SINGLELINE)

	btnNext := RECT{btnPrev.Right + sc(240), topBar.Top + sc(8), btnPrev.Right + sc(272), topBar.Bottom - sc(8)}
	drawButton(hdc, CtrlLiveNextBoard, btnNext, "►", "secondary")

	// Weapons Arm Toggle & Baukasten Editor Button
	btnEdit := RECT{topBar.Right - sc(145), topBar.Top + sc(8), topBar.Right - sc(10), topBar.Bottom - sc(8)}
	drawButton(hdc, CtrlLiveOpenEditor, btnEdit, "BAUKASTEN", "cyan")

	btnArm := RECT{btnEdit.Left - sc(148), topBar.Top + sc(8), btnEdit.Left - sc(8), topBar.Bottom - sc(8)}
	if app.LiveArmed {
		drawButton(hdc, CtrlLiveArmToggle, btnArm, "WAFFEN ARMED", "danger")
	} else {
		drawButton(hdc, CtrlLiveArmToggle, btnArm, "WAFFEN SAFE", "secondary")
	}

	if curDash == nil || len(curDash.Pages) == 0 {
		drawText(hdc, "Keine Seiten in diesem Dashboard gefunden.", RECT{cx, sc(120), cx + cw, sc(160)}, app.FontBody, colMuted, DT_CENTER|DT_SINGLELINE)
		return
	}

	if app.LivePageIdx >= len(curDash.Pages) || app.LivePageIdx < 0 {
		app.LivePageIdx = 0
	}

	// Page Tabs Row (GameGlass MFD Tabs)
	tabY := sc(68)
	tabH := sc(36)
	numPages := len(curDash.Pages)
	tabW := sc(155)
	if int32(numPages)*(tabW+sc(8)) > cw {
		tabW = (cw - int32(numPages)*sc(8)) / int32(maxInt(numPages, 1))
	}

	tx := cx
	for pi, page := range curDash.Pages {
		tr := RECT{tx, tabY, tx + tabW, tabY + tabH}
		id := CtrlLivePageTabBase + pi
		app.Regions = append(app.Regions, HotRegion{ID: id, Rect: tr})
		isAct := pi == app.LivePageIdx
		hov := app.HoverID == id

		if isAct {
			drawShardBox(hdc, tr, 7, colAmberBg, colAmber)
			fillRect(hdc, RECT{tr.Left, tr.Top, tr.Right - sc(10), tr.Top + sc(2)}, colAmber)
			drawText(hdc, strings.ToUpper(page.Label), tr, app.FontBold, colAmber, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
		} else if hov {
			drawShardBox(hdc, tr, 7, colPanelHov, colCyan)
			drawText(hdc, strings.ToUpper(page.Label), tr, app.FontBold, colText, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
		} else {
			drawShardBox(hdc, tr, 7, colPanel, colLine)
			drawText(hdc, strings.ToUpper(page.Label), tr, app.FontBold, colMuted, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
		}
		tx += tabW + sc(8)
	}

	// Render Active Page Groups & Interactive GameGlass Holo-Shards!
	activePage := curDash.Pages[app.LivePageIdx]
	gy := tabY + tabH + sc(12)
	maxBottom := h - sc(46)

	cols := int32(4)
	gap := sc(10)

	for _, grp := range activePage.Groups {
		if gy+sc(70) > maxBottom {
			break
		}
		// Group Header Line
		drawText(hdc, "◆  "+strings.ToUpper(grp.Title), RECT{cx, gy, cx + cw, gy + sc(20)}, app.FontBold, colCyan, DT_LEFT|DT_SINGLELINE)
		drawLine(hdc, cx+sc(170), gy+sc(10), cx+cw, gy+sc(10), colLine, 1)
		gy += sc(24)

		cellW := (cw - (cols-1)*gap) / cols
		cellH := sc(68)
		colIdx := int32(0)

		for _, btn := range grp.Buttons {
			// Skip pure display widgets or map them to quick action tiles
			span := int32(1)
			if btn.Wide || btn.Span >= 2 {
				span = 2
			}
			if colIdx+span > cols {
				colIdx = 0
				gy += cellH + gap
			}
			if gy+cellH > maxBottom {
				break
			}

			bx := cx + colIdx*(cellW+gap)
			bw := span*cellW + (span-1)*gap
			tileRect := RECT{bx, gy, bx + bw, gy + cellH}

			flatIdx := len(app.LivePageBtns)
			app.LivePageBtns = append(app.LivePageBtns, btn)
			ctrlID := CtrlLiveTileBtnBase + flatIdx

			paintGameGlassShardTile(hdc, ctrlID, tileRect, btn)

			colIdx += span
			if colIdx >= cols {
				colIdx = 0
				gy += cellH + gap
			}
		}
		if colIdx > 0 {
			gy += cellH + gap
		}
		gy += sc(6)
	}
}

func paintGameGlassShardTile(hdc uintptr, id int, r RECT, btn DashButton) {
	app.Regions = append(app.Regions, HotRegion{ID: id, Rect: r})
	hov := app.HoverID == id
	press := app.PressID == id
	isToggled := app.LiveToggled[btn.ID]

	// Determine Shard Accent Color
	var accent, fill, border uint32
	switch btn.Color {
	case "amber":
		accent = colAmber
		fill = rgb(0x19, 0x16, 0x0e)
		border = rgb(0x6e, 0x4d, 0x12)
	case "red":
		accent = colRed
		fill = rgb(0x1f, 0x0f, 0x14)
		border = rgb(0x75, 0x20, 0x2c)
	case "green":
		accent = colGreen
		fill = rgb(0x0d, 0x1f, 0x18)
		border = rgb(0x18, 0x6b, 0x3f)
	case "blue":
		accent = colBlue
		fill = colBlueBg
		border = rgb(0x23, 0x52, 0x96)
	case "cyan":
		accent = colCyan
		fill = rgb(0x0b, 0x1d, 0x29)
		border = rgb(0x14, 0x67, 0x78)
	default:
		accent = colSlate
		fill = colPanel
		border = colLineHi
	}

	if isToggled {
		fill = colCyanBg
		border = colCyan
		accent = colCyan
	} else if hov {
		fill = colPanelHov
		border = accent
	}

	if press {
		r.Top += sc(1)
		r.Bottom += sc(1)
	}

	drawShardBox(hdc, r, 10, fill, border)

	// Left laser glow bar
	fillRect(hdc, RECT{r.Left, r.Top + sc(7), r.Left + sc(3), r.Bottom - sc(10)}, accent)

	// Danger top warning stripe
	if btn.Danger {
		stripeCol := colRed
		if !app.LiveArmed {
			stripeCol = colDim
		}
		fillRect(hdc, RECT{r.Left + sc(6), r.Top, r.Right - sc(12), r.Top + sc(3)}, stripeCol)
	}

	// Top Row: Keybind Badge + Mode Tag
	keyHint := btn.Hint
	if keyHint == "" && len(btn.Keys) > 0 {
		keyHint = strings.ToUpper(strings.Join(btn.Keys, " + "))
	}
	modeTag := "TAP"
	if btn.Mode == "hold" {
		modeTag = "HOLD"
	} else if btn.Mode == "toggle" {
		if isToggled {
			modeTag = "ON ●"
		} else {
			modeTag = "TOGGLE"
		}
	}
	if btn.Danger && !app.LiveArmed {
		modeTag = "LOCKED"
	}

	drawText(hdc, keyHint, RECT{r.Left + sc(12), r.Top + sc(8), r.Right - sc(55), r.Top + sc(23)}, app.FontMono, accent, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, modeTag, RECT{r.Right - sc(65), r.Top + sc(8), r.Right - sc(10), r.Top + sc(23)}, app.FontSmall, colMuted, DT_RIGHT|DT_SINGLELINE)

	// Bottom Main Action Label
	labelCol := colText
	if btn.Danger && !app.LiveArmed {
		labelCol = colMuted
	} else if isToggled {
		labelCol = colCyan
	}
	drawText(hdc, strings.ToUpper(btn.Label), RECT{r.Left + sc(12), r.Top + sc(28), r.Right - sc(10), r.Bottom - sc(8)}, app.FontTitle, labelCol, DT_LEFT|DT_VCENTER|DT_SINGLELINE)
}

// ============================================================================
// TAB 2: SHARDS & DASHBOARDS MANAGER
// ============================================================================
func paintTabShards(hdc uintptr, cx, cw, h int32) {
	drawText(hdc, "SHARDS & DASHBOARDS", RECT{cx, sc(22), cx + cw - sc(100), sc(50)}, app.FontHero, colText, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, "Öffne Shards direkt im In-App Live-Cockpit, passe sie im Baukasten an oder synchronisiere deine Star Citizen Tasten.",
		RECT{cx, sc(52), cx + cw, sc(74)}, app.FontBody, colMuted, DT_LEFT|DT_SINGLELINE)

	btnNew := RECT{cx, sc(86), cx + sc(165), sc(122)}
	btnSync := RECT{btnNew.Right + sc(10), sc(86), btnNew.Right + sc(205), sc(122)}
	btnFolder := RECT{btnSync.Right + sc(10), sc(86), btnSync.Right + sc(175), sc(122)}
	btnReload := RECT{btnFolder.Right + sc(10), sc(86), btnFolder.Right + sc(125), sc(122)}

	drawButton(hdc, CtrlBtnNewDash, btnNew, "+ NEUER SHARD", "amber")
	drawButton(hdc, CtrlBtnSyncSCBindings, btnSync, "SC-TASTEN LESEN", "cyan")
	drawButton(hdc, CtrlBtnOpenDashFolder, btnFolder, "ORDNER ÖFFNEN", "secondary")
	drawButton(hdc, CtrlBtnReloadDash, btnReload, "RELOAD", "secondary")

	y := sc(138)
	rowH := sc(72)
	for i, d := range app.Dashboards {
		if y+rowH > h-sc(48) {
			break
		}
		card := RECT{cx, y, cx + cw, y + rowH}
		isActive := d.ID == app.ActiveDash
		border := colLine
		fill := colPanel
		if isActive {
			border = colCyan
			fill = colPanelAct
		}
		drawShardBox(hdc, card, 10, fill, border)
		if isActive {
			fillRect(hdc, RECT{card.Left, card.Top + sc(8), card.Left + sc(4), card.Bottom - sc(10)}, colCyan)
		}

		badge := "OFFICIAL SHARD"
		badgeCol := colCyan
		if d.Source == "local" {
			badge = "CUSTOM SHARD"
			badgeCol = colAmber
		}
		drawText(hdc, strings.ToUpper(d.Title), RECT{card.Left + sc(18), card.Top + sc(12), card.Left + sc(260), card.Top + sc(36)}, app.FontTitle, colText, DT_LEFT|DT_SINGLELINE)
		meta := fmt.Sprintf("%s   •   %s.json   •   %d Seiten   •   %d Shards", badge, d.ID, d.PageCount, d.ButtonCount)
		drawText(hdc, meta, RECT{card.Left + sc(18), card.Top + sc(38), card.Right - sc(360), card.Bottom - sc(10)}, app.FontSmall, badgeCol, DT_LEFT|DT_SINGLELINE)

		bx := card.Right - sc(14)
		if d.Source == "local" {
			btnDel := RECT{bx - sc(86), card.Top + sc(18), bx, card.Bottom - sc(18)}
			drawButton(hdc, CtrlDashDelBase+i, btnDel, "LÖSCHEN", "danger")
			bx = btnDel.Left - sc(8)
		}

		btnDup := RECT{bx - sc(88), card.Top + sc(18), bx, card.Bottom - sc(18)}
		drawButton(hdc, CtrlDashDupBase+i, btnDup, "KOPIE", "secondary")
		bx = btnDup.Left - sc(8)

		btnEdit := RECT{bx - sc(108), card.Top + sc(18), bx, card.Bottom - sc(18)}
		drawButton(hdc, CtrlDashEditBase+i, btnEdit, "BAUKASTEN", "cyan")
		bx = btnEdit.Left - sc(8)

		btnUse := RECT{bx - sc(135), card.Top + sc(18), bx, card.Bottom - sc(18)}
		if isActive {
			drawButton(hdc, CtrlDashUseLiveBase+i, btnUse, "► IM COCKPIT", "amber")
		} else {
			drawButton(hdc, CtrlDashUseLiveBase+i, btnUse, "IM COCKPIT NUTZEN", "green")
		}

		y += rowH + sc(10)
	}
}

// ============================================================================
// TAB 3: TABLET / WLAN SERVER HOST LINK
// ============================================================================
func paintTabHostLink(hdc uintptr, cx, cw, h int32) {
	drawText(hdc, "TABLET & WLAN SHARD SERVER", RECT{cx, sc(22), cx + cw - sc(90), sc(50)}, app.FontHero, colText, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, "Verbinde dein Tablet, Smartphone oder einen Zweitbildschirm im WLAN mit deinem Star Citizen PC.",
		RECT{cx, sc(52), cx + cw, sc(74)}, app.FontBody, colMuted, DT_LEFT|DT_SINGLELINE)

	heroCard := RECT{cx, sc(88), cx + cw, sc(226)}
	drawShardBox(hdc, heroCard, 12, colPanel, colLineHi)
	fillRect(hdc, RECT{heroCard.Left, heroCard.Top, heroCard.Left + sc(90), heroCard.Top + sc(2)}, colCyan)

	url := fmt.Sprintf("http://%s:%d/", app.LanIP, app.Port)

	if app.ServerRunning {
		drawText(hdc, "●  GAMEGLASS HOST SERVER IST ONLINE", RECT{heroCard.Left + sc(22), heroCard.Top + sc(16), heroCard.Right - sc(220), heroCard.Top + sc(36)}, app.FontBold, colGreen, DT_LEFT|DT_SINGLELINE)
	} else {
		drawText(hdc, "○  HOST SERVER IST GESTOPPT", RECT{heroCard.Left + sc(22), heroCard.Top + sc(16), heroCard.Right - sc(220), heroCard.Top + sc(36)}, app.FontBold, colAmber, DT_LEFT|DT_SINGLELINE)
	}

	drawText(hdc, "SHARD-ADRESSE FÜR TABLET / SMARTPHONE IM WLAN:", RECT{heroCard.Left + sc(22), heroCard.Top + sc(42), heroCard.Right - sc(220), heroCard.Top + sc(60)}, app.FontSmall, colMuted, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, url, RECT{heroCard.Left + sc(22), heroCard.Top + sc(60), heroCard.Right - sc(220), heroCard.Top + sc(92)}, app.FontHero, colCyan, DT_LEFT|DT_SINGLELINE)

	btnCopy := RECT{heroCard.Left + sc(22), heroCard.Bottom - sc(46), heroCard.Left + sc(172), heroCard.Bottom - sc(12)}
	btnOpen := RECT{btnCopy.Right + sc(10), heroCard.Bottom - sc(46), btnCopy.Right + sc(185), heroCard.Bottom - sc(12)}
	btnEdit := RECT{btnOpen.Right + sc(10), heroCard.Bottom - sc(46), btnOpen.Right + sc(185), heroCard.Bottom - sc(12)}

	drawButton(hdc, CtrlBtnCopyURL, btnCopy, "URL KOPIEREN", "secondary")
	drawButton(hdc, CtrlBtnOpenBrowser, btnOpen, "WEB-SHARD ÖFFNEN", "cyan")
	drawButton(hdc, CtrlBtnOpenEditor, btnEdit, "BAUKASTEN ÖFFNEN", "secondary")

	powerBtn := RECT{heroCard.Right - sc(195), heroCard.Top + sc(20), heroCard.Right - sc(22), heroCard.Bottom - sc(56)}
	if app.ServerRunning {
		drawButton(hdc, CtrlBtnToggleServer, powerBtn, "SERVER STOPPEN", "danger")
	} else {
		drawButton(hdc, CtrlBtnToggleServer, powerBtn, "SERVER STARTEN", "amber")
	}

	// Admin & Telemetry Log
	adminCard := RECT{cx, sc(240), cx + cw, sc(304)}
	drawShardBox(hdc, adminCard, 8, colPanel, colLine)
	if app.IsAdmin {
		drawText(hdc, "●  ADMINISTRATOR-BERECHTIGUNG AKTIV (Sendet Tasten auch an Star Citizen im Admin-Modus)",
			RECT{adminCard.Left + sc(18), adminCard.Top, adminCard.Right - sc(18), adminCard.Bottom}, app.FontBold, colGreen, DT_LEFT|DT_VCENTER|DT_SINGLELINE)
	} else {
		drawText(hdc, "Berechtigung: Standard-Benutzer (Falls Star Citizen als Admin läuft, bitte rechts klicken)",
			RECT{adminCard.Left + sc(18), adminCard.Top, adminCard.Right - sc(185), adminCard.Bottom}, app.FontBody, colMuted, DT_LEFT|DT_VCENTER|DT_SINGLELINE)
		adminBtn := RECT{adminCard.Right - sc(175), adminCard.Top + sc(14), adminCard.Right - sc(14), adminCard.Bottom - sc(14)}
		drawButton(hdc, CtrlBtnRestartAdmin, adminBtn, "ALS ADMIN STARTEN", "secondary")
	}

	logCard := RECT{cx, sc(318), cx + cw, h - sc(50)}
	drawShardBox(hdc, logCard, 10, rgb(0x04, 0x06, 0x0a), colLine)
	drawText(hdc, "LIVE TELEMETRIE- & TASTEN-PROTOKOLL", RECT{logCard.Left + sc(18), logCard.Top + sc(10), logCard.Right - sc(18), logCard.Top + sc(28)}, app.FontSmall, colDim, DT_LEFT|DT_SINGLELINE)

	app.mu.Lock()
	logs := append([]string(nil), app.ServerLogs...)
	app.mu.Unlock()

	ly := logCard.Top + sc(32)
	if len(logs) == 0 {
		drawText(hdc, "Bereit. Klicke auf eine Kachel im 'LIVE COCKPIT'-Tab oder auf deinem Tablet.",
			RECT{logCard.Left + sc(18), ly, logCard.Right - sc(18), ly + sc(22)}, app.FontMono, colDim, DT_LEFT|DT_SINGLELINE)
	} else {
		maxLines := int((logCard.Bottom - ly - sc(8)) / sc(19))
		if len(logs) > maxLines && maxLines > 0 {
			logs = logs[len(logs)-maxLines:]
		}
		for i, line := range logs {
			c := colMuted
			if i == len(logs)-1 {
				c = colCyan
			}
			drawText(hdc, line, RECT{logCard.Left + sc(18), ly, logCard.Right - sc(18), ly + sc(18)}, app.FontMono, c, DT_LEFT|DT_SINGLELINE)
			ly += sc(19)
		}
	}
}

// ============================================================================
// TAB 4: SETTINGS & UNINSTALL
// ============================================================================
func paintTabSettings(hdc uintptr, cx, cw, h int32) {
	drawText(hdc, "SYSTEM & EINSTELLUNGEN", RECT{cx, sc(22), cx + cw - sc(90), sc(50)}, app.FontHero, colText, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, "Netzwerk-Port anpassen, Verzeichnisse öffnen oder SC Cockpit deinstallieren.",
		RECT{cx, sc(52), cx + cw, sc(74)}, app.FontBody, colMuted, DT_LEFT|DT_SINGLELINE)

	portCard := RECT{cx, sc(88), cx + cw, sc(162)}
	drawShardBox(hdc, portCard, 10, colPanel, colLine)
	drawText(hdc, "WLAN Server-Port (Standard: 8765)", RECT{portCard.Left + sc(20), portCard.Top + sc(14), portCard.Left + sc(350), portCard.Top + sc(36)}, app.FontBold, colText, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, "Port für den GameGlass Web-Shard auf Tablet und Smartphone", RECT{portCard.Left + sc(20), portCard.Top + sc(38), portCard.Left + sc(450), portCard.Bottom - sc(12)}, app.FontSmall, colMuted, DT_LEFT|DT_SINGLELINE)

	btnMinus := RECT{portCard.Right - sc(180), portCard.Top + sc(18), portCard.Right - sc(140), portCard.Bottom - sc(18)}
	portBox := RECT{btnMinus.Right + sc(6), portCard.Top + sc(18), btnMinus.Right + sc(82), portCard.Bottom - sc(18)}
	btnPlus := RECT{portBox.Right + sc(6), portCard.Top + sc(18), portBox.Right + sc(46), portCard.Bottom - sc(18)}

	drawButton(hdc, CtrlBtnPortMinus, btnMinus, "—", "secondary")
	drawShardBox(hdc, portBox, 4, colBg, colCyan)
	drawText(hdc, fmt.Sprintf("%d", app.Port), portBox, app.FontBold, colCyan, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
	drawButton(hdc, CtrlBtnPortPlus, btnPlus, "+", "secondary")

	dirCard := RECT{cx, sc(176), cx + cw, sc(266)}
	drawShardBox(hdc, dirCard, 10, colPanel, colLine)
	drawText(hdc, "Speicherorte & Verknüpfungen", RECT{dirCard.Left + sc(20), dirCard.Top + sc(14), dirCard.Right - sc(20), dirCard.Top + sc(34)}, app.FontBold, colText, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, fmt.Sprintf("App: %s   |   Eigene Shards: %s", app.AppDir, app.UserDashDir),
		RECT{dirCard.Left + sc(20), dirCard.Top + sc(34), dirCard.Right - sc(20), dirCard.Top + sc(52)}, app.FontSmall, colMuted, DT_LEFT|DT_SINGLELINE)

	btnUserDir := RECT{dirCard.Left + sc(20), dirCard.Bottom - sc(36), dirCard.Left + sc(210), dirCard.Bottom - sc(8)}
	btnAppDir := RECT{btnUserDir.Right + sc(10), dirCard.Bottom - sc(36), btnUserDir.Right + sc(205), dirCard.Bottom - sc(8)}
	btnRepair := RECT{btnAppDir.Right + sc(10), dirCard.Bottom - sc(36), btnAppDir.Right + sc(215), dirCard.Bottom - sc(8)}

	drawButton(hdc, CtrlBtnOpenUserFolder, btnUserDir, "SHARD-ORDNER ÖFFNEN", "secondary")
	drawButton(hdc, CtrlBtnOpenAppFolder, btnAppDir, "APP-ORDNER ÖFFNEN", "secondary")
	drawButton(hdc, CtrlBtnUpdateRepair, btnRepair, "SHORTCUTS REPARIEREN", "secondary")

	uninstCard := RECT{cx, sc(280), cx + cw, sc(430)}
	drawShardBox(hdc, uninstCard, 10, colRedBg, colRed)
	drawText(hdc, "SC COCKPIT DEINSTALLIEREN", RECT{uninstCard.Left + sc(20), uninstCard.Top + sc(14), uninstCard.Right - sc(20), uninstCard.Top + sc(36)}, app.FontBold, colRed, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, "Entfernt SC Cockpit sowie alle Desktop- und Startmenü-Verknüpfungen von deinem Computer.",
		RECT{uninstCard.Left + sc(20), uninstCard.Top + sc(36), uninstCard.Right - sc(20), uninstCard.Top + sc(56)}, app.FontSmall, colText, DT_LEFT|DT_SINGLELINE)

	drawCheckboxRow(hdc, CtrlChkPurgeData,
		RECT{uninstCard.Left + sc(20), uninstCard.Top + sc(60), uninstCard.Right - sc(230), uninstCard.Bottom - sc(14)},
		"Eigene Shards & Dashboards ebenfalls löschen",
		"Wenn deaktiviert, bleiben deine eigenen Dashboards im Ordner 'user' erhalten",
		app.OptPurgeData)

	if !app.ConfirmUninst {
		btnUninst := RECT{uninstCard.Right - sc(210), uninstCard.Top + sc(72), uninstCard.Right - sc(20), uninstCard.Bottom - sc(22)}
		drawButton(hdc, CtrlBtnShowUninstall, btnUninst, "DEINSTALLIEREN...", "danger")
	} else {
		btnConfirm := RECT{uninstCard.Right - sc(210), uninstCard.Top + sc(60), uninstCard.Right - sc(20), uninstCard.Top + sc(96)}
		btnCancel := RECT{uninstCard.Right - sc(210), uninstCard.Top + sc(102), uninstCard.Right - sc(20), uninstCard.Bottom - sc(12)}
		drawButton(hdc, CtrlBtnConfirmUninstall, btnConfirm, "WIRKLICH ENTFERNEN!", "danger")
		drawButton(hdc, CtrlBtnCancelUninstall, btnCancel, "ABBRECHEN", "secondary")
	}
}

func handleControlClick(id int) {
	app.ToastMsg = ""

	// 1. In-App Live Cockpit Page Tab Click
	if id >= CtrlLivePageTabBase && id < CtrlLivePageTabBase+100 {
		app.LivePageIdx = id - CtrlLivePageTabBase
		pInvalidateRect.Call(app.Hwnd, 0, 0)
		return
	}

	// 2. In-App Live Cockpit Shard Tile Click -> Send Keybind directly to Star Citizen!
	if id >= CtrlLiveTileBtnBase && id < CtrlLiveTileBtnBase+500 {
		idx := id - CtrlLiveTileBtnBase
		if idx >= 0 && idx < len(app.LivePageBtns) {
			btn := app.LivePageBtns[idx]
			if btn.Danger && !app.LiveArmed {
				app.ToastMsg = "Waffe/Gefahr gesichert! Schalte oben rechts zuerst 'WAFFEN SAFE' auf 'WAFFEN ARMED'."
				pInvalidateRect.Call(app.Hwnd, 0, 0)
				return
			}
			if btn.Mode == "toggle" {
				app.LiveToggled[btn.ID] = !app.LiveToggled[btn.ID]
			}
			triggerInAppButton(btn)
		}
		pInvalidateRect.Call(app.Hwnd, 0, 0)
		return
	}

	// 3. Shards Manager List Actions
	if id >= CtrlDashUseLiveBase && id < CtrlDashUseLiveBase+100 {
		idx := id - CtrlDashUseLiveBase
		if idx < len(app.Dashboards) {
			setActiveDashboard(app.Dashboards[idx].ID)
			app.LivePageIdx = 0
			app.Tab = TabLiveCockpit
			app.ToastMsg = fmt.Sprintf("Shard '%s' im Live-Cockpit geladen.", app.Dashboards[idx].Title)
		}
		pInvalidateRect.Call(app.Hwnd, 0, 0)
		return
	}
	if id >= CtrlDashEditBase && id < CtrlDashEditBase+100 {
		idx := id - CtrlDashEditBase
		if idx < len(app.Dashboards) {
			setActiveDashboard(app.Dashboards[idx].ID)
			if !app.ServerRunning {
				startServer()
			}
			url := fmt.Sprintf("http://127.0.0.1:%d/", app.Port)
			pShellExecuteW.Call(0, u16ptr("open"), u16ptr(url), 0, 0, SW_SHOWNORMAL)
		}
		pInvalidateRect.Call(app.Hwnd, 0, 0)
		return
	}
	if id >= CtrlDashDupBase && id < CtrlDashDupBase+100 {
		idx := id - CtrlDashDupBase
		if idx < len(app.Dashboards) {
			newID, err := duplicateDashboard(app.Dashboards[idx])
			if err != nil {
				app.ToastMsg = "Fehler: " + err.Error()
			} else {
				app.ToastMsg = fmt.Sprintf("Shard-Kopie '%s.json' erstellt.", newID)
			}
		}
		pInvalidateRect.Call(app.Hwnd, 0, 0)
		return
	}
	if id >= CtrlDashDelBase && id < CtrlDashDelBase+100 {
		idx := id - CtrlDashDelBase
		if idx < len(app.Dashboards) {
			d := app.Dashboards[idx]
			if d.Source == "local" {
				_ = os.Remove(d.Path)
				reloadDashboards()
				app.ToastMsg = fmt.Sprintf("Custom Shard '%s' gelöscht.", d.Title)
			}
		}
		pInvalidateRect.Call(app.Hwnd, 0, 0)
		return
	}

	switch id {
	case CtrlWinClose:
		stopServer()
		pPostQuitMessage.Call(0)
	case CtrlInstSecondary:
		// Switch directly into the Live Cockpit without installing/reinstalling!
		enablePortableOrInstalledMode()
		app.Mode = ModeLauncher
		app.Tab = TabLiveCockpit
		reloadDashboards()
		startServer()
	case CtrlWinMin:
		pShowWindow.Call(app.Hwnd, SW_MINIMIZE)

	// Navigation Tabs
	case CtrlNavLiveCockpit:
		reloadDashboards()
		app.Tab = TabLiveCockpit
		app.ConfirmUninst = false
	case CtrlNavShards:
		reloadDashboards()
		app.Tab = TabShards
		app.ConfirmUninst = false
	case CtrlNavHostLink:
		app.Tab = TabHostLink
		app.ConfirmUninst = false
	case CtrlNavSettings:
		app.Tab = TabSettings
		app.ConfirmUninst = false

	// Live Cockpit Top Bar Controls
	case CtrlLivePrevBoard:
		cycleActiveDashboard(-1)
	case CtrlLiveNextBoard:
		cycleActiveDashboard(1)
	case CtrlLiveArmToggle:
		app.LiveArmed = !app.LiveArmed
		if app.LiveArmed {
			app.ToastMsg = "WAFFEN & GEFAHREN-TASTEN SIND JETZT SCHARF (ARMED)!"
		} else {
			app.ToastMsg = "Waffen & Gefahren-Tasten gesichert (SAFE)."
		}
	case CtrlLiveOpenEditor:
		if !app.ServerRunning {
			startServer()
		}
		url := fmt.Sprintf("http://127.0.0.1:%d/", app.Port)
		pShellExecuteW.Call(0, u16ptr("open"), u16ptr(url), 0, 0, SW_SHOWNORMAL)

	// Server Controls
	case CtrlBtnToggleServer:
		if app.ServerRunning {
			stopServer()
		} else {
			startServer()
		}
	case CtrlBtnCopyURL:
		url := fmt.Sprintf("http://%s:%d/", app.LanIP, app.Port)
		copyToClipboard(url)
		app.ToastMsg = "Shard-URL kopiert: " + url
	case CtrlBtnOpenBrowser, CtrlBtnOpenEditor:
		if !app.ServerRunning {
			startServer()
		}
		url := fmt.Sprintf("http://127.0.0.1:%d/", app.Port)
		pShellExecuteW.Call(0, u16ptr("open"), u16ptr(url), 0, 0, SW_SHOWNORMAL)
	case CtrlBtnRestartAdmin:
		stopServer()
		selfExe, _ := os.Executable()
		pShellExecuteW.Call(0, u16ptr("runas"), u16ptr(selfExe), u16ptr("--launcher"), u16ptr(app.AppDir), SW_SHOWNORMAL)
		pPostQuitMessage.Call(0)

	// Shards Manager Controls
	case CtrlBtnNewDash:
		newID, err := createNewDashboard()
		if err != nil {
			app.ToastMsg = "Fehler: " + err.Error()
		} else {
			app.ToastMsg = fmt.Sprintf("Neuer GameGlass Shard '%s.json' angelegt.", newID)
		}
	case CtrlBtnSyncSCBindings:
		app.ToastMsg = syncStarCitizenActionmaps()
	case CtrlBtnOpenDashFolder:
		_ = os.MkdirAll(app.UserDashDir, 0755)
		pShellExecuteW.Call(0, u16ptr("open"), u16ptr(app.UserDashDir), 0, 0, SW_SHOWNORMAL)
	case CtrlBtnReloadDash:
		reloadDashboards()
		app.ToastMsg = "Shards neu geladen."

	// Settings Controls
	case CtrlBtnPortMinus:
		if app.Port > 1024 {
			app.Port--
			saveConfigPort(app.Port)
			if app.ServerRunning {
				stopServer()
				startServer()
			}
		}
	case CtrlBtnPortPlus:
		if app.Port < 65530 {
			app.Port++
			saveConfigPort(app.Port)
			if app.ServerRunning {
				stopServer()
				startServer()
			}
		}
	case CtrlBtnOpenUserFolder:
		_ = os.MkdirAll(app.UserDir, 0755)
		pShellExecuteW.Call(0, u16ptr("open"), u16ptr(app.UserDir), 0, 0, SW_SHOWNORMAL)
	case CtrlBtnOpenAppFolder:
		pShellExecuteW.Call(0, u16ptr("open"), u16ptr(app.AppDir), 0, 0, SW_SHOWNORMAL)
	case CtrlBtnUpdateRepair:
		createShortcutsAndRegistry()
		app.ToastMsg = "Verknüpfungen wurden aktualisiert."
	case CtrlChkPurgeData:
		app.OptPurgeData = !app.OptPurgeData
	case CtrlBtnShowUninstall:
		app.ConfirmUninst = true
	case CtrlBtnCancelUninstall:
		app.ConfirmUninst = false
	case CtrlBtnConfirmUninstall:
		stopServer()
		app.Mode = ModeInstaller
		app.IsUninstall = true
		app.InstPage = InstPageProgress
		go runUninstallWorker()

	// Installer Mode Controls
	case CtrlInstChkDesktop:
		app.OptDesktop = !app.OptDesktop
	case CtrlInstChkStartMenu:
		app.OptStartMenu = !app.OptStartMenu
	case CtrlInstChkWingetPy:
		app.OptWingetPy = !app.OptWingetPy
	case CtrlInstSwitchUninstall:
		if app.InstPage == InstPageWelcome {
			app.InstPage = InstPageUninstall
		} else {
			app.InstPage = InstPageWelcome
		}
	case CtrlInstPrimary:
		if app.InstPage == InstPageWelcome {
			app.IsUninstall = false
			app.InstPage = InstPageProgress
			go runInstallWorker()
		} else if app.InstPage == InstPageUninstall {
			stopServer()
			app.IsUninstall = true
			app.InstPage = InstPageProgress
			go runUninstallWorker()
		} else if app.InstPage == InstPageDone {
			pPostQuitMessage.Call(0)
		}
	}

	pInvalidateRect.Call(app.Hwnd, 0, 0)
}

func wndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_ERASEBKGND:
		return 1

	case WM_PAINT:
		var ps PAINTSTRUCT
		hdc, _, _ := pBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		var rc RECT
		pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
		w := rc.Right - rc.Left
		h := rc.Bottom - rc.Top

		memDC, _, _ := pCreateCompatibleDC.Call(hdc)
		memBmp, _, _ := pCreateCompatibleBitmap.Call(hdc, uintptr(w), uintptr(h))
		oldBmp, _, _ := pSelectObject.Call(memDC, memBmp)

		paintWindow(memDC, w, h)

		pBitBlt.Call(hdc, 0, 0, uintptr(w), uintptr(h), memDC, 0, 0, SRCCOPY)
		pSelectObject.Call(memDC, oldBmp)
		pDeleteObject.Call(memBmp)
		pDeleteDC.Call(memDC)
		pEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0

	case WM_NCHITTEST:
		pt := POINT{int32(int16(lParam & 0xFFFF)), int32(int16((lParam >> 16) & 0xFFFF))}
		pScreenToClient.Call(hwnd, uintptr(unsafe.Pointer(&pt)))
		for _, reg := range app.Regions {
			if ptInRect(pt.X, pt.Y, reg.Rect) {
				return HTCLIENT
			}
		}
		if pt.Y < sc(56) {
			return HTCAPTION
		}
		return HTCLIENT

	case WM_SETCURSOR:
		if (lParam & 0xFFFF) == HTCLIENT {
			if app.HoverID != CtrlNone {
				pSetCursor.Call(app.CursorHand)
			} else {
				pSetCursor.Call(app.CursorArrow)
			}
			return 1
		}

	case WM_MOUSEMOVE:
		x := int32(int16(lParam & 0xFFFF))
		y := int32(int16((lParam >> 16) & 0xFFFF))
		newHover := CtrlNone
		for _, reg := range app.Regions {
			if ptInRect(x, y, reg.Rect) {
				newHover = reg.ID
				break
			}
		}
		if newHover != app.HoverID {
			app.HoverID = newHover
			pInvalidateRect.Call(hwnd, 0, 0)
		}
		return 0

	case WM_LBUTTONDOWN:
		x := int32(int16(lParam & 0xFFFF))
		y := int32(int16((lParam >> 16) & 0xFFFF))
		for _, reg := range app.Regions {
			if ptInRect(x, y, reg.Rect) {
				app.PressID = reg.ID
				pSetCapture.Call(hwnd)
				pInvalidateRect.Call(hwnd, 0, 0)
				return 0
			}
		}
		return 0

	case WM_LBUTTONUP:
		x := int32(int16(lParam & 0xFFFF))
		y := int32(int16((lParam >> 16) & 0xFFFF))
		pressed := app.PressID
		if pressed != CtrlNone {
			app.PressID = CtrlNone
			pReleaseCapture.Call()
			for _, reg := range app.Regions {
				if reg.ID == pressed && ptInRect(x, y, reg.Rect) {
					handleControlClick(pressed)
					return 0
				}
			}
			pInvalidateRect.Call(hwnd, 0, 0)
		}
		return 0

	case WM_KEYDOWN:
		if wParam == VK_ESCAPE {
			stopServer()
			pPostQuitMessage.Call(0)
		}
		return 0

	case WM_APP_REFRESH:
		pInvalidateRect.Call(hwnd, 0, 0)
		return 0

	case WM_APP_DONE:
		if !app.IsUninstall && app.InstallErr == "" {
			app.Mode = ModeLauncher
			app.Tab = TabLiveCockpit
			reloadDashboards()
			startServer()
		} else {
			app.InstPage = InstPageDone
		}
		pInvalidateRect.Call(hwnd, 0, 0)
		return 0

	case WM_CLOSE, WM_DESTROY:
		stopServer()
		pPostQuitMessage.Call(0)
		return 0
	}

	ret, _, _ := pDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func makeFont(height int32, weight int32, face string) uintptr {
	f, _, _ := pCreateFontW.Call(
		uintptr(int(-sc(height))),
		0, 0, 0,
		uintptr(weight),
		0, 0, 0,
		0, 0, 0,
		CLEARTYPE_QUALITY,
		0,
		uintptr(unsafe.Pointer(u16(face))),
	)
	return f
}

func main() {
	runtime.LockOSThread()
	pSetProcessDPIAware.Call()

	initEnvironment()

	screenDC, _, _ := user32.NewProc("GetDC").Call(0)
	dpi, _, _ := pGetDeviceCaps.Call(screenDC, 88)
	user32.NewProc("ReleaseDC").Call(0, screenDC)
	if dpi < 96 {
		dpi = 96
	}
	app.Scale = float64(dpi) / 96.0

	app.Width = sc(1020)
	app.Height = sc(640)

	app.CursorArrow, _, _ = pLoadCursorW.Call(0, IDC_ARROW)
	app.CursorHand, _, _ = pLoadCursorW.Call(0, IDC_HAND)

	app.FontHero = makeFont(20, FW_BOLD, "Bahnschrift")
	app.FontTitle = makeFont(14, FW_BOLD, "Bahnschrift")
	app.FontSub = makeFont(13, FW_SEMIBOLD, "Segoe UI")
	app.FontBody = makeFont(13, FW_NORMAL, "Segoe UI")
	app.FontBold = makeFont(12, FW_BOLD, "Bahnschrift")
	app.FontMono = makeFont(11, FW_SEMIBOLD, "Consolas")
	app.FontSmall = makeFont(11, FW_NORMAL, "Segoe UI")

	hInst, _, _ := pGetModuleHandleW.Call(0)
	className := u16("SCCockpitGameGlassHub")
	hIcon, _, _ := pLoadIconW.Call(hInst, 1)

	wc := WNDCLASSEXW{
		Size:      uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		Style:     CS_HREDRAW | CS_VREDRAW | CS_DROPSHADOW,
		WndProc:   syscall.NewCallback(wndProc),
		Instance:  hInst,
		Icon:      hIcon,
		IconSm:    hIcon,
		Cursor:    app.CursorArrow,
		ClassName: className,
	}
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	screenW, _, _ := pGetSystemMetrics.Call(0)
	screenH, _, _ := pGetSystemMetrics.Call(1)
	winX := (int32(screenW) - app.Width) / 2
	winY := (int32(screenH) - app.Height) / 2

	// WS_EX_NOACTIVATE (0x08000000) is NOT set on the main window so user can interact normally,
	// and when clicking a Live Cockpit tile we automatically restore focus to Star Citizen!
	hwnd, _, _ := pCreateWindowExW.Call(
		WS_EX_APPWINDOW,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(u16("SC Cockpit // GameGlass Shard Hub"))),
		WS_POPUP|WS_VISIBLE|WS_SYSMENU|WS_MINIMIZEBOX|WS_CLIPCHILDREN,
		uintptr(winX), uintptr(winY),
		uintptr(app.Width), uintptr(app.Height),
		0, 0, hInst, 0,
	)
	app.Hwnd = hwnd

	if app.Mode == ModeLauncher {
		reloadDashboards()
		startServer()
	}

	pShowWindow.Call(hwnd, SW_SHOWNORMAL)
	pUpdateWindow.Call(hwnd)

	var msg MSG
	for {
		ret, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
	stopServer()
}
