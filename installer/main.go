package main

import (
	"fmt"
	"os"
	"runtime"
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

var (
	colBg       = rgb(0x0b, 0x0e, 0x13) // #0b0e13 Deep Space Background
	colSidebar  = rgb(0x0e, 0x13, 0x1a) // #0e131a RSI Sidebar Dark
	colPanel    = rgb(0x13, 0x1a, 0x24) // #131a24 Card / Panel Surface
	colPanelHov = rgb(0x1a, 0x23, 0x30) // #1a2330 Hover Surface
	colPanelAct = rgb(0x1d, 0x28, 0x38) // #1d2838 Active Surface
	colLine     = rgb(0x22, 0x2d, 0x3d) // #222d3d Subtle Border
	colLineHi   = rgb(0x33, 0x45, 0x5c) // #33455c Highlight Border
	colText     = rgb(0xe9, 0xef, 0xf7) // #e9eff7 Primary Text
	colMuted    = rgb(0x88, 0x99, 0xae) // #8899ae Secondary Text
	colDim      = rgb(0x52, 0x62, 0x75) // #526275 Muted Label
	colAmber    = rgb(0xe2, 0xa2, 0x3a) // #e2a23a Star Citizen Amber
	colAmberHov = rgb(0xf0, 0xb2, 0x4c) // #f0b24c Amber Hover
	colAmberBg  = rgb(0x27, 0x1e, 0x11) // #271e11 Amber Card Fill
	colCyan     = rgb(0x3e, 0xc8, 0xc4) // #3ec8c4 Star Citizen Cyan
	colCyanBg   = rgb(0x11, 0x25, 0x29) // #112529 Cyan Card Fill
	colGreen    = rgb(0x4e, 0xc9, 0x78) // #4ec978 Online Green
	colGreenBg  = rgb(0x11, 0x26, 0x1a) // #11261a Online Fill
	colRed      = rgb(0xe1, 0x5b, 0x4c) // #e15b4c Danger Red
	colRedBg    = rgb(0x2a, 0x14, 0x14) // #2a1414 Danger Fill
)

// Win32 Constants
const (
	WS_POPUP        = 0x80000000
	WS_VISIBLE      = 0x10000000
	WS_SYSMENU      = 0x00080000
	WS_MINIMIZEBOX  = 0x00020000
	WS_CLIPCHILDREN = 0x02000000
	WS_EX_APPWINDOW = 0x00040000

	WM_DESTROY      = 0x0002
	WM_PAINT        = 0x000F
	WM_CLOSE        = 0x0010
	WM_ERASEBKGND   = 0x0014
	WM_SETCURSOR    = 0x0020
	WM_NCHITTEST    = 0x0084
	WM_KEYDOWN      = 0x0100
	WM_TIMER        = 0x0113
	WM_MOUSEMOVE    = 0x0200
	WM_LBUTTONDOWN  = 0x0201
	WM_LBUTTONUP    = 0x0202
	WM_APP_REFRESH  = 0x8001
	WM_APP_DONE     = 0x8002

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
	VK_RETURN = 0x0D

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
	pSetTimer           = user32.NewProc("SetTimer")
	pKillTimer          = user32.NewProc("KillTimer")
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
	pRoundRect              = gdi32.NewProc("RoundRect")
	pGetDeviceCaps          = gdi32.NewProc("GetDeviceCaps")

	pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	pGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	pGlobalLock       = kernel32.NewProc("GlobalLock")
	pGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
	pShellExecuteW    = shell32.NewProc("ShellExecuteW")
)

// Modes and Navigation Tabs
const (
	ModeInstaller = iota
	ModeLauncher
)

const (
	TabServer = iota
	TabDashboards
	TabSettings
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

	// Sidebar Tabs (Launcher Mode)
	CtrlNavServer
	CtrlNavDashboards
	CtrlNavSettings

	// Server Tab Controls
	CtrlBtnToggleServer
	CtrlBtnCopyURL
	CtrlBtnOpenBrowser
	CtrlBtnOpenEditor
	CtrlBtnRestartAdmin

	// Dashboards Tab Controls
	CtrlBtnNewDash
	CtrlBtnOpenDashFolder
	CtrlBtnReloadDash

	// Settings Tab Controls
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

	CtrlDashActivateBase = 1000 // + index
	CtrlDashEditBase     = 2000 // + index
	CtrlDashDupBase      = 3000 // + index
	CtrlDashDelBase      = 4000 // + index
)

type HotRegion struct {
	ID   int
	Rect RECT
}

type DashboardInfo struct {
	ID          string
	Title       string
	Source      string // "builtin" or "local"
	Path        string
	PageCount   int
	ButtonCount int
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

	// Dashboards Catalog
	Dashboards []DashboardInfo
	ActiveDash string

	// Installer Options & Progress
	OptDesktop   bool
	OptStartMenu bool
	OptWingetPy  bool
	OptPurgeData bool
	ConfirmUninst bool
	ProgressPct  int
	StatusLine   string
	InstallErr   string

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
	hMem, _, _ := pGlobalAlloc.Call(0x0002, size) // GMEM_MOVEABLE
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
	pSetClipboardData.Call(13, hMem) // CF_UNICODETEXT = 13
}

// Drawing Primitives
func fillRect(hdc uintptr, r RECT, col uint32) {
	brush, _, _ := pCreateSolidBrush.Call(uintptr(col))
	pFillRect.Call(hdc, uintptr(unsafe.Pointer(&r)), brush)
	pDeleteObject.Call(brush)
}

func drawCard(hdc uintptr, r RECT, fillCol, borderCol uint32, radius int32) {
	brush, _, _ := pCreateSolidBrush.Call(uintptr(fillCol))
	pen, _, _ := pCreatePen.Call(PS_SOLID, uintptr(sc(1)), uintptr(borderCol))
	oldB, _, _ := pSelectObject.Call(hdc, brush)
	oldP, _, _ := pSelectObject.Call(hdc, pen)
	rad := sc(radius)
	pRoundRect.Call(hdc, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right), uintptr(r.Bottom), uintptr(rad), uintptr(rad))
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

// RSI Launcher Style Button
func drawButton(hdc uintptr, id int, r RECT, label string, style string) {
	app.Regions = append(app.Regions, HotRegion{ID: id, Rect: r})
	hov := app.HoverID == id
	press := app.PressID == id

	var fill, border, textCol uint32
	switch style {
	case "amber": // Primary RSI Launch Button
		fill = colAmber
		border = colAmber
		textCol = rgb(0x0b, 0x0e, 0x13)
		if hov {
			fill = colAmberHov
			border = colAmberHov
		}
	case "cyan":
		fill = colCyanBg
		border = colCyan
		textCol = colCyan
		if hov {
			fill = rgb(0x18, 0x35, 0x3b)
			textCol = colText
		}
	case "danger":
		fill = colRedBg
		border = colRed
		textCol = colRed
		if hov {
			fill = colRed
			textCol = colText
		}
	case "green":
		fill = colGreenBg
		border = colGreen
		textCol = colGreen
		if hov {
			fill = rgb(0x19, 0x38, 0x26)
		}
	default: // "secondary"
		fill = colPanel
		border = colLineHi
		textCol = colText
		if hov {
			fill = colPanelHov
			border = colMuted
		}
	}

	if press {
		r.Top += sc(1)
		r.Bottom += sc(1)
	}

	drawCard(hdc, r, fill, border, 8)
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
		border = colAmber
	}
	drawCard(hdc, r, fill, border, 10)

	boxSize := sc(20)
	bx := r.Left + sc(18)
	by := r.Top + (r.Bottom-r.Top-boxSize)/2
	boxRect := RECT{bx, by, bx + boxSize, by + boxSize}

	if checked {
		drawCard(hdc, boxRect, colAmber, colAmber, 5)
		drawLine(hdc, bx+sc(5), by+sc(10), bx+sc(9), by+sc(14), colBg, sc(2))
		drawLine(hdc, bx+sc(9), by+sc(14), bx+sc(15), by+sc(6), colBg, sc(2))
	} else {
		drawCard(hdc, boxRect, colBg, colMuted, 5)
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

	// Window Top-Right Controls (Minimize & Close)
	btnW := sc(36)
	btnH := sc(28)
	closeRect := RECT{w - sc(14) - btnW, sc(12), w - sc(14), sc(12) + btnH}
	minRect := RECT{closeRect.Left - sc(6) - btnW, sc(12), closeRect.Left - sc(6), sc(12) + btnH}

	app.Regions = append(app.Regions, HotRegion{ID: CtrlWinMin, Rect: minRect})
	app.Regions = append(app.Regions, HotRegion{ID: CtrlWinClose, Rect: closeRect})

	if app.HoverID == CtrlWinMin {
		drawCard(hdc, minRect, colPanelHov, colLine, 6)
	}
	drawText(hdc, "—", minRect, app.FontBold, colMuted, DT_CENTER|DT_VCENTER|DT_SINGLELINE)

	closeCol := colMuted
	if app.HoverID == CtrlWinClose {
		drawCard(hdc, closeRect, colRed, colRed, 6)
		closeCol = colText
	}
	drawText(hdc, "✕", closeRect, app.FontBold, closeCol, DT_CENTER|DT_VCENTER|DT_SINGLELINE)

	// Subtle outer window border
	drawLine(hdc, 0, 0, w, 0, colLine, 1)
	drawLine(hdc, 0, h-1, w, h-1, colLine, 1)
	drawLine(hdc, 0, 0, 0, h, colLine, 1)
	drawLine(hdc, w-1, 0, w-1, h, colLine, 1)
}

// ============================================================================
// 1. RSI-STYLE CLEAN INSTALLER UI
// ============================================================================
func paintInstallerMode(hdc uintptr, w, h int32) {
	// Left RSI Brand Sidebar
	sideW := sc(240)
	fillRect(hdc, RECT{0, 0, sideW, h}, colSidebar)
	drawLine(hdc, sideW, 0, sideW, h, colLine, 1)

	// Amber Brand Pill + Title
	fillRect(hdc, RECT{sc(28), sc(28), sc(36), sc(58)}, colAmber)
	drawText(hdc, "SC COCKPIT", RECT{sc(48), sc(25), sideW - sc(16), sc(46)}, app.FontTitle, colText, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, "SETUP & DEPLOYMENT", RECT{sc(48), sc(45), sideW - sc(16), sc(62)}, app.FontSmall, colAmber, DT_LEFT|DT_SINGLELINE)

	// Step Indicator in Sidebar
	steps := []string{"1.  Konfiguration", "2.  Installation", "3.  Bereit zum Start"}
	sy := sc(110)
	for i, st := range steps {
		active := (app.InstPage == InstPageWelcome && i == 0) ||
			(app.InstPage == InstPageProgress && i == 1) ||
			(app.InstPage == InstPageDone && i == 2)
		col := colMuted
		if active {
			col = colText
			fillRect(hdc, RECT{0, sy, sc(4), sy + sc(32)}, colAmber)
			fillRect(hdc, RECT{sc(4), sy, sideW, sy + sc(32)}, colPanel)
		}
		drawText(hdc, st, RECT{sc(28), sy, sideW - sc(16), sy + sc(32)}, app.FontBold, col, DT_LEFT|DT_VCENTER|DT_SINGLELINE)
		sy += sc(40)
	}

	// Sidebar Footer Info
	drawText(hdc, fmt.Sprintf("Version %s\nKein Admin erforderlich\nLokales WLAN-Dashboard", AppVersion),
		RECT{sc(28), h - sc(82), sideW - sc(16), h - sc(20)}, app.FontSmall, colDim, DT_LEFT|DT_WORDBREAK)

	// Right Content Area
	cx := sideW + sc(36)
	cw := w - cx - sc(36)

	switch app.InstPage {
	case InstPageWelcome:
		title := "SC Cockpit installieren"
		sub := "Richtet das Star Citizen Touch-Dashboard und das Control Center auf deinem PC ein."
		if app.ExistingInst {
			title = "SC Cockpit aktualisieren oder verwalten"
			sub = "Eine bestehende Installation wurde gefunden. Deine eigenen Dashboards bleiben beim Update erhalten."
		}
		drawText(hdc, title, RECT{cx, sc(32), cx + cw - sc(80), sc(62)}, app.FontHero, colText, DT_LEFT|DT_SINGLELINE)
		drawText(hdc, sub, RECT{cx, sc(64), cx + cw, sc(90)}, app.FontBody, colMuted, DT_LEFT|DT_SINGLELINE)

		// System Check Card
		sysCard := RECT{cx, sc(102), cx + cw, sc(174)}
		drawCard(hdc, sysCard, colPanel, colLine, 10)
		drawText(hdc, "ZIELORDNER", RECT{sysCard.Left + sc(18), sysCard.Top + sc(12), sysCard.Left + sc(310), sysCard.Top + sc(28)}, app.FontSmall, colDim, DT_LEFT|DT_SINGLELINE)
		drawText(hdc, app.AppDir, RECT{sysCard.Left + sc(18), sysCard.Top + sc(30), sysCard.Left + sc(330), sysCard.Bottom - sc(12)}, app.FontMono, colText, DT_LEFT|DT_SINGLELINE)

		drawLine(hdc, sysCard.Left+sc(345), sysCard.Top+sc(12), sysCard.Left+sc(345), sysCard.Bottom-sc(12), colLine, 1)
		drawText(hdc, "PYTHON 3 RUNTIME", RECT{sysCard.Left + sc(365), sysCard.Top + sc(12), sysCard.Right - sc(16), sysCard.Top + sc(28)}, app.FontSmall, colDim, DT_LEFT|DT_SINGLELINE)
		if app.PythonFound {
			drawText(hdc, "●  "+app.PythonVer+" bereit", RECT{sysCard.Left + sc(365), sysCard.Top + sc(30), sysCard.Right - sc(16), sysCard.Bottom - sc(12)}, app.FontBold, colGreen, DT_LEFT|DT_SINGLELINE)
		} else {
			drawText(hdc, "●  Wird automatisch eingerichtet", RECT{sysCard.Left + sc(365), sysCard.Top + sc(30), sysCard.Right - sc(16), sysCard.Bottom - sc(12)}, app.FontBold, colAmber, DT_LEFT|DT_SINGLELINE)
		}

		// Clean Option Checkboxes
		oy := sc(192)
		oh := sc(58)
		drawCheckboxRow(hdc, CtrlInstChkDesktop, RECT{cx, oy, cx + cw, oy + oh},
			"Desktop-Verknüpfung erstellen",
			"Legt 'SC Cockpit' mit Star-Citizen-Icon auf deinem Desktop an",
			app.OptDesktop)

		oy += oh + sc(12)
		drawCheckboxRow(hdc, CtrlInstChkStartMenu, RECT{cx, oy, cx + cw, oy + oh},
			"Startmenü-Einträge erstellen",
			"Erstellt SC Cockpit & SC Cockpit (Administrator) im Windows-Startmenü",
			app.OptStartMenu)

		if !app.PythonFound && app.WingetFound {
			oy += oh + sc(12)
			drawCheckboxRow(hdc, CtrlInstChkWingetPy, RECT{cx, oy, cx + cw, oy + oh},
				"Python 3.12 automatisch installieren (empfohlen)",
				"Installiert Python 3 für den lokalen Tastatur-Server über Windows winget",
				app.OptWingetPy)
		}

		// Bottom Action Bar
		btnPrimary := RECT{w - sc(36) - sc(210), h - sc(68), w - sc(36), h - sc(22)}
		btnLabel := "JETZT INSTALLIEREN"
		if app.ExistingInst {
			btnLabel = "AKTUALISIEREN"
			btnUninst := RECT{cx, h - sc(68), cx + sc(160), h - sc(22)}
			drawButton(hdc, CtrlInstSwitchUninstall, btnUninst, "Deinstallieren...", "danger")
		}
		drawButton(hdc, CtrlInstPrimary, btnPrimary, btnLabel, "amber")

	case InstPageUninstall:
		drawText(hdc, "SC Cockpit deinstallieren", RECT{cx, sc(32), cx + cw - sc(80), sc(62)}, app.FontHero, colText, DT_LEFT|DT_SINGLELINE)
		drawText(hdc, "Möchtest du SC Cockpit von deinem Computer entfernen?", RECT{cx, sc(64), cx + cw, sc(90)}, app.FontBody, colMuted, DT_LEFT|DT_SINGLELINE)

		warnCard := RECT{cx, sc(110), cx + cw, sc(185)}
		drawCard(hdc, warnCard, colRedBg, colRed, 10)
		drawText(hdc, "Das Programm sowie alle Desktop- und Startmenü-Verknüpfungen werden entfernt.\nDeine eigenen Dashboards bleiben standardmäßig im Benutzerordner gespeichert.",
			RECT{warnCard.Left + sc(18), warnCard.Top + sc(16), warnCard.Right - sc(18), warnCard.Bottom - sc(14)}, app.FontBody, colText, DT_LEFT|DT_WORDBREAK)

		drawCheckboxRow(hdc, CtrlChkPurgeData, RECT{cx, sc(205), cx + cw, sc(265)},
			"Auch alle eigenen Dashboards und Einstellungen unwiderruflich löschen",
			fmt.Sprintf("Löscht zusätzlich den Ordner %s", app.UserDir),
			app.OptPurgeData)

		btnBack := RECT{cx, h - sc(68), cx + sc(140), h - sc(22)}
		btnRemove := RECT{w - sc(36) - sc(210), h - sc(68), w - sc(36), h - sc(22)}
		drawButton(hdc, CtrlInstSwitchUninstall, btnBack, "Zurück", "secondary")
		drawButton(hdc, CtrlInstPrimary, btnRemove, "JETZT DEINSTALLIEREN", "danger")

	case InstPageProgress:
		app.mu.Lock()
		pct := app.ProgressPct
		status := app.StatusLine
		app.mu.Unlock()

		title := "Installation läuft..."
		if app.IsUninstall {
			title = "Deinstallation läuft..."
		}
		drawText(hdc, title, RECT{cx, sc(32), cx + cw, sc(62)}, app.FontHero, colText, DT_LEFT|DT_SINGLELINE)
		drawText(hdc, status, RECT{cx, sc(68), cx + cw, sc(94)}, app.FontBody, colMuted, DT_LEFT|DT_SINGLELINE)

		// Clean Smooth Progress Bar
		barBg := RECT{cx, sc(130), cx + cw, sc(152)}
		drawCard(hdc, barBg, colPanel, colLine, 10)
		fillW := ( (barBg.Right - barBg.Left) * int32(pct) ) / 100
		if fillW > sc(8) {
			drawCard(hdc, RECT{barBg.Left, barBg.Top, barBg.Left + fillW, barBg.Bottom}, colAmber, colAmber, 10)
		}
		drawText(hdc, fmt.Sprintf("%d %%", pct), RECT{cx, sc(162), cx + cw, sc(188)}, app.FontBold, colAmber, DT_RIGHT|DT_SINGLELINE)

	case InstPageDone:
		if app.InstallErr != "" {
			drawText(hdc, "Fehler bei der Ausführung", RECT{cx, sc(32), cx + cw, sc(62)}, app.FontHero, colRed, DT_LEFT|DT_SINGLELINE)
			errCard := RECT{cx, sc(80), cx + cw, sc(180)}
			drawCard(hdc, errCard, colRedBg, colRed, 10)
			drawText(hdc, app.InstallErr, RECT{errCard.Left + sc(18), errCard.Top + sc(16), errCard.Right - sc(18), errCard.Bottom - sc(16)}, app.FontBody, colText, DT_LEFT|DT_WORDBREAK)
		} else if app.IsUninstall {
			drawText(hdc, "Deinstallation abgeschlossen", RECT{cx, sc(32), cx + cw, sc(62)}, app.FontHero, colText, DT_LEFT|DT_SINGLELINE)
			doneCard := RECT{cx, sc(80), cx + cw, sc(165)}
			drawCard(hdc, doneCard, colPanel, colLine, 10)
			msg := "SC Cockpit wurde erfolgreich von deinem PC entfernt."
			if !app.OptPurgeData {
				msg += "\nDeine eigenen Dashboards im Ordner 'user' wurden behalten."
			}
			drawText(hdc, msg, RECT{doneCard.Left + sc(18), doneCard.Top + sc(18), doneCard.Right - sc(18), doneCard.Bottom - sc(18)}, app.FontBody, colMuted, DT_LEFT|DT_WORDBREAK)
		}
		btnClose := RECT{w - sc(36) - sc(180), h - sc(68), w - sc(36), h - sc(22)}
		drawButton(hdc, CtrlInstPrimary, btnClose, "SCHLIESSEN", "amber")
	}
}

// ============================================================================
// 2. RSI-STYLE GRAPHICAL CONTROL CENTER & LAUNCHER (POST-INSTALL)
// ============================================================================
func paintLauncherMode(hdc uintptr, w, h int32) {
	// Left RSI Navigation Sidebar
	sideW := sc(230)
	fillRect(hdc, RECT{0, 0, sideW, h}, colSidebar)
	drawLine(hdc, sideW, 0, sideW, h, colLine, 1)

	// Brand Header
	fillRect(hdc, RECT{sc(24), sc(24), sc(32), sc(54)}, colAmber)
	drawText(hdc, "SC COCKPIT", RECT{sc(44), sc(21), sideW - sc(12), sc(42)}, app.FontTitle, colText, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, "CONTROL CENTER", RECT{sc(44), sc(41), sideW - sc(12), sc(58)}, app.FontSmall, colCyan, DT_LEFT|DT_SINGLELINE)

	// Navigation Items
	navItems := []struct {
		id    int
		tab   int
		label string
	}{
		{CtrlNavServer, TabServer, "Cockpit & Server"},
		{CtrlNavDashboards, TabDashboards, fmt.Sprintf("Dashboards (%d)", len(app.Dashboards))},
		{CtrlNavSettings, TabSettings, "Einstellungen"},
	}

	ny := sc(92)
	for _, item := range navItems {
		r := RECT{sc(12), ny, sideW - sc(12), ny + sc(44)}
		app.Regions = append(app.Regions, HotRegion{ID: item.id, Rect: r})
		active := app.Tab == item.tab
		hov := app.HoverID == item.id

		if active {
			drawCard(hdc, r, colPanelAct, colLineHi, 8)
			fillRect(hdc, RECT{r.Left, r.Top + sc(10), r.Left + sc(4), r.Bottom - sc(10)}, colAmber)
		} else if hov {
			drawCard(hdc, r, colPanel, colLine, 8)
		}

		col := colMuted
		if active || hov {
			col = colText
		}
		drawText(hdc, item.label, RECT{r.Left + sc(18), r.Top, r.Right - sc(10), r.Bottom}, app.FontBold, col, DT_LEFT|DT_VCENTER|DT_SINGLELINE)
		ny += sc(52)
	}

	// Live Server Status Pill in Bottom-Left Sidebar
	statusCard := RECT{sc(16), h - sc(78), sideW - sc(16), h - sc(18)}
	if app.ServerRunning {
		drawCard(hdc, statusCard, colGreenBg, colGreen, 8)
		drawText(hdc, "●  SERVER ONLINE", RECT{statusCard.Left + sc(14), statusCard.Top + sc(10), statusCard.Right - sc(10), statusCard.Top + sc(28)}, app.FontBold, colGreen, DT_LEFT|DT_SINGLELINE)
		drawText(hdc, fmt.Sprintf("Port %d aktiv", app.Port), RECT{statusCard.Left + sc(14), statusCard.Top + sc(30), statusCard.Right - sc(10), statusCard.Bottom - sc(8)}, app.FontSmall, colText, DT_LEFT|DT_SINGLELINE)
	} else {
		drawCard(hdc, statusCard, colPanel, colLine, 8)
		drawText(hdc, "○  SERVER GESTOPPT", RECT{statusCard.Left + sc(14), statusCard.Top + sc(10), statusCard.Right - sc(10), statusCard.Top + sc(28)}, app.FontBold, colMuted, DT_LEFT|DT_SINGLELINE)
		drawText(hdc, "Klicke auf Starten", RECT{statusCard.Left + sc(14), statusCard.Top + sc(30), statusCard.Right - sc(10), statusCard.Bottom - sc(8)}, app.FontSmall, colDim, DT_LEFT|DT_SINGLELINE)
	}

	// Main Content Area
	cx := sideW + sc(32)
	cw := w - cx - sc(32)

	if app.ToastMsg != "" {
		toastRect := RECT{cx, h - sc(46), cx + cw, h - sc(14)}
		drawCard(hdc, toastRect, colAmberBg, colAmber, 6)
		drawText(hdc, app.ToastMsg, toastRect, app.FontBold, colAmber, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
	}

	switch app.Tab {
	case TabServer:
		paintTabServer(hdc, cx, cw, h)
	case TabDashboards:
		paintTabDashboards(hdc, cx, cw, h)
	case TabSettings:
		paintTabSettings(hdc, cx, cw, h)
	}
}

func paintTabServer(hdc uintptr, cx, cw, h int32) {
	drawText(hdc, "Cockpit Serversteuerung", RECT{cx, sc(26), cx + cw - sc(90), sc(56)}, app.FontHero, colText, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, "Starte den Server auf deinem PC und öffne die WLAN-Adresse auf deinem Tablet oder Smartphone.",
		RECT{cx, sc(56), cx + cw, sc(80)}, app.FontBody, colMuted, DT_LEFT|DT_SINGLELINE)

	// Hero Server Control Card (RSI Launcher Style)
	heroCard := RECT{cx, sc(92), cx + cw, sc(230)}
	drawCard(hdc, heroCard, colPanel, colLineHi, 12)

	url := fmt.Sprintf("http://%s:%d/", app.LanIP, app.Port)

	if app.ServerRunning {
		drawText(hdc, "STATUS: SERVER LÄUFT & EMPFÄNGT TASTEN", RECT{heroCard.Left + sc(24), heroCard.Top + sc(18), heroCard.Right - sc(220), heroCard.Top + sc(38)}, app.FontBold, colGreen, DT_LEFT|DT_SINGLELINE)
	} else {
		drawText(hdc, "STATUS: SERVER IST AUSGESCHALTET", RECT{heroCard.Left + sc(24), heroCard.Top + sc(18), heroCard.Right - sc(220), heroCard.Top + sc(38)}, app.FontBold, colAmber, DT_LEFT|DT_SINGLELINE)
	}

	drawText(hdc, "ADRESSE FÜR TABLET / HANDY IM WLAN:", RECT{heroCard.Left + sc(24), heroCard.Top + sc(46), heroCard.Right - sc(220), heroCard.Top + sc(64)}, app.FontSmall, colMuted, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, url, RECT{heroCard.Left + sc(24), heroCard.Top + sc(64), heroCard.Right - sc(220), heroCard.Top + sc(96)}, app.FontHero, colCyan, DT_LEFT|DT_SINGLELINE)

	// Quick Action Buttons inside Hero Card
	btnCopy := RECT{heroCard.Left + sc(24), heroCard.Bottom - sc(46), heroCard.Left + sc(174), heroCard.Bottom - sc(14)}
	btnOpen := RECT{btnCopy.Right + sc(10), heroCard.Bottom - sc(46), btnCopy.Right + sc(195), heroCard.Bottom - sc(14)}
	btnEdit := RECT{btnOpen.Right + sc(10), heroCard.Bottom - sc(46), btnOpen.Right + sc(195), heroCard.Bottom - sc(14)}

	drawButton(hdc, CtrlBtnCopyURL, btnCopy, "Adresse kopieren", "secondary")
	drawButton(hdc, CtrlBtnOpenBrowser, btnOpen, "Dashboard öffnen", "cyan")
	drawButton(hdc, CtrlBtnOpenEditor, btnEdit, "Im Browser anpassen", "secondary")

	// Big Power Toggle Button on Right Side of Hero Card
	powerBtn := RECT{heroCard.Right - sc(200), heroCard.Top + sc(22), heroCard.Right - sc(24), heroCard.Bottom - sc(58)}
	if app.ServerRunning {
		drawButton(hdc, CtrlBtnToggleServer, powerBtn, "SERVER STOPPEN", "danger")
	} else {
		drawButton(hdc, CtrlBtnToggleServer, powerBtn, "SERVER STARTEN", "amber")
	}

	// Active Dashboard & Admin Status Row
	midY := sc(246)
	cardW := (cw - sc(16)) / 2
	leftCard := RECT{cx, midY, cx + cardW, midY + sc(84)}
	rightCard := RECT{cx + cardW + sc(16), midY, cx + cw, midY + sc(84)}

	drawCard(hdc, leftCard, colPanel, colLine, 10)
	drawText(hdc, "AKTIVES DASHBOARD", RECT{leftCard.Left + sc(18), leftCard.Top + sc(12), leftCard.Right - sc(16), leftCard.Top + sc(28)}, app.FontSmall, colDim, DT_LEFT|DT_SINGLELINE)
	activeTitle := app.ActiveDash
	for _, d := range app.Dashboards {
		if d.ID == app.ActiveDash {
			activeTitle = fmt.Sprintf("%s  (%d Seiten, %d Tasten)", d.Title, d.PageCount, d.ButtonCount)
			break
		}
	}
	drawText(hdc, activeTitle, RECT{leftCard.Left + sc(18), leftCard.Top + sc(32), leftCard.Right - sc(16), leftCard.Bottom - sc(12)}, app.FontBold, colText, DT_LEFT|DT_SINGLELINE)

	drawCard(hdc, rightCard, colPanel, colLine, 10)
	drawText(hdc, "BERECHTIGUNGS-MODUS", RECT{rightCard.Left + sc(18), rightCard.Top + sc(12), rightCard.Right - sc(16), rightCard.Top + sc(28)}, app.FontSmall, colDim, DT_LEFT|DT_SINGLELINE)
	if app.IsAdmin {
		drawText(hdc, "●  Administrator-Modus aktiv", RECT{rightCard.Left + sc(18), rightCard.Top + sc(32), rightCard.Right - sc(16), rightCard.Bottom - sc(12)}, app.FontBold, colGreen, DT_LEFT|DT_SINGLELINE)
	} else {
		drawText(hdc, "Standard-Benutzer", RECT{rightCard.Left + sc(18), rightCard.Top + sc(32), rightCard.Right - sc(165), rightCard.Bottom - sc(12)}, app.FontBold, colText, DT_LEFT|DT_SINGLELINE)
		adminBtn := RECT{rightCard.Right - sc(155), rightCard.Top + sc(24), rightCard.Right - sc(14), rightCard.Bottom - sc(16)}
		drawButton(hdc, CtrlBtnRestartAdmin, adminBtn, "Als Admin starten", "secondary")
	}

	// Clean Live Activity Log
	logCard := RECT{cx, sc(346), cx + cw, h - sc(54)}
	drawCard(hdc, logCard, rgb(0x09, 0x0c, 0x10), colLine, 10)
	drawText(hdc, "LIVE TASTEN- & SERVER-PROTOKOLL", RECT{logCard.Left + sc(18), logCard.Top + sc(10), logCard.Right - sc(18), logCard.Top + sc(28)}, app.FontSmall, colDim, DT_LEFT|DT_SINGLELINE)

	app.mu.Lock()
	logs := append([]string(nil), app.ServerLogs...)
	app.mu.Unlock()

	ly := logCard.Top + sc(32)
	if len(logs) == 0 {
		drawText(hdc, "Noch keine Ereignisse. Sobald du auf dem Tablet eine Taste drückst, erscheint sie hier.",
			RECT{logCard.Left + sc(18), ly, logCard.Right - sc(18), ly + sc(24)}, app.FontMono, colDim, DT_LEFT|DT_SINGLELINE)
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

func paintTabDashboards(hdc uintptr, cx, cw, h int32) {
	drawText(hdc, "Dashboards verwalten", RECT{cx, sc(26), cx + cw - sc(260), sc(56)}, app.FontHero, colText, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, "Wähle dein aktives Dashboard, erstelle Kopien für eigene Schiffe oder bearbeite Tasten direkt im Browser.",
		RECT{cx, sc(56), cx + cw, sc(80)}, app.FontBody, colMuted, DT_LEFT|DT_SINGLELINE)

	// Top Action Buttons
	btnNew := RECT{cx, sc(92), cx + sc(185), sc(128)}
	btnFolder := RECT{btnNew.Right + sc(12), sc(92), btnNew.Right + sc(235), sc(128)}
	btnReload := RECT{btnFolder.Right + sc(12), sc(92), btnFolder.Right + sc(150), sc(128)}

	drawButton(hdc, CtrlBtnNewDash, btnNew, "+ Neues Dashboard", "amber")
	drawButton(hdc, CtrlBtnOpenDashFolder, btnFolder, "Dashboard-Ordner öffnen", "secondary")
	drawButton(hdc, CtrlBtnReloadDash, btnReload, "Aktualisieren", "secondary")

	// Dashboard List Cards
	y := sc(144)
	rowH := sc(68)
	for i, d := range app.Dashboards {
		if y+rowH > h-sc(50) {
			break
		}
		card := RECT{cx, y, cx + cw, y + rowH}
		isActive := d.ID == app.ActiveDash
		border := colLine
		fill := colPanel
		if isActive {
			border = colAmber
			fill = colPanelAct
		}
		drawCard(hdc, card, fill, border, 10)

		if isActive {
			fillRect(hdc, RECT{card.Left, card.Top + sc(12), card.Left + sc(4), card.Bottom - sc(12)}, colAmber)
		}

		// Title & Source Badge
		badge := "MITGELIEFERT"
		badgeCol := colCyan
		if d.Source == "local" {
			badge = "EIGENES DASHBOARD"
			badgeCol = colAmber
		}
		drawText(hdc, d.Title, RECT{card.Left + sc(20), card.Top + sc(12), card.Left + sc(260), card.Top + sc(36)}, app.FontTitle, colText, DT_LEFT|DT_SINGLELINE)
		meta := fmt.Sprintf("%s   •   ID: %s.json   •   %d Seiten   •   %d Buttons", badge, d.ID, d.PageCount, d.ButtonCount)
		drawText(hdc, meta, RECT{card.Left + sc(20), card.Top + sc(38), card.Right - sc(310), card.Bottom - sc(10)}, app.FontSmall, badgeCol, DT_LEFT|DT_SINGLELINE)

		// Action Buttons on the right of each dashboard card
		bx := card.Right - sc(16)

		if d.Source == "local" {
			btnDel := RECT{bx - sc(90), card.Top + sc(16), bx, card.Bottom - sc(16)}
			drawButton(hdc, CtrlDashDelBase+i, btnDel, "Löschen", "danger")
			bx = btnDel.Left - sc(8)
		}

		btnDup := RECT{bx - sc(100), card.Top + sc(16), bx, card.Bottom - sc(16)}
		drawButton(hdc, CtrlDashDupBase+i, btnDup, "Duplizieren", "secondary")
		bx = btnDup.Left - sc(8)

		btnEdit := RECT{bx - sc(100), card.Top + sc(16), bx, card.Bottom - sc(16)}
		drawButton(hdc, CtrlDashEditBase+i, btnEdit, "Bearbeiten", "cyan")
		bx = btnEdit.Left - sc(8)

		btnAct := RECT{bx - sc(105), card.Top + sc(16), bx, card.Bottom - sc(16)}
		if isActive {
			drawButton(hdc, CtrlDashActivateBase+i, btnAct, "● AKTIV", "green")
		} else {
			drawButton(hdc, CtrlDashActivateBase+i, btnAct, "Aktivieren", "secondary")
		}

		y += rowH + sc(10)
	}
}

func paintTabSettings(hdc uintptr, cx, cw, h int32) {
	drawText(hdc, "Einstellungen & Systemverwaltung", RECT{cx, sc(26), cx + cw - sc(90), sc(56)}, app.FontHero, colText, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, "Konfiguriere den Netzwerk-Port, öffne Systemordner oder deinstalliere SC Cockpit.",
		RECT{cx, sc(56), cx + cw, sc(80)}, app.FontBody, colMuted, DT_LEFT|DT_SINGLELINE)

	// 1. Port & Network Card
	portCard := RECT{cx, sc(94), cx + cw, sc(168)}
	drawCard(hdc, portCard, colPanel, colLine, 10)
	drawText(hdc, "Server-Port (Standard: 8765)", RECT{portCard.Left + sc(20), portCard.Top + sc(14), portCard.Left + sc(350), portCard.Top + sc(36)}, app.FontBold, colText, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, "Port für die Verbindung zwischen PC und Tablet/Handy im lokalen WLAN", RECT{portCard.Left + sc(20), portCard.Top + sc(38), portCard.Left + sc(450), portCard.Bottom - sc(12)}, app.FontSmall, colMuted, DT_LEFT|DT_SINGLELINE)

	btnMinus := RECT{portCard.Right - sc(180), portCard.Top + sc(18), portCard.Right - sc(140), portCard.Bottom - sc(18)}
	portBox := RECT{btnMinus.Right + sc(6), portCard.Top + sc(18), btnMinus.Right + sc(82), portCard.Bottom - sc(18)}
	btnPlus := RECT{portBox.Right + sc(6), portCard.Top + sc(18), portBox.Right + sc(46), portCard.Bottom - sc(18)}

	drawButton(hdc, CtrlBtnPortMinus, btnMinus, "—", "secondary")
	drawCard(hdc, portBox, colBg, colLineHi, 6)
	drawText(hdc, fmt.Sprintf("%d", app.Port), portBox, app.FontBold, colCyan, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
	drawButton(hdc, CtrlBtnPortPlus, btnPlus, "+", "secondary")

	// 2. Directories & Updates Card
	dirCard := RECT{cx, sc(182), cx + cw, sc(272)}
	drawCard(hdc, dirCard, colPanel, colLine, 10)
	drawText(hdc, "Speicherorte & Programmdateien", RECT{dirCard.Left + sc(20), dirCard.Top + sc(14), dirCard.Right - sc(20), dirCard.Top + sc(34)}, app.FontBold, colText, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, fmt.Sprintf("Programm: %s   |   Eigene Dashboards: %s", app.AppDir, app.UserDashDir),
		RECT{dirCard.Left + sc(20), dirCard.Top + sc(34), dirCard.Right - sc(20), dirCard.Top + sc(52)}, app.FontSmall, colMuted, DT_LEFT|DT_SINGLELINE)

	btnUserDir := RECT{dirCard.Left + sc(20), dirCard.Bottom - sc(36), dirCard.Left + sc(220), dirCard.Bottom - sc(8)}
	btnAppDir := RECT{btnUserDir.Right + sc(10), dirCard.Bottom - sc(36), btnUserDir.Right + sc(210), dirCard.Bottom - sc(8)}
	btnRepair := RECT{btnAppDir.Right + sc(10), dirCard.Bottom - sc(36), btnAppDir.Right + sc(210), dirCard.Bottom - sc(8)}

	drawButton(hdc, CtrlBtnOpenUserFolder, btnUserDir, "Benutzerordner öffnen", "secondary")
	drawButton(hdc, CtrlBtnOpenAppFolder, btnAppDir, "Programmordner öffnen", "secondary")
	drawButton(hdc, CtrlBtnUpdateRepair, btnRepair, "Verknüpfungen reparieren", "secondary")

	// 3. Uninstall Section inside Settings
	uninstCard := RECT{cx, sc(286), cx + cw, sc(435)}
	drawCard(hdc, uninstCard, colRedBg, colRed, 10)
	drawText(hdc, "SC Cockpit deinstallieren", RECT{uninstCard.Left + sc(20), uninstCard.Top + sc(14), uninstCard.Right - sc(20), uninstCard.Top + sc(36)}, app.FontBold, colRed, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, "Entfernt den SC Cockpit Server, das Control Center sowie alle Desktop- und Startmenü-Verknüpfungen von deinem PC.",
		RECT{uninstCard.Left + sc(20), uninstCard.Top + sc(36), uninstCard.Right - sc(20), uninstCard.Top + sc(56)}, app.FontSmall, colText, DT_LEFT|DT_SINGLELINE)

	drawCheckboxRow(hdc, CtrlChkPurgeData,
		RECT{uninstCard.Left + sc(20), uninstCard.Top + sc(60), uninstCard.Right - sc(230), uninstCard.Bottom - sc(14)},
		"Eigene Dashboards ebenfalls löschen",
		"Wenn deaktiviert, bleiben deine eigenen Dashboards im Ordner 'user' erhalten",
		app.OptPurgeData)

	if !app.ConfirmUninst {
		btnUninst := RECT{uninstCard.Right - sc(210), uninstCard.Top + sc(72), uninstCard.Right - sc(20), uninstCard.Bottom - sc(22)}
		drawButton(hdc, CtrlBtnShowUninstall, btnUninst, "Deinstallieren...", "danger")
	} else {
		btnConfirm := RECT{uninstCard.Right - sc(210), uninstCard.Top + sc(60), uninstCard.Right - sc(20), uninstCard.Top + sc(96)}
		btnCancel := RECT{uninstCard.Right - sc(210), uninstCard.Top + sc(102), uninstCard.Right - sc(20), uninstCard.Bottom - sc(12)}
		drawButton(hdc, CtrlBtnConfirmUninstall, btnConfirm, "Wirklich entfernen!", "danger")
		drawButton(hdc, CtrlBtnCancelUninstall, btnCancel, "Abbrechen", "secondary")
	}
}

func handleControlClick(id int) {
	app.ToastMsg = ""

	// Dashboard Row Actions
	if id >= CtrlDashActivateBase && id < CtrlDashActivateBase+100 {
		idx := id - CtrlDashActivateBase
		if idx < len(app.Dashboards) {
			setActiveDashboard(app.Dashboards[idx].ID)
			app.ToastMsg = fmt.Sprintf("Dashboard '%s' als aktiv gesetzt.", app.Dashboards[idx].Title)
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
				app.ToastMsg = fmt.Sprintf("Kopie '%s.json' unter eigene Dashboards angelegt.", newID)
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
				app.ToastMsg = fmt.Sprintf("Eigenes Dashboard '%s' gelöscht.", d.Title)
			}
		}
		pInvalidateRect.Call(app.Hwnd, 0, 0)
		return
	}

	switch id {
	case CtrlWinClose, CtrlInstSecondary:
		stopServer()
		pPostQuitMessage.Call(0)
	case CtrlWinMin:
		pShowWindow.Call(app.Hwnd, SW_MINIMIZE)

	// Navigation Tabs
	case CtrlNavServer:
		app.Tab = TabServer
		app.ConfirmUninst = false
	case CtrlNavDashboards:
		reloadDashboards()
		app.Tab = TabDashboards
		app.ConfirmUninst = false
	case CtrlNavSettings:
		app.Tab = TabSettings
		app.ConfirmUninst = false

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
		app.ToastMsg = "WLAN-Adresse in die Zwischenablage kopiert: " + url
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

	// Dashboards Tab Controls
	case CtrlBtnNewDash:
		newID, err := createNewDashboard()
		if err != nil {
			app.ToastMsg = "Fehler: " + err.Error()
		} else {
			app.ToastMsg = fmt.Sprintf("Neues Dashboard '%s.json' angelegt.", newID)
		}
	case CtrlBtnOpenDashFolder:
		_ = os.MkdirAll(app.UserDashDir, 0755)
		pShellExecuteW.Call(0, u16ptr("open"), u16ptr(app.UserDashDir), 0, 0, SW_SHOWNORMAL)
	case CtrlBtnReloadDash:
		reloadDashboards()
		app.ToastMsg = "Dashboard-Liste aktualisiert."

	// Settings Tab Controls
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
		app.ToastMsg = "Desktop- & Startmenü-Verknüpfungen wurden aktualisiert."
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

	// Installer Controls
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
		if pt.Y < sc(64) {
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
			// Immediately transition into the Graphical Control Center & start server!
			app.Mode = ModeLauncher
			app.Tab = TabServer
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

	app.Width = sc(920)
	app.Height = sc(580)

	app.CursorArrow, _, _ = pLoadCursorW.Call(0, IDC_ARROW)
	app.CursorHand, _, _ = pLoadCursorW.Call(0, IDC_HAND)

	app.FontHero = makeFont(22, FW_BOLD, "Segoe UI")
	app.FontTitle = makeFont(16, FW_BOLD, "Segoe UI")
	app.FontSub = makeFont(14, FW_SEMIBOLD, "Segoe UI")
	app.FontBody = makeFont(13, FW_NORMAL, "Segoe UI")
	app.FontBold = makeFont(13, FW_SEMIBOLD, "Segoe UI")
	app.FontMono = makeFont(12, FW_SEMIBOLD, "Consolas")
	app.FontSmall = makeFont(11, FW_NORMAL, "Segoe UI")

	hInst, _, _ := pGetModuleHandleW.Call(0)
	className := u16("SCCockpitControlCenter")
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

	hwnd, _, _ := pCreateWindowExW.Call(
		WS_EX_APPWINDOW,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(u16("SC Cockpit"))),
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
