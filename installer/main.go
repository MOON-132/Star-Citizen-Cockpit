package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	AppVersion = "0.2.0"
	AppName    = "SC Cockpit"
)

// Star Citizen Cockpit Palette (COLORREF = 0x00BBGGRR)
func rgb(r, g, b uint8) uint32 {
	return uint32(r) | (uint32(g) << 8) | (uint32(b) << 16)
}

var (
	colBg       = rgb(0x0b, 0x0e, 0x13) // #0b0e13 Deep Void
	colBg2      = rgb(0x12, 0x18, 0x20) // #121820 Panel Dark
	colCard     = rgb(0x16, 0x1d, 0x27) // #161d27 Card Fill
	colCardHov  = rgb(0x1c, 0x26, 0x34) // #1c2634 Hover Fill
	colLine     = rgb(0x24, 0x30, 0x41) // #243041 Structural Line
	colLineHi   = rgb(0x36, 0x48, 0x62) // #364862 Highlight Line
	colText     = rgb(0xe7, 0xed, 0xf5) // #e7edf5 Primary Text
	colMuted    = rgb(0x8b, 0x9b, 0xb0) // #8b9bb0 Telemetry Muted
	colDim      = rgb(0x52, 0x61, 0x75) // #526175 Dim Text
	colAmber    = rgb(0xe2, 0xa2, 0x3a) // #e2a23a SC Amber Gold
	colAmberDim = rgb(0x7a, 0x56, 0x1c) // #7a561c Dim Amber
	colAmberBg  = rgb(0x24, 0x1c, 0x10) // #241c10 Amber Banner Bg
	colCyan     = rgb(0x3e, 0xc8, 0xc4) // #3ec8c4 SC MFD Cyan
	colCyanDim  = rgb(0x1d, 0x5e, 0x5c) // #1d5e5c Dim Cyan
	colCyanBg   = rgb(0x0f, 0x22, 0x26) // #0f2226 Cyan Active Bg
	colRed      = rgb(0xe1, 0x5b, 0x4c) // #e15b4c Danger Red
	colRedBg    = rgb(0x2c, 0x14, 0x14) // #2c1414 Danger Bg
	colGreen    = rgb(0x6f, 0xbf, 0x73) // #6fbf73 Online Green
)

// Win32 Constants
const (
	WS_POPUP         = 0x80000000
	WS_VISIBLE       = 0x10000000
	WS_SYSMENU       = 0x00080000
	WS_MINIMIZEBOX   = 0x00020000
	WS_CLIPCHILDREN  = 0x02000000
	WS_EX_APPWINDOW  = 0x00040000
	WS_EX_LAYERED    = 0x00080000

	WM_DESTROY       = 0x0002
	WM_PAINT         = 0x000F
	WM_CLOSE         = 0x0010
	WM_ERASEBKGND    = 0x0014
	WM_SETCURSOR     = 0x0020
	WM_NCHITTEST     = 0x0084
	WM_KEYDOWN       = 0x0100
	WM_TIMER         = 0x0113
	WM_MOUSEMOVE     = 0x0200
	WM_LBUTTONDOWN   = 0x0201
	WM_LBUTTONUP     = 0x0202
	WM_MOUSELEAVE    = 0x02A3
	WM_APP_PROGRESS  = 0x8001
	WM_APP_DONE      = 0x8002

	HTCLIENT         = 1
	HTCAPTION        = 2

	IDC_ARROW        = 32512
	IDC_HAND         = 32649

	DT_LEFT          = 0x00000000
	DT_CENTER        = 0x00000001
	DT_RIGHT         = 0x00000002
	DT_VCENTER       = 0x00000004
	DT_WORDBREAK     = 0x00000010
	DT_SINGLELINE    = 0x00000020
	DT_NOPREFIX      = 0x00000800

	TRANSPARENT      = 1
	PS_SOLID         = 0
	NULL_BRUSH       = 5

	SW_HIDE          = 0
	SW_SHOWNORMAL    = 1
	SW_MINIMIZE      = 6

	CS_HREDRAW       = 0x0002
	CS_VREDRAW       = 0x0001
	CS_DROPSHADOW    = 0x00020000

	VK_ESCAPE        = 0x1B
	VK_RETURN        = 0x0D

	SRCCOPY          = 0x00CC0020
	 CLEARTYPE_QUALITY = 5
	FW_NORMAL        = 400
	FW_SEMIBOLD      = 600
	FW_BOLD          = 700
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

	pSetProcessDPIAware       = user32.NewProc("SetProcessDPIAware")
	pRegisterClassExW         = user32.NewProc("RegisterClassExW")
	pCreateWindowExW          = user32.NewProc("CreateWindowExW")
	pDefWindowProcW           = user32.NewProc("DefWindowProcW")
	pShowWindow               = user32.NewProc("ShowWindow")
	pUpdateWindow             = user32.NewProc("UpdateWindow")
	pGetMessageW              = user32.NewProc("GetMessageW")
	pTranslateMessage         = user32.NewProc("TranslateMessage")
	pDispatchMessageW         = user32.NewProc("DispatchMessageW")
	pPostQuitMessage          = user32.NewProc("PostQuitMessage")
	pPostMessageW             = user32.NewProc("PostMessageW")
	pBeginPaint               = user32.NewProc("BeginPaint")
	pEndPaint                 = user32.NewProc("EndPaint")
	pInvalidateRect           = user32.NewProc("InvalidateRect")
	pGetClientRect            = user32.NewProc("GetClientRect")
	pFillRect                 = user32.NewProc("FillRect")
	pDrawTextW                = user32.NewProc("DrawTextW")
	pLoadCursorW              = user32.NewProc("LoadCursorW")
	pLoadIconW                = user32.NewProc("LoadIconW")
	pSetCursor                = user32.NewProc("SetCursor")
	pGetSystemMetrics         = user32.NewProc("GetSystemMetrics")
	pSetCapture               = user32.NewProc("SetCapture")
	pReleaseCapture           = user32.NewProc("ReleaseCapture")
	pScreenToClient           = user32.NewProc("ScreenToClient")
	pSetTimer                 = user32.NewProc("SetTimer")
	pKillTimer                = user32.NewProc("KillTimer")

	pCreateCompatibleDC       = gdi32.NewProc("CreateCompatibleDC")
	pCreateCompatibleBitmap   = gdi32.NewProc("CreateCompatibleBitmap")
	pSelectObject             = gdi32.NewProc("SelectObject")
	pDeleteObject             = gdi32.NewProc("DeleteObject")
	pDeleteDC                 = gdi32.NewProc("DeleteDC")
	pBitBlt                   = gdi32.NewProc("BitBlt")
	pCreateSolidBrush         = gdi32.NewProc("CreateSolidBrush")
	pCreatePen                = gdi32.NewProc("CreatePen")
	pCreateFontW              = gdi32.NewProc("CreateFontW")
	pSetTextColor             = gdi32.NewProc("SetTextColor")
	pSetBkMode                = gdi32.NewProc("SetBkMode")
	pMoveToEx                 = gdi32.NewProc("MoveToEx")
	pLineTo                   = gdi32.NewProc("LineTo")
	pPolygon                  = gdi32.NewProc("Polygon")
	pGetStockObject           = gdi32.NewProc("GetStockObject")
	pGetDeviceCaps            = gdi32.NewProc("GetDeviceCaps")

	pGetModuleHandleW         = kernel32.NewProc("GetModuleHandleW")
	pShellExecuteW            = shell32.NewProc("ShellExecuteW")
)

// Application Pages
const (
	PageWelcome = iota
	PageInstalling
	PageDone
	PageUninstallConfirm
)

// Interactive Control IDs
const (
	CtrlNone = iota
	CtrlClose
	CtrlMinimize
	CtrlChkDesktop
	CtrlChkStartMenu
	CtrlChkWingetPy
	CtrlChkLaunchNow
	CtrlChkOpenDocs
	CtrlChkPurgeData
	CtrlBtnPrimary
	CtrlBtnSecondary
	CtrlBtnModeSwitch
)

type HotRegion struct {
	ID   int
	Rect RECT
}

type AppState struct {
	mu           sync.Mutex
	Hwnd         uintptr
	Scale        float64
	Width        int32
	Height       int32
	Page         int
	IsUninstall  bool
	ExistingInst bool
	PythonFound  bool
	PythonVer    string
	WingetFound  bool
	InstallDir   string
	UserDir      string

	// Options
	OptDesktop   bool
	OptStartMenu bool
	OptWingetPy  bool
	OptLaunchNow bool
	OptOpenDocs  bool
	OptPurgeData bool

	// Progress & Telemetry
	ProgressPct  int
	StatusLine   string
	Logs         []string
	InstallErr   string
	InstallOK    bool
	AnimTick     int

	// Mouse Interaction
	HoverID      int
	PressID      int
	Regions      []HotRegion

	// GDI Resources
	CursorArrow  uintptr
	CursorHand   uintptr
	FontHero     uintptr
	FontTitle    uintptr
	FontSub      uintptr
	FontBody     uintptr
	FontBold     uintptr
	FontMono     uintptr
	FontSmall    uintptr
}

var app = &AppState{
	OptDesktop:   true,
	OptStartMenu: true,
	OptLaunchNow: true,
}

func u16(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func sc(v int32) int32 {
	return int32(float64(v)*app.Scale + 0.5)
}

func ptInRect(x, y int32, r RECT) bool {
	return x >= r.Left && x < r.Right && y >= r.Top && y < r.Bottom
}

func (a *AppState) addLog(msg string) {
	a.mu.Lock()
	ts := time.Now().Format("15:04:05")
	a.Logs = append(a.Logs, fmt.Sprintf("[%s]  %s", ts, msg))
	if len(a.Logs) > 9 {
		a.Logs = a.Logs[len(a.Logs)-9:]
	}
	a.StatusLine = msg
	a.mu.Unlock()
	if a.Hwnd != 0 {
		pPostMessageW.Call(a.Hwnd, WM_APP_PROGRESS, 0, 0)
	}
}

func (a *AppState) setProgress(pct int, msg string) {
	a.mu.Lock()
	a.ProgressPct = pct
	if msg != "" {
		ts := time.Now().Format("15:04:05")
		a.Logs = append(a.Logs, fmt.Sprintf("[%s]  %s", ts, msg))
		if len(a.Logs) > 9 {
			a.Logs = a.Logs[len(a.Logs)-9:]
		}
		a.StatusLine = msg
	}
	a.mu.Unlock()
	if a.Hwnd != 0 {
		pPostMessageW.Call(a.Hwnd, WM_APP_PROGRESS, 0, 0)
	}
}

// Chamfered polygon helper (draws an octagonal/chamfered sci-fi box)
func drawChamferBox(hdc uintptr, r RECT, cutTL, cutTR, cutBR, cutBL int32, fillCol, borderCol uint32, borderWidth int32) {
	pts := []POINT{
		{r.Left + cutTL, r.Top},
		{r.Right - cutTR, r.Top},
		{r.Right, r.Top + cutTR},
		{r.Right, r.Bottom - cutBR},
		{r.Right - cutBR, r.Bottom},
		{r.Left + cutBL, r.Bottom},
		{r.Left, r.Bottom - cutBL},
		{r.Left, r.Top + cutTL},
	}
	brush, _, _ := pCreateSolidBrush.Call(uintptr(fillCol))
	pen, _, _ := pCreatePen.Call(PS_SOLID, uintptr(borderWidth), uintptr(borderCol))
	oldB, _, _ := pSelectObject.Call(hdc, brush)
	oldP, _, _ := pSelectObject.Call(hdc, pen)

	pPolygon.Call(hdc, uintptr(unsafe.Pointer(&pts[0])), uintptr(len(pts)))

	pSelectObject.Call(hdc, oldB)
	pSelectObject.Call(hdc, oldP)
	pDeleteObject.Call(brush)
	pDeleteObject.Call(pen)
}

func fillSolidRect(hdc uintptr, r RECT, col uint32) {
	brush, _, _ := pCreateSolidBrush.Call(uintptr(col))
	pFillRect.Call(hdc, uintptr(unsafe.Pointer(&r)), brush)
	pDeleteObject.Call(brush)
}

func drawLine(hdc uintptr, x1, y1, x2, y2 int32, col uint32, width int32) {
	pen, _, _ := pCreatePen.Call(PS_SOLID, uintptr(width), uintptr(col))
	oldP, _, _ := pSelectObject.Call(hdc, pen)
	pMoveToEx.Call(hdc, uintptr(x1), uintptr(y1), 0)
	pLineTo.Call(hdc, uintptr(x2), uintptr(y2))
	pSelectObject.Call(hdc, oldP)
	pDeleteObject.Call(pen)
}

func drawCornerBrackets(hdc uintptr, r RECT, length, thick int32, col uint32) {
	pen, _, _ := pCreatePen.Call(PS_SOLID, uintptr(thick), uintptr(col))
	oldP, _, _ := pSelectObject.Call(hdc, pen)

	// Top-Left
	pMoveToEx.Call(hdc, uintptr(r.Left), uintptr(r.Top+length), 0)
	pLineTo.Call(hdc, uintptr(r.Left), uintptr(r.Top))
	pLineTo.Call(hdc, uintptr(r.Left+length), uintptr(r.Top))

	// Top-Right
	pMoveToEx.Call(hdc, uintptr(r.Right-length), uintptr(r.Top), 0)
	pLineTo.Call(hdc, uintptr(r.Right), uintptr(r.Top))
	pLineTo.Call(hdc, uintptr(r.Right), uintptr(r.Top+length))

	// Bottom-Right
	pMoveToEx.Call(hdc, uintptr(r.Right), uintptr(r.Bottom-length), 0)
	pLineTo.Call(hdc, uintptr(r.Right), uintptr(r.Bottom))
	pLineTo.Call(hdc, uintptr(r.Right-length), uintptr(r.Bottom))

	// Bottom-Left
	pMoveToEx.Call(hdc, uintptr(r.Left+length), uintptr(r.Bottom), 0)
	pLineTo.Call(hdc, uintptr(r.Left), uintptr(r.Bottom))
	pLineTo.Call(hdc, uintptr(r.Left), uintptr(r.Bottom-length))

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

func drawCheckboxCard(hdc uintptr, id int, r RECT, title, subtitle string, checked bool, accentCol uint32) {
	app.Regions = append(app.Regions, HotRegion{ID: id, Rect: r})
	hov := app.HoverID == id
	fill := colCard
	border := colLine
	if hov {
		fill = colCardHov
		border = colLineHi
	}
	if checked {
		border = accentCol
	}

	cut := sc(8)
	drawChamferBox(hdc, r, 0, cut, 0, cut, fill, border, sc(1))

	// Left accent bar
	barCol := colLine
	if checked {
		barCol = accentCol
	}
	fillSolidRect(hdc, RECT{r.Left, r.Top + sc(4), r.Left + sc(3), r.Bottom - sc(4)}, barCol)

	// Checkbox indicator square
	boxSize := sc(18)
	bx := r.Left + sc(16)
	by := r.Top + (r.Bottom-r.Top-boxSize)/2
	boxRect := RECT{bx, by, bx + boxSize, by + boxSize}
	boxFill := colBg
	boxBorder := colMuted
	if checked {
		boxFill = accentCol
		boxBorder = accentCol
	} else if hov {
		boxBorder = colText
	}
	drawChamferBox(hdc, boxRect, sc(3), sc(3), sc(3), sc(3), boxFill, boxBorder, sc(1))

	if checked {
		// Draw crisp checkmark inside box
		drawLine(hdc, bx+sc(4), by+sc(9), bx+sc(8), by+sc(13), colBg, sc(2))
		drawLine(hdc, bx+sc(8), by+sc(13), bx+sc(14), by+sc(5), colBg, sc(2))
	}

	// Title and subtitle
	tx := bx + boxSize + sc(14)
	if subtitle != "" {
		drawText(hdc, title, RECT{tx, r.Top + sc(7), r.Right - sc(12), r.Top + sc(26)}, app.FontBold, colText, DT_LEFT|DT_SINGLELINE|DT_VCENTER)
		drawText(hdc, subtitle, RECT{tx, r.Top + sc(25), r.Right - sc(12), r.Bottom - sc(5)}, app.FontSmall, colMuted, DT_LEFT|DT_SINGLELINE|DT_VCENTER)
	} else {
		drawText(hdc, title, RECT{tx, r.Top, r.Right - sc(12), r.Bottom}, app.FontBold, colText, DT_LEFT|DT_SINGLELINE|DT_VCENTER)
	}
}

func drawMFDButton(hdc uintptr, id int, r RECT, label string, primary bool, danger bool) {
	app.Regions = append(app.Regions, HotRegion{ID: id, Rect: r})
	hov := app.HoverID == id
	press := app.PressID == id

	var fill, border, textCol, accent uint32
	if danger {
		fill = colRedBg
		border = colRed
		textCol = colText
		accent = colRed
		if hov {
			fill = rgb(0x42, 0x1a, 0x18)
		}
	} else if primary {
		fill = colAmberBg
		border = colAmber
		textCol = colAmber
		accent = colAmber
		if hov {
			fill = rgb(0x36, 0x28, 0x14)
			textCol = rgb(0xff, 0xc8, 0x64)
		}
	} else {
		fill = colCard
		border = colLine
		textCol = colMuted
		accent = colCyan
		if hov {
			fill = colCardHov
			border = colCyan
			textCol = colText
		}
	}

	if press {
		r.Top += sc(1)
		r.Bottom += sc(1)
	}

	cut := sc(10)
	drawChamferBox(hdc, r, cut, 0, cut, 0, fill, border, sc(1))

	// Side glow bar
	fillSolidRect(hdc, RECT{r.Left, r.Top + cut, r.Left + sc(3), r.Bottom - sc(2)}, accent)
	// Top right corner tick
	drawLine(hdc, r.Right-sc(12), r.Top+sc(3), r.Right-sc(3), r.Top+sc(3), accent, sc(2))
	drawLine(hdc, r.Right-sc(3), r.Top+sc(3), r.Right-sc(3), r.Top+sc(12), accent, sc(2))

	drawText(hdc, label, r, app.FontBold, textCol, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
}

func paintHUD(hdc uintptr, w, h int32) {
	app.Regions = app.Regions[:0]

	// 1. Deep Space Background
	fillSolidRect(hdc, RECT{0, 0, w, h}, colBg)

	// Subtle tactical background grid lines
	gridStep := sc(36)
	gridCol := rgb(0x0f, 0x14, 0x1c)
	for x := gridStep; x < w; x += gridStep {
		drawLine(hdc, x, 0, x, h, gridCol, 1)
	}
	for y := gridStep; y < h; y += gridStep {
		drawLine(hdc, 0, y, w, y, gridCol, 1)
	}

	// Top header panel background
	headerH := sc(72)
	fillSolidRect(hdc, RECT{0, 0, w, headerH}, colBg2)
	drawLine(hdc, 0, headerH, w, headerH, colLine, sc(1))
	// Glowing amber segment on header line
	drawLine(hdc, sc(24), headerH, sc(220), headerH, colAmber, sc(2))
	drawLine(hdc, sc(224), headerH, sc(260), headerH, colCyan, sc(2))

	// Amber Brand Mark (matches index.html .mark)
	markRect := RECT{sc(24), sc(18), sc(36), sc(54)}
	fillSolidRect(hdc, markRect, colAmber)
	fillSolidRect(hdc, RECT{sc(24), sc(38), sc(36), sc(54)}, colAmberDim)

	// Header Titles
	drawText(hdc, "SC COCKPIT // DEPLOYMENT SYSTEM", RECT{sc(48), sc(15), w - sc(120), sc(38)}, app.FontTitle, colText, DT_LEFT|DT_SINGLELINE|DT_VCENTER)
	subHeader := fmt.Sprintf("AEGIS-MFD TERMINAL  |  BUILD %s  |  UEE STANDARD TOUCH INTERFACE", AppVersion)
	drawText(hdc, subHeader, RECT{sc(48), sc(38), w - sc(120), sc(56)}, app.FontMono, colCyan, DT_LEFT|DT_SINGLELINE|DT_VCENTER)

	// Window Control Buttons (Minimize & Close)
	btnW := sc(38)
	btnH := sc(28)
	closeRect := RECT{w - sc(16) - btnW, sc(14), w - sc(16), sc(14) + btnH}
	minRect := RECT{closeRect.Left - sc(6) - btnW, sc(14), closeRect.Left - sc(6), sc(14) + btnH}

	app.Regions = append(app.Regions, HotRegion{ID: CtrlMinimize, Rect: minRect})
	app.Regions = append(app.Regions, HotRegion{ID: CtrlClose, Rect: closeRect})

	if app.HoverID == CtrlMinimize {
		fillSolidRect(hdc, minRect, colCardHov)
	}
	drawText(hdc, "—", minRect, app.FontBold, colMuted, DT_CENTER|DT_VCENTER|DT_SINGLELINE)

	closeCol := colMuted
	if app.HoverID == CtrlClose {
		fillSolidRect(hdc, closeRect, colRed)
		closeCol = colText
	}
	drawText(hdc, "✕", closeRect, app.FontBold, closeCol, DT_CENTER|DT_VCENTER|DT_SINGLELINE)

	// Telemetry Status Bar below header
	teleY := headerH + sc(12)
	teleH := sc(62)
	teleRect := RECT{sc(24), teleY, w - sc(24), teleY + teleH}
	drawChamferBox(hdc, teleRect, sc(8), sc(8), sc(8), sc(8), colBg2, colLine, sc(1))
	drawCornerBrackets(hdc, teleRect, sc(10), sc(2), colAmber)

	// Telemetry 3 Columns: Target Path, Python Runtime, System Mode
	col1W := (teleRect.Right - teleRect.Left) / 2
	col2X := teleRect.Left + col1W + sc(10)
	drawLine(hdc, teleRect.Left+col1W, teleRect.Top+sc(10), teleRect.Left+col1W, teleRect.Bottom-sc(10), colLine, 1)

	drawText(hdc, "ZIELVERZEICHNIS // TARGET PATH", RECT{teleRect.Left + sc(16), teleRect.Top + sc(8), teleRect.Left + col1W - sc(10), teleRect.Top + sc(24)}, app.FontSmall, colMuted, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, app.InstallDir, RECT{teleRect.Left + sc(16), teleRect.Top + sc(26), teleRect.Left + col1W - sc(10), teleRect.Bottom - sc(8)}, app.FontMono, colText, DT_LEFT|DT_SINGLELINE|DT_VCENTER)

	drawText(hdc, "SYSTEM-TELEMETRIE // RUNTIME CHECK", RECT{col2X, teleRect.Top + sc(8), teleRect.Right - sc(16), teleRect.Top + sc(24)}, app.FontSmall, colMuted, DT_LEFT|DT_SINGLELINE)

	pyStatus := "PYTHON 3: NICHT GEFUNDEN (WINGET VERFÜGBAR)"
	pyCol := colAmber
	if app.PythonFound {
		pyStatus = fmt.Sprintf("PYTHON RUNTIME: %s [ONLINE]", app.PythonVer)
		pyCol = colGreen
	} else if !app.WingetFound {
		pyStatus = "PYTHON 3: FEHLT (BITTE VON PYTHON.ORG INSTALLIEREN)"
		pyCol = colRed
	}
	// Status dot
	dotY := teleRect.Top + sc(34)
	fillSolidRect(hdc, RECT{col2X, dotY, col2X + sc(8), dotY + sc(8)}, pyCol)
	drawText(hdc, pyStatus, RECT{col2X + sc(14), teleRect.Top + sc(26), teleRect.Right - sc(16), teleRect.Bottom - sc(8)}, app.FontMono, pyCol, DT_LEFT|DT_SINGLELINE|DT_VCENTER)

	// Main Content Area
	contentTop := teleRect.Bottom + sc(14)
	footerTop := h - sc(74)

	switch app.Page {
	case PageWelcome:
		paintPageWelcome(hdc, w, contentTop, footerTop)
	case PageUninstallConfirm:
		paintPageUninstall(hdc, w, contentTop, footerTop)
	case PageInstalling:
		paintPageInstalling(hdc, w, contentTop, footerTop)
	case PageDone:
		paintPageDone(hdc, w, contentTop, footerTop)
	}

	// Footer Bar
	fillSolidRect(hdc, RECT{0, footerTop, w, h}, colBg2)
	drawLine(hdc, 0, footerTop, w, footerTop, colLine, 1)

	// Footer left metadata
	modeTag := "MODUS: INSTALLATION"
	if app.ExistingInst && app.Page == PageWelcome {
		modeTag = "MODUS: UPDATE / REPARATUR"
	} else if app.Page == PageUninstallConfirm || app.IsUninstall {
		modeTag = "MODUS: DEINSTALLATION"
	}
	drawText(hdc, modeTag, RECT{sc(24), footerTop + sc(14), sc(300), footerTop + sc(32)}, app.FontMono, colAmber, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, "KEIN ADMIN NÖTIG  •  LOKALES WLAN-PANEL  •  ZERO CLOUD", RECT{sc(24), footerTop + sc(34), sc(420), footerTop + sc(54)}, app.FontSmall, colMuted, DT_LEFT|DT_SINGLELINE)

	// Footer Buttons
	btnPrimaryRect := RECT{w - sc(24) - sc(215), footerTop + sc(15), w - sc(24), footerTop + sc(59)}
	btnSecondaryRect := RECT{btnPrimaryRect.Left - sc(12) - sc(145), footerTop + sc(15), btnPrimaryRect.Left - sc(12), footerTop + sc(59)}

	switch app.Page {
	case PageWelcome:
		if app.ExistingInst {
			drawMFDButton(hdc, CtrlBtnModeSwitch, btnSecondaryRect, "DEINSTALLIEREN", false, true)
			drawMFDButton(hdc, CtrlBtnPrimary, btnPrimaryRect, "AKTUALISIEREN  ►", true, false)
		} else {
			drawMFDButton(hdc, CtrlBtnSecondary, btnSecondaryRect, "ABBRECHEN", false, false)
			drawMFDButton(hdc, CtrlBtnPrimary, btnPrimaryRect, "INSTALLIEREN  ►", true, false)
		}
	case PageUninstallConfirm:
		drawMFDButton(hdc, CtrlBtnModeSwitch, btnSecondaryRect, "◄  ZURÜCK", false, false)
		drawMFDButton(hdc, CtrlBtnPrimary, btnPrimaryRect, "ENTFERNEN  ✕", false, true)
	case PageInstalling:
		drawText(hdc, "BITTE WARTEN — SYSTEMOPERATION LÄUFT...", RECT{w - sc(380), footerTop, w - sc(24), h}, app.FontMono, colCyan, DT_RIGHT|DT_VCENTER|DT_SINGLELINE)
	case PageDone:
		drawMFDButton(hdc, CtrlBtnPrimary, btnPrimaryRect, "FERTIGSTELLEN  ►", true, false)
	}

	// Outer Window tactical border
	drawCornerBrackets(hdc, RECT{1, 1, w - 2, h - 2}, sc(18), sc(2), colAmber)
	drawLine(hdc, sc(20), 0, w-sc(20), 0, colLine, 1)
	drawLine(hdc, sc(20), h-1, w-sc(20), h-1, colLine, 1)
	drawLine(hdc, 0, sc(20), 0, h-sc(20), colLine, 1)
	drawLine(hdc, w-1, sc(20), w-1, h-sc(20), colLine, 1)
}

func paintPageWelcome(hdc uintptr, w, top, bottom int32) {
	padX := sc(24)

	// Section Header
	secTitle := "KONFIGURATION & BEREITSTELLUNG // DEPLOYMENT OPTIONS"
	if app.ExistingInst {
		secTitle = "BESTEHENDE INSTALLATION ERKANNT // UPDATE ODER REPARATUR"
	}
	drawText(hdc, secTitle, RECT{padX, top, w - padX, top + sc(20)}, app.FontMono, colAmber, DT_LEFT|DT_SINGLELINE)

	desc := "Installiert den SC Cockpit Server, das Touch-Interface sowie die Standard- und Mining-Dashboards in dein Benutzerprofil. Eigene Dashboards im Ordner 'user' bleiben bei Updates unangetastet."
	drawText(hdc, desc, RECT{padX, top + sc(22), w - padX, top + sc(58)}, app.FontBody, colMuted, DT_LEFT|DT_WORDBREAK)

	cardY := top + sc(64)
	cardH := sc(52)
	gap := sc(10)

	drawCheckboxCard(hdc, CtrlChkDesktop,
		RECT{padX, cardY, w - padX, cardY + cardH},
		"Desktop-Verknüpfung anlegen",
		"Erstellt 'SC Cockpit' direkt auf dem Desktop für schnellen Spielstart",
		app.OptDesktop, colAmber)

	cardY += cardH + gap
	drawCheckboxCard(hdc, CtrlChkStartMenu,
		RECT{padX, cardY, w - padX, cardY + cardH},
		"Startmenü-Ordner & Administrator-Starter erstellen",
		"Inklusive 'SC Cockpit (Administrator)' falls Star Citizen mit Admin-Rechten läuft",
		app.OptStartMenu, colAmber)

	cardY += cardH + gap
	if !app.PythonFound && app.WingetFound {
		drawCheckboxCard(hdc, CtrlChkWingetPy,
			RECT{padX, cardY, w - padX, cardY + cardH},
			"Python 3.12 automatisch via Windows Package Manager (winget) installieren",
			"Erhältlich ohne Admin-Rechte (User-Scope) — wird für den lokalen Cockpit-Server benötigt",
			app.OptWingetPy, colCyan)
	} else {
		// Info panel about included modules
		infoRect := RECT{padX, cardY, w - padX, cardY + cardH}
		drawChamferBox(hdc, infoRect, 0, sc(8), 0, sc(8), colBg2, colLine, 1)
		fillSolidRect(hdc, RECT{infoRect.Left, infoRect.Top + sc(4), infoRect.Left + sc(3), infoRect.Bottom - sc(4)}, colCyan)
		drawText(hdc, "INKLUSIVE MODULE:  STANDARD (FLUG / ENERGIE / QUANTUM / KAMPF)  •  MINING  •  LIVE-EDITOR",
			RECT{infoRect.Left + sc(16), infoRect.Top + sc(8), infoRect.Right - sc(16), infoRect.Top + sc(26)}, app.FontMono, colCyan, DT_LEFT|DT_SINGLELINE)
		drawText(hdc, fmt.Sprintf("Eigene Dashboards & Konfiguration:  %s", app.UserDir),
			RECT{infoRect.Left + sc(16), infoRect.Top + sc(26), infoRect.Right - sc(16), infoRect.Bottom - sc(6)}, app.FontSmall, colMuted, DT_LEFT|DT_SINGLELINE)
	}
}

func paintPageUninstall(hdc uintptr, w, top, bottom int32) {
	padX := sc(24)

	// Warning Banner
	warnRect := RECT{padX, top + sc(4), w - padX, top + sc(82)}
	drawChamferBox(hdc, warnRect, sc(10), sc(10), sc(10), sc(10), colRedBg, colRed, sc(1))
	fillSolidRect(hdc, RECT{warnRect.Left, warnRect.Top + sc(10), warnRect.Left + sc(4), warnRect.Bottom - sc(10)}, colRed)

	drawText(hdc, "WARNUNG // SYSTEM-DEINSTALLATION", RECT{warnRect.Left + sc(18), warnRect.Top + sc(12), warnRect.Right - sc(16), warnRect.Top + sc(32)}, app.FontBold, colRed, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, "Der Programmordner (%LOCALAPPDATA%\\SC Cockpit\\app) sowie alle Desktop- und Startmenü-Verknüpfungen werden von diesem System entfernt.",
		RECT{warnRect.Left + sc(18), warnRect.Top + sc(34), warnRect.Right - sc(16), warnRect.Bottom - sc(10)}, app.FontBody, colText, DT_LEFT|DT_WORDBREAK)

	cardY := warnRect.Bottom + sc(18)
	cardH := sc(58)
	drawCheckboxCard(hdc, CtrlChkPurgeData,
		RECT{padX, cardY, w - padX, cardY + cardH},
		"Eigene Dashboards und Einstellungen ebenfalls unwiderruflich löschen (-Purge)",
		fmt.Sprintf("Löscht auch %s (standardmäßig bleiben deine eigenen Dashboards erhalten)", app.UserDir),
		app.OptPurgeData, colRed)
}

func paintPageInstalling(hdc uintptr, w, top, bottom int32) {
	padX := sc(24)
	app.mu.Lock()
	pct := app.ProgressPct
	status := app.StatusLine
	logs := append([]string(nil), app.Logs...)
	tick := app.AnimTick
	app.mu.Unlock()

	drawText(hdc, "QUANTUM DEPLOYMENT SEQUENCE // DATEIÜBERTRAGUNG", RECT{padX, top, w - padX, top + sc(20)}, app.FontMono, colCyan, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, fmt.Sprintf("%d %%", pct), RECT{padX, top - sc(2), w - padX, top + sc(22)}, app.FontTitle, colAmber, DT_RIGHT|DT_SINGLELINE)

	// Segmented Sci-Fi Progress Bar
	barRect := RECT{padX, top + sc(26), w - padX, top + sc(52)}
	drawChamferBox(hdc, barRect, sc(6), sc(6), sc(6), sc(6), colBg2, colLineHi, 1)

	totalSegs := 32
	innerW := barRect.Right - barRect.Left - sc(12)
	segGap := sc(3)
	segW := (innerW - int32(totalSegs-1)*segGap) / int32(totalSegs)
	activeSegs := (pct * totalSegs) / 100

	for i := 0; i < totalSegs; i++ {
		sx := barRect.Left + sc(6) + int32(i)*(segW+segGap)
		sRect := RECT{sx, barRect.Top + sc(5), sx + segW, barRect.Bottom - sc(5)}
		if i < activeSegs {
			col := colAmber
			if i == activeSegs-1 && (tick%2 == 0) {
				col = colCyan
			}
			fillSolidRect(hdc, sRect, col)
		} else {
			fillSolidRect(hdc, sRect, rgb(0x18, 0x20, 0x2b))
		}
	}

	drawText(hdc, status, RECT{padX, barRect.Bottom + sc(8), w - padX, barRect.Bottom + sc(26)}, app.FontBody, colText, DT_LEFT|DT_SINGLELINE)

	// Telemetry Console Log Window
	logRect := RECT{padX, barRect.Bottom + sc(32), w - padX, bottom - sc(14)}
	drawChamferBox(hdc, logRect, sc(8), sc(8), sc(8), sc(8), rgb(0x08, 0x0b, 0x0f), colLine, 1)
	drawCornerBrackets(hdc, logRect, sc(8), sc(1), colCyan)

	ly := logRect.Top + sc(10)
	lineH := sc(18)
	for i, line := range logs {
		c := colMuted
		if i == len(logs)-1 {
			c = colCyan
		}
		drawText(hdc, line, RECT{logRect.Left + sc(14), ly, logRect.Right - sc(14), ly + lineH}, app.FontMono, c, DT_LEFT|DT_SINGLELINE)
		ly += lineH
	}
}

func paintPageDone(hdc uintptr, w, top, bottom int32) {
	padX := sc(24)

	bannerRect := RECT{padX, top + sc(4), w - padX, top + sc(86)}
	if app.InstallErr != "" {
		drawChamferBox(hdc, bannerRect, sc(10), sc(10), sc(10), sc(10), colRedBg, colRed, 1)
		fillSolidRect(hdc, RECT{bannerRect.Left, bannerRect.Top + sc(8), bannerRect.Left + sc(4), bannerRect.Bottom - sc(8)}, colRed)
		drawText(hdc, "FEHLER BEI DER AUSFÜHRUNG // OPERATION ABORTED", RECT{bannerRect.Left + sc(18), bannerRect.Top + sc(12), bannerRect.Right - sc(16), bannerRect.Top + sc(32)}, app.FontBold, colRed, DT_LEFT|DT_SINGLELINE)
		drawText(hdc, app.InstallErr, RECT{bannerRect.Left + sc(18), bannerRect.Top + sc(36), bannerRect.Right - sc(16), bannerRect.Bottom - sc(10)}, app.FontBody, colText, DT_LEFT|DT_WORDBREAK)
		return
	}

	if app.IsUninstall {
		drawChamferBox(hdc, bannerRect, sc(10), sc(10), sc(10), sc(10), colCyanBg, colCyan, 1)
		fillSolidRect(hdc, RECT{bannerRect.Left, bannerRect.Top + sc(8), bannerRect.Left + sc(4), bannerRect.Bottom - sc(8)}, colCyan)
		drawText(hdc, "DEINSTALLATION ABGESCHLOSSEN // SYSTEM CLEAN", RECT{bannerRect.Left + sc(18), bannerRect.Top + sc(14), bannerRect.Right - sc(16), bannerRect.Top + sc(36)}, app.FontBold, colCyan, DT_LEFT|DT_SINGLELINE)
		msg := "SC Cockpit wurde erfolgreich entfernt. Deine eigenen Dashboards unter %LOCALAPPDATA%\\SC Cockpit\\user wurden behalten."
		if app.OptPurgeData {
			msg = "SC Cockpit sowie alle eigenen Dashboards und Benutzerdaten wurden vollständig entfernt."
		}
		drawText(hdc, msg, RECT{bannerRect.Left + sc(18), bannerRect.Top + sc(38), bannerRect.Right - sc(16), bannerRect.Bottom - sc(10)}, app.FontBody, colText, DT_LEFT|DT_WORDBREAK)
		return
	}

	// Successful Installation Banner
	drawChamferBox(hdc, bannerRect, sc(10), sc(10), sc(10), sc(10), colCyanBg, colCyan, 1)
	fillSolidRect(hdc, RECT{bannerRect.Left, bannerRect.Top + sc(8), bannerRect.Left + sc(4), bannerRect.Bottom - sc(8)}, colGreen)
	drawText(hdc, "SYSTEM BEREIT // SC COCKPIT ERFOLGREICH INSTALLIERT", RECT{bannerRect.Left + sc(18), bannerRect.Top + sc(12), bannerRect.Right - sc(16), bannerRect.Top + sc(34)}, app.FontBold, colGreen, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, "Starte SC Cockpit auf dem Gaming-PC und öffne die angezeigte http://192.168.x.x:8765 Adresse im Browser deines Tablets oder Smartphones im selben WLAN.",
		RECT{bannerRect.Left + sc(18), bannerRect.Top + sc(36), bannerRect.Right - sc(16), bannerRect.Bottom - sc(10)}, app.FontBody, colText, DT_LEFT|DT_WORDBREAK)

	cardY := bannerRect.Bottom + sc(16)
	cardH := sc(52)
	gap := sc(10)

	drawCheckboxCard(hdc, CtrlChkLaunchNow,
		RECT{padX, cardY, w - padX, cardY + cardH},
		"SC Cockpit Server jetzt sofort starten",
		"Öffnet das Server-Fenster mit deiner WLAN-Adresse für Tablet & Smartphone",
		app.OptLaunchNow, colAmber)

	cardY += cardH + gap
	drawCheckboxCard(hdc, CtrlChkOpenDocs,
		RECT{padX, cardY, w - padX, cardY + cardH},
		"Installationsordner & Dashboard-Dokumentation im Explorer anzeigen",
		fmt.Sprintf("Öffnet %s", app.InstallDir),
		app.OptOpenDocs, colCyan)
}

func handleControlClick(id int) {
	switch id {
	case CtrlClose, CtrlBtnSecondary:
		if app.Page != PageInstalling {
			pPostQuitMessage.Call(0)
		}
	case CtrlMinimize:
		pShowWindow.Call(app.Hwnd, SW_MINIMIZE)
	case CtrlChkDesktop:
		app.OptDesktop = !app.OptDesktop
	case CtrlChkStartMenu:
		app.OptStartMenu = !app.OptStartMenu
	case CtrlChkWingetPy:
		app.OptWingetPy = !app.OptWingetPy
	case CtrlChkLaunchNow:
		app.OptLaunchNow = !app.OptLaunchNow
	case CtrlChkOpenDocs:
		app.OptOpenDocs = !app.OptOpenDocs
	case CtrlChkPurgeData:
		app.OptPurgeData = !app.OptPurgeData
	case CtrlBtnModeSwitch:
		if app.Page == PageWelcome {
			app.Page = PageUninstallConfirm
		} else if app.Page == PageUninstallConfirm {
			app.Page = PageWelcome
		}
	case CtrlBtnPrimary:
		switch app.Page {
		case PageWelcome:
			app.IsUninstall = false
			app.Page = PageInstalling
			pSetTimer.Call(app.Hwnd, 1, 120, 0)
			go runInstallWorker()
		case PageUninstallConfirm:
			app.IsUninstall = true
			app.Page = PageInstalling
			pSetTimer.Call(app.Hwnd, 1, 120, 0)
			go runUninstallWorker()
		case PageDone:
			finishAndExit()
		}
	}
	pInvalidateRect.Call(app.Hwnd, 0, 0)
}

func u16ptr(s string) uintptr {
	p, _ := syscall.UTF16PtrFromString(s)
	return uintptr(unsafe.Pointer(p))
}

func finishAndExit() {
	if !app.IsUninstall && app.InstallErr == "" {
		if app.OptLaunchNow {
			bat := filepath.Join(app.InstallDir, "Starten.bat")
			pShellExecuteW.Call(0, u16ptr("open"), u16ptr(bat), 0, u16ptr(app.InstallDir), SW_SHOWNORMAL)
		}
		if app.OptOpenDocs {
			pShellExecuteW.Call(0, u16ptr("open"), u16ptr(app.InstallDir), 0, 0, SW_SHOWNORMAL)
		}
	}
	pPostQuitMessage.Call(0)
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

		paintHUD(memDC, w, h)

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
		if pt.Y < sc(72) {
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
		if wParam == VK_ESCAPE && app.Page != PageInstalling {
			pPostQuitMessage.Call(0)
		} else if wParam == VK_RETURN && (app.Page == PageWelcome || app.Page == PageDone) {
			handleControlClick(CtrlBtnPrimary)
		}
		return 0

	case WM_TIMER:
		app.mu.Lock()
		app.AnimTick++
		app.mu.Unlock()
		pInvalidateRect.Call(hwnd, 0, 0)
		return 0

	case WM_APP_PROGRESS:
		pInvalidateRect.Call(hwnd, 0, 0)
		return 0

	case WM_APP_DONE:
		pKillTimer.Call(hwnd, 1)
		app.Page = PageDone
		pInvalidateRect.Call(hwnd, 0, 0)
		return 0

	case WM_CLOSE:
		if app.Page != PageInstalling {
			pPostQuitMessage.Call(0)
		}
		return 0

	case WM_DESTROY:
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
		0, // DEFAULT_CHARSET
		0, 0,
		CLEARTYPE_QUALITY,
		0,
		uintptr(unsafe.Pointer(u16(face))),
	)
	return f
}

func detectEnvironment() {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		home, _ := os.UserHomeDir()
		localAppData = filepath.Join(home, "AppData", "Local")
	}
	root := filepath.Join(localAppData, "SC Cockpit")
	app.InstallDir = filepath.Join(root, "app")
	app.UserDir = filepath.Join(root, "user", "dashboards")

	if _, err := os.Stat(filepath.Join(app.InstallDir, "server.py")); err == nil {
		app.ExistingInst = true
	}

	// Check command line flags
	for _, arg := range os.Args[1:] {
		lower := strings.ToLower(arg)
		if lower == "--uninstall" || lower == "/uninstall" || lower == "-uninstall" {
			app.IsUninstall = true
			app.Page = PageUninstallConfirm
		} else if lower == "-purge" || lower == "--purge" {
			app.OptPurgeData = true
		}
	}

	// Check Python 3 availability without flashing console windows
	for _, cmdName := range []string{"py", "python"} {
		if path, err := exec.LookPath(cmdName); err == nil && path != "" {
			cmd := exec.Command(cmdName, "--version")
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			if out, err := cmd.CombinedOutput(); err == nil {
				ver := strings.TrimSpace(string(out))
				if strings.HasPrefix(ver, "Python 3") {
					app.PythonFound = true
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
}

func main() {
	runtime.LockOSThread()
	pSetProcessDPIAware.Call()

	detectEnvironment()

	// Determine DPI scale factor
	screenDC, _, _ := user32.NewProc("GetDC").Call(0)
	dpi, _, _ := pGetDeviceCaps.Call(screenDC, 88) // LOGPIXELSX
	user32.NewProc("ReleaseDC").Call(0, screenDC)
	if dpi < 96 {
		dpi = 96
	}
	app.Scale = float64(dpi) / 96.0

	app.Width = sc(760)
	app.Height = sc(510)

	app.CursorArrow, _, _ = pLoadCursorW.Call(0, IDC_ARROW)
	app.CursorHand, _, _ = pLoadCursorW.Call(0, IDC_HAND)

	app.FontHero = makeFont(22, FW_BOLD, "Bahnschrift")
	app.FontTitle = makeFont(17, FW_BOLD, "Bahnschrift")
	app.FontSub = makeFont(14, FW_SEMIBOLD, "Segoe UI")
	app.FontBody = makeFont(13, FW_NORMAL, "Segoe UI")
	app.FontBold = makeFont(14, FW_SEMIBOLD, "Segoe UI")
	app.FontMono = makeFont(12, FW_SEMIBOLD, "Consolas")
	app.FontSmall = makeFont(12, FW_NORMAL, "Segoe UI")

	hInst, _, _ := pGetModuleHandleW.Call(0)
	className := u16("SCCockpitInstallerHUD")

	// Load embedded application icon (resource ID 1)
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
		uintptr(unsafe.Pointer(u16("SC Cockpit Setup"))),
		WS_POPUP|WS_VISIBLE|WS_SYSMENU|WS_MINIMIZEBOX|WS_CLIPCHILDREN,
		uintptr(winX), uintptr(winY),
		uintptr(app.Width), uintptr(app.Height),
		0, 0, hInst, 0,
	)
	app.Hwnd = hwnd

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
}
