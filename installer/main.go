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
	WS_THICKFRAME   = 0x00040000
	WS_MINIMIZEBOX  = 0x00020000
	WS_MAXIMIZEBOX  = 0x00010000
	WS_CLIPCHILDREN = 0x02000000
	WS_EX_APPWINDOW = 0x00040000

	WM_DESTROY       = 0x0002
	WM_SIZE          = 0x0005
	WM_PAINT         = 0x000F
	WM_CLOSE         = 0x0010
	WM_ERASEBKGND    = 0x0014
	WM_SETCURSOR     = 0x0020
	WM_NCHITTEST     = 0x0084
	WM_NCLBUTTONDBLCLK = 0x00A3
	WM_KEYDOWN       = 0x0100
	WM_CHAR          = 0x0102
	WM_MOUSEMOVE     = 0x0200
	WM_LBUTTONDOWN   = 0x0201
	WM_LBUTTONUP     = 0x0202
	WM_LBUTTONDBLCLK = 0x0203
	WM_MOUSEWHEEL    = 0x020A
	WM_APP_REFRESH   = 0x8001
	WM_APP_DONE      = 0x8002

	HTCLIENT      = 1
	HTCAPTION     = 2
	HTLEFT        = 10
	HTRIGHT       = 11
	HTTOP         = 12
	HTTOPLEFT     = 13
	HTTOPRIGHT    = 14
	HTBOTTOM      = 15
	HTBOTTOMLEFT  = 16
	HTBOTTOMRIGHT = 17

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
	SW_MAXIMIZE   = 3
	SW_MINIMIZE   = 6
	SW_RESTORE    = 9

	CS_HREDRAW    = 0x0002
	CS_VREDRAW    = 0x0001
	CS_DBLCLKS    = 0x0008
	CS_DROPSHADOW = 0x00020000

	VK_BACK   = 0x08
	VK_ESCAPE = 0x1B
	VK_PRIOR  = 0x21
	VK_NEXT   = 0x22
	VK_UP     = 0x26
	VK_DOWN   = 0x28

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
	pIsZoomed           = user32.NewProc("IsZoomed")
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
	TabShards             // 2. HANGAR & SCHIFFE (Search 100+ Ships, Favorites ★, Subsystems)
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
	CtrlWinMax
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
	CtrlLiveAlwaysOnTop
	CtrlLiveGitPull
	CtrlLiveOpenEditor
	CtrlLiveOpenHangar

	// Hangar & Ship Search Controls
	CtrlBtnNewDash
	CtrlBtnSyncSCBindings
	CtrlBtnOpenDashFolder
	CtrlBtnReloadDash
	CtrlAddSubMining
	CtrlAddSubSalvage
	CtrlAddSubCombat
	CtrlAddSubCargo
	CtrlAddSubPower
	CtrlHangarSearchBox
	CtrlHangarSearchClear
	CtrlHangarFilterFav
	CtrlHangarFilterAll
	CtrlHangarFilterCombat
	CtrlHangarFilterMining
	CtrlHangarFilterSalvage
	CtrlHangarFilterCargo
	CtrlHangarFilterExplore
	CtrlHangarPagePrev
	CtrlHangarPageNext

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

	// Dynamic Ranges (must stay at end of const block!)
	CtrlDashUseLiveBase  = 1000 // + index (Opens ship directly in Live Cockpit tab)
	CtrlDashEditBase     = 2000 // + index
	CtrlDashDupBase      = 3000 // + index
	CtrlDashDelBase      = 4000 // + index
	CtrlLivePageTabBase  = 5000 // + page index
	CtrlLiveTileBtnBase  = 6000 // + flat button index in current page
	CtrlLiveShipChipBase = 7000 // + ship index in Hangar quick bar
	CtrlDashFavBase      = 8000 // + index (Toggles ★ Favorite in Hangar)
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
	ID           string
	Title        string
	Ship         string
	Manufacturer string
	Role         string
	HullType     string
	Subsystems   []string
	Favorite     bool
	Source       string
	Path         string
	PageCount    int
	ButtonCount  int
	Pages        []DashPage
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

	// Dashboards, Hangar Search & In-App Live Cockpit Deck State
	Dashboards      []DashboardInfo
	ActiveDash      string
	LivePageIdx     int
	LiveArmed       bool
	AlwaysOnTop     bool
	LiveToggled     map[string]bool
	LivePageBtns    []DashButton // Flattened buttons on current Live Cockpit page
	LastActionTxt   string
	HangarFilter    string // "hangar", "all", "kampf", "mining", "salvage", "cargo", "exploration"
	HangarSearch    string
	HangarScrollIdx int

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
	HangarFilter: "hangar",
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

// Draw a stylized Star Citizen ship silhouette inside the given bounding rectangle (GDI Polygon)
func drawShipSilhouette(hdc uintptr, r RECT, hullType string, strokeCol, fillCol uint32) {
	cx := (r.Left + r.Right) / 2
	cy := (r.Top + r.Bottom) / 2
	hw := (r.Right - r.Left) / 2
	hh := (r.Bottom - r.Top) / 2
	if hw < 10 || hh < 8 {
		return
	}
	pt := func(nx, ny float64) POINT {
		return POINT{
			X: cx + int32(nx*float64(hw)),
			Y: cy + int32(ny*float64(hh)),
		}
	}

	var poly []POINT
	switch hullType {
	case "twin_industrial": // Drake Vulture / Cutlass / Caterpillar twin-prong industrial
		poly = []POINT{
			pt(-0.35, -0.82), pt(-0.15, -0.82), pt(-0.12, -0.20), pt(0.12, -0.20),
			pt(0.15, -0.82), pt(0.35, -0.82), pt(0.45, -0.10), pt(0.85, 0.15),
			pt(0.88, 0.45), pt(0.50, 0.55), pt(0.42, 0.85), pt(-0.42, 0.85),
			pt(-0.50, 0.55), pt(-0.88, 0.45), pt(-0.85, 0.15), pt(-0.45, -0.10),
		}
	case "mining_bubble": // MISC Prospector / Argo Mole
		poly = []POINT{
			pt(0.0, -0.85), pt(0.28, -0.70), pt(0.35, -0.30), pt(0.82, -0.05),
			pt(0.88, 0.30), pt(0.60, 0.50), pt(0.48, 0.85), pt(-0.48, 0.85),
			pt(-0.60, 0.50), pt(-0.88, 0.30), pt(-0.82, -0.05), pt(-0.35, -0.30),
			pt(-0.28, -0.70),
		}
	case "crusader_wing": // Crusader C2 / Spirit / Mercury Star Runner
		poly = []POINT{
			pt(0.0, -0.85), pt(0.25, -0.45), pt(0.95, 0.15), pt(0.88, 0.45),
			pt(0.45, 0.40), pt(0.32, 0.85), pt(-0.32, 0.85), pt(-0.45, 0.40),
			pt(-0.88, 0.45), pt(-0.95, 0.15), pt(-0.25, -0.45),
		}
	case "carrack_expedition", "constellation_nacelle", "hauler_hull": // Carrack / Constellation / Hull
		poly = []POINT{
			pt(0.0, -0.88), pt(0.28, -0.52), pt(0.38, -0.10), pt(0.80, 0.10),
			pt(0.75, 0.45), pt(0.38, 0.52), pt(0.32, 0.88), pt(-0.32, 0.88),
			pt(-0.38, 0.52), pt(-0.75, 0.45), pt(-0.80, 0.10), pt(-0.38, -0.10),
			pt(-0.28, -0.52),
		}
	default: // fighter_delta / fighter_anvil (Gladius / Sabre / Hornet / F8C)
		poly = []POINT{
			pt(0.0, -0.88), pt(0.18, -0.35), pt(0.92, 0.25), pt(0.92, 0.48),
			pt(0.35, 0.45), pt(0.25, 0.85), pt(0.0, 0.72), pt(-0.25, 0.85),
			pt(-0.35, 0.45), pt(-0.92, 0.48), pt(-0.92, 0.25), pt(-0.18, -0.35),
		}
	}

	// HUD targeting crosshairs behind the ship
	drawLine(hdc, r.Left+sc(4), cy, r.Right-sc(4), cy, colLine, 1)
	drawLine(hdc, cx, r.Top+sc(4), cx, r.Bottom-sc(4), colLine, 1)

	brush, _, _ := pCreateSolidBrush.Call(uintptr(fillCol))
	pen, _, _ := pCreatePen.Call(PS_SOLID, uintptr(sc(1)), uintptr(strokeCol))
	oldB, _, _ := pSelectObject.Call(hdc, brush)
	oldP, _, _ := pSelectObject.Call(hdc, pen)
	pPolygon.Call(hdc, uintptr(unsafe.Pointer(&poly[0])), uintptr(len(poly)))
	pSelectObject.Call(hdc, oldB)
	pSelectObject.Call(hdc, oldP)
	pDeleteObject.Call(brush)
	pDeleteObject.Call(pen)
}

func isWindowMaximized() bool {
	if app.Hwnd == 0 {
		return false
	}
	z, _, _ := pIsZoomed.Call(app.Hwnd)
	return z != 0
}

func toggleMaximizeWindow() {
	if app.Hwnd == 0 {
		return
	}
	if isWindowMaximized() {
		pShowWindow.Call(app.Hwnd, SW_RESTORE)
	} else {
		pShowWindow.Call(app.Hwnd, SW_MAXIMIZE)
	}
}

func paintWindow(hdc uintptr, w, h int32) {
	app.Regions = app.Regions[:0]
	fillRect(hdc, RECT{0, 0, w, h}, colBg)

	if app.Mode == ModeInstaller {
		paintInstallerMode(hdc, w, h)
	} else {
		paintLauncherMode(hdc, w, h)
	}

	// Top-Right Window Controls: Minimize (—), Maximize/Restore (□ / ❐), Close (✕)
	btnW := sc(34)
	btnH := sc(28)
	closeRect := RECT{w - sc(12) - btnW, sc(12), w - sc(12), sc(12) + btnH}
	maxRect := RECT{closeRect.Left - sc(5) - btnW, sc(12), closeRect.Left - sc(5), sc(12) + btnH}
	minRect := RECT{maxRect.Left - sc(5) - btnW, sc(12), maxRect.Left - sc(5), sc(12) + btnH}

	app.Regions = append(app.Regions, HotRegion{ID: CtrlWinMin, Rect: minRect})
	app.Regions = append(app.Regions, HotRegion{ID: CtrlWinMax, Rect: maxRect})
	app.Regions = append(app.Regions, HotRegion{ID: CtrlWinClose, Rect: closeRect})

	if app.HoverID == CtrlWinMin {
		drawShardBox(hdc, minRect, 5, colPanelHov, colLine)
	}
	drawText(hdc, "—", minRect, app.FontBold, colMuted, DT_CENTER|DT_VCENTER|DT_SINGLELINE)

	maxCol := colMuted
	if app.HoverID == CtrlWinMax {
		drawShardBox(hdc, maxRect, 5, colPanelHov, colCyan)
		maxCol = colCyan
	}
	maxGlyph := "□"
	if isWindowMaximized() {
		maxGlyph = "❐"
	}
	drawText(hdc, maxGlyph, maxRect, app.FontBold, maxCol, DT_CENTER|DT_VCENTER|DT_SINGLELINE)

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

	drawText(hdc, fmt.Sprintf("BUILD %s\n100+ STAR CITIZEN SCHIFFE\nIN-APP LIVE DECK & WI-FI", AppVersion),
		RECT{sc(24), h - sc(80), sideW - sc(16), h - sc(18)}, app.FontSmall, colDim, DT_LEFT|DT_WORDBREAK)

	cx := sideW + sc(36)
	cw := w - cx - sc(36)

	switch app.InstPage {
	case InstPageWelcome:
		title := "SC COCKPIT // GAMEGLASS EDITION"
		sub := "Installiert das interaktive Desktop- & Tablet-Shard-Deck für alle Star Citizen Schiffe."
		if app.ExistingInst {
			title = "SC COCKPIT // UPDATE & VERWALTUNG"
			sub = "Bestehende Installation erkannt. Deine eigenen Shards & Hangar-Favoriten bleiben erhalten."
		}
		drawText(hdc, title, RECT{cx, sc(30), cx + cw - sc(115), sc(58)}, app.FontHero, colText, DT_LEFT|DT_SINGLELINE)
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
		drawText(hdc, "SC COCKPIT ENTFERNEN", RECT{cx, sc(30), cx + cw - sc(115), sc(58)}, app.FontHero, colRed, DT_LEFT|DT_SINGLELINE)
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
// 2. GAMEGLASS IN-APP COCKPIT & HANGAR CONTROL CENTER
// ============================================================================
func paintLauncherMode(hdc uintptr, w, h int32) {
	sideW := sc(225)
	fillRect(hdc, RECT{0, 0, sideW, h}, colSidebar)
	drawLine(hdc, sideW, 0, sideW, h, colLine, 1)

	hexRect := RECT{sc(18), sc(18), sc(48), sc(48)}
	drawShardBox(hdc, hexRect, 7, colCyanBg, colCyan)
	drawText(hdc, "SC", hexRect, app.FontBold, colCyan, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
	drawText(hdc, "SC COCKPIT", RECT{sc(58), sc(17), sideW - sc(10), sc(36)}, app.FontTitle, colText, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, "GAMEGLASS SHARD HUB", RECT{sc(58), sc(36), sideW - sc(10), sc(51)}, app.FontSmall, colCyan, DT_LEFT|DT_SINGLELINE)

	favCount := 0
	for _, d := range app.Dashboards {
		if d.Favorite {
			favCount++
		}
	}

	navItems := []struct {
		id    int
		tab   int
		label string
		tag   string
	}{
		{CtrlNavLiveCockpit, TabLiveCockpit, "► LIVE COCKPIT", "DECK"},
		{CtrlNavShards, TabShards, "★ HANGAR & SCHIFFE", fmt.Sprintf("%d/%d", favCount, len(app.Dashboards))},
		{CtrlNavHostLink, TabHostLink, "TABLET / SERVER", "WLAN"},
		{CtrlNavSettings, TabSettings, "EINSTELLUNGEN", "SYS"},
	}

	ny := sc(70)
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
		drawText(hdc, item.label, RECT{r.Left + sc(12), r.Top, r.Right - sc(50), r.Bottom}, app.FontBold, col, DT_LEFT|DT_VCENTER|DT_SINGLELINE)
		drawText(hdc, item.tag, RECT{r.Right - sc(50), r.Top, r.Right - sc(8), r.Bottom}, app.FontMono, tagCol, DT_RIGHT|DT_VCENTER|DT_SINGLELINE)
		ny += sc(50)
	}

	// Active Ship Blueprint Card inside Left Sidebar
	var curDash *DashboardInfo
	for i := range app.Dashboards {
		if app.Dashboards[i].ID == app.ActiveDash {
			curDash = &app.Dashboards[i]
			break
		}
	}
	if curDash != nil && h > sc(520) {
		bpCard := RECT{sc(12), ny + sc(8), sideW - sc(12), ny + sc(124)}
		app.Regions = append(app.Regions, HotRegion{ID: CtrlLiveOpenHangar, Rect: bpCard})
		drawShardBox(hdc, bpCard, 8, rgb(0x07, 0x12, 0x1c), colLineHi)
		drawText(hdc, curDash.Manufacturer+" // AKTIVES SCHIFF", RECT{bpCard.Left + sc(10), bpCard.Top + sc(6), bpCard.Right - sc(8), bpCard.Top + sc(20)}, app.FontSmall, colCyan, DT_LEFT|DT_SINGLELINE)
		silRect := RECT{bpCard.Left + sc(16), bpCard.Top + sc(22), bpCard.Right - sc(16), bpCard.Bottom - sc(24)}
		drawShipSilhouette(hdc, silRect, curDash.HullType, colCyan, rgb(0x0b, 0x28, 0x38))
		drawText(hdc, strings.ToUpper(curDash.Ship), RECT{bpCard.Left + sc(10), bpCard.Bottom - sc(22), bpCard.Right - sc(8), bpCard.Bottom - sc(4)}, app.FontBold, colAmber, DT_CENTER|DT_SINGLELINE)
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
	cx := sideW + sc(20)
	cw := w - cx - sc(20)

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
		toastRect := RECT{cx, h - sc(40), cx + cw, h - sc(10)}
		drawShardBox(hdc, toastRect, 6, colCyanBg, colCyan)
		drawText(hdc, app.ToastMsg, toastRect, app.FontBold, colCyan, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
	}
}

// ============================================================================
// TAB 1: IN-APP INTERACTIVE LIVE COCKPIT (HANGAR QUICK BAR + SUBSYSTEMS + TILES)
// ============================================================================
func paintTabLiveCockpit(hdc uintptr, cx, cw, h int32) {
	app.LivePageBtns = app.LivePageBtns[:0]

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

	// 1. Top MFD Active Ship Banner (with mini ship silhouette!) & Action Bar
	topBar := RECT{cx, sc(12), cx + cw - sc(118), sc(52)}
	drawShardBox(hdc, topBar, 8, colPanel, colLineHi)
	fillRect(hdc, RECT{topBar.Left, topBar.Top, topBar.Left + sc(65), topBar.Top + sc(2)}, colCyan)

	btnPrev := RECT{topBar.Left + sc(6), topBar.Top + sc(6), topBar.Left + sc(32), topBar.Bottom - sc(6)}
	drawButton(hdc, CtrlLivePrevBoard, btnPrev, "◄", "secondary")

	// Mini Ship Blueprint Silhouette next to title
	miniSil := RECT{btnPrev.Right + sc(6), topBar.Top + sc(5), btnPrev.Right + sc(54), topBar.Bottom - sc(5)}
	if curDash != nil {
		drawShipSilhouette(hdc, miniSil, curDash.HullType, colCyan, rgb(0x09, 0x22, 0x30))
	}

	dashTitle := "KEIN SCHIFF"
	if curDash != nil {
		shipLbl := curDash.Ship
		if shipLbl == "" {
			shipLbl = curDash.Title
		}
		dashTitle = strings.ToUpper(shipLbl)
	}
	drawText(hdc, dashTitle, RECT{miniSil.Right + sc(6), topBar.Top, miniSil.Right + sc(200), topBar.Bottom}, app.FontTitle, colAmber, DT_LEFT|DT_VCENTER|DT_SINGLELINE)

	btnNext := RECT{miniSil.Right + sc(204), topBar.Top + sc(6), miniSil.Right + sc(230), topBar.Bottom - sc(6)}
	drawButton(hdc, CtrlLiveNextBoard, btnNext, "►", "secondary")

	btnEdit := RECT{topBar.Right - sc(102), topBar.Top + sc(6), topBar.Right - sc(6), topBar.Bottom - sc(6)}
	drawButton(hdc, CtrlLiveOpenEditor, btnEdit, "BAUKASTEN", "cyan")

	btnHangar := RECT{btnEdit.Left - sc(108), topBar.Top + sc(6), btnEdit.Left - sc(6), topBar.Bottom - sc(6)}
	drawButton(hdc, CtrlLiveOpenHangar, btnHangar, "★ HANGAR", "amber")

	btnPull := RECT{btnHangar.Left - sc(94), topBar.Top + sc(6), btnHangar.Left - sc(6), topBar.Bottom - sc(6)}
	drawButton(hdc, CtrlLiveGitPull, btnPull, "⟳ UPDATE", "secondary")

	btnPin := RECT{btnPull.Left - sc(102), topBar.Top + sc(6), btnPull.Left - sc(6), topBar.Bottom - sc(6)}
	if app.AlwaysOnTop {
		drawButton(hdc, CtrlLiveAlwaysOnTop, btnPin, "📌 OVERLAY AN", "amber")
	} else {
		drawButton(hdc, CtrlLiveAlwaysOnTop, btnPin, "📌 OVERLAY", "secondary")
	}

	btnArm := RECT{btnPin.Left - sc(114), topBar.Top + sc(6), btnPin.Left - sc(6), topBar.Bottom - sc(6)}
	if app.LiveArmed {
		drawButton(hdc, CtrlLiveArmToggle, btnArm, "WAFFEN ARMED", "danger")
	} else {
		drawButton(hdc, CtrlLiveArmToggle, btnArm, "WAFFEN SAFE", "secondary")
	}

	// 2. 1-Tap Hangar Quick Bar (Shows ONLY the user's ★ Favorited Hangar ships!)
	fleetY := sc(58)
	fleetH := sc(34)
	var favIndices []int
	for i, d := range app.Dashboards {
		if d.Favorite || d.ID == app.ActiveDash {
			favIndices = append(favIndices, i)
		}
	}
	if len(favIndices) > 0 {
		maxChips := int(cw / sc(145))
		if maxChips < 4 {
			maxChips = 4
		}
		if maxChips > len(favIndices) {
			maxChips = len(favIndices)
		}
		activePos := 0
		for pos, idx := range favIndices {
			if app.Dashboards[idx].ID == app.ActiveDash {
				activePos = pos
				break
			}
		}
		startPos := 0
		if activePos >= maxChips {
			startPos = activePos - maxChips + 1
		}
		chipW := (cw - int32(maxChips-1)*sc(6)) / int32(maxChips)
		if chipW > sc(185) {
			chipW = sc(185)
		}
		sx := cx
		for c := 0; c < maxChips; c++ {
			pos := startPos + c
			if pos >= len(favIndices) {
				break
			}
			si := favIndices[pos]
			ship := app.Dashboards[si]
			sr := RECT{sx, fleetY, sx + chipW, fleetY + fleetH}
			cid := CtrlLiveShipChipBase + si
			app.Regions = append(app.Regions, HotRegion{ID: cid, Rect: sr})
			isActShip := ship.ID == app.ActiveDash
			hov := app.HoverID == cid

			sName := ship.Ship
			if sName == "" {
				sName = ship.Title
			}
			label := "★ " + strings.ToUpper(sName)
			if isActShip {
				drawShardBox(hdc, sr, 6, colAmberBg, colAmber)
				fillRect(hdc, RECT{sr.Left, sr.Top + sc(4), sr.Left + sc(3), sr.Bottom - sc(6)}, colAmber)
				drawText(hdc, label, RECT{sr.Left + sc(8), sr.Top, sr.Right - sc(6), sr.Bottom}, app.FontBold, colText, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
			} else if hov {
				drawShardBox(hdc, sr, 6, colPanelHov, colCyan)
				drawText(hdc, label, RECT{sr.Left + sc(8), sr.Top, sr.Right - sc(6), sr.Bottom}, app.FontBold, colCyan, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
			} else {
				drawShardBox(hdc, sr, 6, colPanel, colLine)
				drawText(hdc, label, RECT{sr.Left + sc(8), sr.Top, sr.Right - sc(6), sr.Bottom}, app.FontSmall, colMuted, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
			}
			sx += chipW + sc(6)
		}
	}

	if curDash == nil || len(curDash.Pages) == 0 {
		drawText(hdc, "Keine Unterpunkte in diesem Schiff gefunden.", RECT{cx, sc(130), cx + cw, sc(170)}, app.FontBody, colMuted, DT_CENTER|DT_SINGLELINE)
		return
	}

	if app.LivePageIdx >= len(curDash.Pages) || app.LivePageIdx < 0 {
		app.LivePageIdx = 0
	}

	// 3. Ship Subsystem Tabs Row (Unterpunkte: Mining, Salvage, Flug, Kampf, Energie...)
	tabY := sc(98)
	tabH := sc(34)
	numPages := len(curDash.Pages)
	tabW := sc(150)
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
			drawShardBox(hdc, tr, 6, colCyanBg, colCyan)
			fillRect(hdc, RECT{tr.Left, tr.Top, tr.Right - sc(10), tr.Top + sc(2)}, colCyan)
			drawText(hdc, strings.ToUpper(page.Label), tr, app.FontBold, colCyan, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
		} else if hov {
			drawShardBox(hdc, tr, 6, colPanelHov, colLineHi)
			drawText(hdc, strings.ToUpper(page.Label), tr, app.FontBold, colText, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
		} else {
			drawShardBox(hdc, tr, 6, colPanel, colLine)
			drawText(hdc, strings.ToUpper(page.Label), tr, app.FontBold, colMuted, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
		}
		tx += tabW + sc(8)
	}

	// 4. Watermark Ship Silhouette in Cockpit Background + Responsive Grid of GameGlass Shards
	stageRect := RECT{cx + cw/5, tabY + sc(45), cx + (cw*4)/5, h - sc(45)}
	drawShipSilhouette(hdc, stageRect, curDash.HullType, rgb(0x13, 0x2d, 0x3d), rgb(0x07, 0x11, 0x1a))

	activePage := curDash.Pages[app.LivePageIdx]
	gy := tabY + tabH + sc(10)
	maxBottom := h - sc(44)

	// Responsive columns when maximized or resized wider!
	cols := int32(4)
	if cw >= sc(1150) {
		cols = 6
	} else if cw >= sc(920) {
		cols = 5
	}
	gap := sc(9)

	for _, grp := range activePage.Groups {
		if gy+sc(65) > maxBottom {
			break
		}
		drawText(hdc, "◆  "+strings.ToUpper(grp.Title), RECT{cx, gy, cx + cw, gy + sc(18)}, app.FontBold, colAmber, DT_LEFT|DT_SINGLELINE)
		drawLine(hdc, cx+sc(230), gy+sc(9), cx+cw, gy+sc(9), colLine, 1)
		gy += sc(22)

		cellW := (cw - (cols-1)*gap) / cols
		cellH := sc(64)
		if h >= sc(800) {
			cellH = sc(74)
		}
		colIdx := int32(0)

		var expanded []DashButton
		for _, b := range grp.Buttons {
			if b.Kind == "power" {
				expanded = append(expanded,
					DashButton{ID: b.ID + "-wep", Label: "PWR ► WAFFEN", Hint: "F5", Keys: []string{"f5"}, Mode: "tap", Color: "red"},
					DashButton{ID: b.ID + "-eng", Label: "PWR ► ANTRIEB", Hint: "F6", Keys: []string{"f6"}, Mode: "tap", Color: "amber"},
					DashButton{ID: b.ID + "-shd", Label: "PWR ► SCHILDE", Hint: "F7", Keys: []string{"f7"}, Mode: "tap", Color: "cyan"},
					DashButton{ID: b.ID + "-rst", Label: "PWR ► BALANCE", Hint: "F8", Keys: []string{"f8"}, Mode: "tap", Color: "slate"},
				)
			} else if b.Kind == "shield" {
				expanded = append(expanded,
					DashButton{ID: b.ID + "-f", Label: "SCHILD ► BUG", Hint: "NUM8", Keys: []string{"num8"}, Mode: "tap", Color: "cyan"},
					DashButton{ID: b.ID + "-b", Label: "SCHILD ► HECK", Hint: "NUM2", Keys: []string{"num2"}, Mode: "tap", Color: "cyan"},
					DashButton{ID: b.ID + "-l", Label: "SCHILD ► LINKS", Hint: "NUM4", Keys: []string{"num4"}, Mode: "tap", Color: "blue"},
					DashButton{ID: b.ID + "-r", Label: "SCHILD ► RECHTS", Hint: "NUM6", Keys: []string{"num6"}, Mode: "tap", Color: "blue"},
					DashButton{ID: b.ID + "-rst", Label: "SCHILD ► RESET", Hint: "NUM5", Keys: []string{"num5"}, Mode: "tap", Color: "amber", Wide: true},
				)
			} else {
				expanded = append(expanded, b)
			}
		}

		for _, btn := range expanded {
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
	fillRect(hdc, RECT{r.Left, r.Top + sc(7), r.Left + sc(3), r.Bottom - sc(10)}, accent)

	if btn.Danger {
		stripeCol := colRed
		if !app.LiveArmed {
			stripeCol = colDim
		}
		fillRect(hdc, RECT{r.Left + sc(6), r.Top, r.Right - sc(12), r.Top + sc(3)}, stripeCol)
	}

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

	drawText(hdc, keyHint, RECT{r.Left + sc(12), r.Top + sc(7), r.Right - sc(55), r.Top + sc(22)}, app.FontMono, accent, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, modeTag, RECT{r.Right - sc(65), r.Top + sc(7), r.Right - sc(10), r.Top + sc(22)}, app.FontSmall, colMuted, DT_RIGHT|DT_SINGLELINE)

	labelCol := colText
	if btn.Danger && !app.LiveArmed {
		labelCol = colMuted
	} else if isToggled {
		labelCol = colCyan
	}
	drawText(hdc, strings.ToUpper(btn.Label), RECT{r.Left + sc(12), r.Top + sc(25), r.Right - sc(10), r.Bottom - sc(6)}, app.FontTitle, labelCol, DT_LEFT|DT_VCENTER|DT_SINGLELINE)
}

// ============================================================================
// TAB 2: ★ HANGAR & STAR CITIZEN SHIP DATABASE (SEARCH 100+ SHIPS & FAVORITES)
// ============================================================================
func getFilteredHangarShips() []int {
	var indices []int
	q := strings.ToLower(strings.TrimSpace(app.HangarSearch))
	for i, d := range app.Dashboards {
		if app.HangarFilter == "hangar" && !d.Favorite {
			continue
		}
		if app.HangarFilter != "all" && app.HangarFilter != "hangar" {
			hasSub := false
			for _, s := range d.Subsystems {
				if strings.EqualFold(s, app.HangarFilter) {
					hasSub = true
					break
				}
			}
			if !hasSub && !strings.Contains(strings.ToLower(d.Role), app.HangarFilter) {
				continue
			}
		}
		if q != "" {
			hay := strings.ToLower(d.Ship + " " + d.Title + " " + d.Manufacturer + " " + d.Role + " " + strings.Join(d.Subsystems, " "))
			if !strings.Contains(hay, q) {
				continue
			}
		}
		indices = append(indices, i)
	}
	return indices
}

func paintTabShards(hdc uintptr, cx, cw, h int32) {
	drawText(hdc, "★ STAR CITIZEN HANGAR & SCHIFFS-DATENBANK", RECT{cx, sc(16), cx + cw - sc(120), sc(42)}, app.FontHero, colText, DT_LEFT|DT_SINGLELINE)
	drawText(hdc, "Suche aus über 100 Star Citizen Schiffen, markiere Favoriten (★) für deinen Hangar oder passe Unterpunkte an.",
		RECT{cx, sc(42), cx + cw, sc(62)}, app.FontBody, colMuted, DT_LEFT|DT_SINGLELINE)

	// Row 1: Interactive Search Input + Quick Actions
	searchBox := RECT{cx, sc(68), cx + sc(310), sc(100)}
	app.Regions = append(app.Regions, HotRegion{ID: CtrlHangarSearchBox, Rect: searchBox})
	drawShardBox(hdc, searchBox, 6, rgb(0x08, 0x13, 0x1f), colCyan)
	searchTxt := "🔍 Tippe zum Suchen (z.B. Vulture, Gladius, Carrack)..."
	searchCol := colMuted
	if app.HangarSearch != "" {
		searchTxt = "🔍 " + app.HangarSearch + "_"
		searchCol = colCyan
	}
	drawText(hdc, searchTxt, RECT{searchBox.Left + sc(10), searchBox.Top, searchBox.Right - sc(30), searchBox.Bottom}, app.FontBold, searchCol, DT_LEFT|DT_VCENTER|DT_SINGLELINE)
	if app.HangarSearch != "" {
		clrRect := RECT{searchBox.Right - sc(28), searchBox.Top + sc(4), searchBox.Right - sc(4), searchBox.Bottom - sc(4)}
		drawButton(hdc, CtrlHangarSearchClear, clrRect, "✕", "danger")
	}

	btnNew := RECT{searchBox.Right + sc(8), sc(68), searchBox.Right + sc(150), sc(100)}
	btnSync := RECT{btnNew.Right + sc(6), sc(68), btnNew.Right + sc(145), sc(100)}
	btnReload := RECT{btnSync.Right + sc(6), sc(68), btnSync.Right + sc(105), sc(100)}
	drawButton(hdc, CtrlBtnNewDash, btnNew, "+ EIGENES SCHIFF", "amber")
	drawButton(hdc, CtrlBtnSyncSCBindings, btnSync, "SC-TASTEN LESEN", "cyan")
	drawButton(hdc, CtrlBtnReloadDash, btnReload, "⟳ UPDATE", "secondary")

	// Row 2: Category Filter Tabs (★ MEIN HANGAR | ALLE SCHIFFE | KAMPF | MINING | SALVAGE | FRACHT | EXPLORATION)
	fy := sc(106)
	fh := sc(30)
	filters := []struct {
		id    int
		key   string
		label string
	}{
		{CtrlHangarFilterFav, "hangar", "★ MEIN HANGAR"},
		{CtrlHangarFilterAll, "all", "ALLE SCHIFFE (100+)"},
		{CtrlHangarFilterCombat, "kampf", "KAMPF"},
		{CtrlHangarFilterMining, "mining", "MINING"},
		{CtrlHangarFilterSalvage, "salvage", "SALVAGE"},
		{CtrlHangarFilterCargo, "cargo", "FRACHT"},
		{CtrlHangarFilterExplore, "exploration", "EXPLORATION"},
	}
	fx := cx
	for _, f := range filters {
		fw := sc(96)
		if f.key == "all" {
			fw = sc(142)
		} else if f.key == "hangar" {
			fw = sc(122)
		}
		style := "secondary"
		if app.HangarFilter == f.key {
			if f.key == "hangar" {
				style = "amber"
			} else {
				style = "cyan"
			}
		}
		drawButton(hdc, f.id, RECT{fx, fy, fx + fw, fy + fh}, f.label, style)
		fx += fw + sc(6)
	}

	// Row 3: Quick Subsystem Adder for Active Ship
	subStrip := RECT{cx, sc(142), cx + cw, sc(176)}
	drawShardBox(hdc, subStrip, 6, colPanel, colLine)
	drawText(hdc, "UNTERPUNKT ZUM AKTIVEN SCHIFF (+):", RECT{subStrip.Left + sc(12), subStrip.Top, subStrip.Left + sc(225), subStrip.Bottom}, app.FontSmall, colAmber, DT_LEFT|DT_VCENTER|DT_SINGLELINE)
	sbX := subStrip.Left + sc(230)
	sbW := sc(92)
	subBtns := []struct {
		id    int
		label string
	}{
		{CtrlAddSubMining, "+ MINING"},
		{CtrlAddSubSalvage, "+ SALVAGE"},
		{CtrlAddSubCombat, "+ KAMPF"},
		{CtrlAddSubCargo, "+ FRACHT"},
		{CtrlAddSubPower, "+ ENERGIE"},
	}
	for _, sb := range subBtns {
		drawButton(hdc, sb.id, RECT{sbX, subStrip.Top + sc(4), sbX + sbW, subStrip.Bottom - sc(4)}, sb.label, "cyan")
		sbX += sbW + sc(6)
	}

	// Filtered Ship Cards List with Depicted Ship Silhouette & Pagination
	filtered := getFilteredHangarShips()
	y := sc(184)
	rowH := sc(68)
	availH := h - y - sc(44)
	maxRows := int(availH / (rowH + sc(6)))
	if maxRows < 1 {
		maxRows = 1
	}

	if app.HangarScrollIdx >= len(filtered) {
		app.HangarScrollIdx = 0
	}
	if app.HangarScrollIdx < 0 {
		app.HangarScrollIdx = 0
	}

	// Pagination controls on right side of subStrip
	totalPages := (len(filtered) + maxRows - 1) / maxRows
	if totalPages < 1 {
		totalPages = 1
	}
	curPage := (app.HangarScrollIdx / maxRows) + 1
	pageInfoRect := RECT{subStrip.Right - sc(215), subStrip.Top, subStrip.Right - sc(82), subStrip.Bottom}
	drawText(hdc, fmt.Sprintf("%d Schiffe (S. %d/%d)", len(filtered), curPage, totalPages), pageInfoRect, app.FontSmall, colCyan, DT_RIGHT|DT_VCENTER|DT_SINGLELINE)
	drawButton(hdc, CtrlHangarPagePrev, RECT{subStrip.Right - sc(76), subStrip.Top + sc(4), subStrip.Right - sc(44), subStrip.Bottom - sc(4)}, "▲", "secondary")
	drawButton(hdc, CtrlHangarPageNext, RECT{subStrip.Right - sc(38), subStrip.Top + sc(4), subStrip.Right - sc(6), subStrip.Bottom - sc(4)}, "▼", "secondary")

	if len(filtered) == 0 {
		drawText(hdc, "Keine passenden Schiffe gefunden. Klicke auf 'ALLE SCHIFFE (100+)' oder lösche den Suchbegriff.",
			RECT{cx, y + sc(30), cx + cw, y + sc(70)}, app.FontBody, colMuted, DT_CENTER|DT_SINGLELINE)
		return
	}

	for rIdx := 0; rIdx < maxRows; rIdx++ {
		fPos := app.HangarScrollIdx + rIdx
		if fPos >= len(filtered) {
			break
		}
		i := filtered[fPos]
		d := app.Dashboards[i]

		card := RECT{cx, y, cx + cw, y + rowH}
		isActive := d.ID == app.ActiveDash
		border := colLine
		fill := colPanel
		if isActive {
			border = colCyan
			fill = colPanelAct
		} else if d.Favorite {
			border = rgb(0x6e, 0x4d, 0x12)
		}
		drawShardBox(hdc, card, 10, fill, border)
		if isActive {
			fillRect(hdc, RECT{card.Left, card.Top + sc(8), card.Left + sc(4), card.Bottom - sc(10)}, colCyan)
		}

		// Left Ship Blueprint Silhouette Preview Box
		artBox := RECT{card.Left + sc(10), card.Top + sc(7), card.Left + sc(96), card.Bottom - sc(7)}
		drawShardBox(hdc, artBox, 6, rgb(0x06, 0x11, 0x1b), colLineHi)
		silCol := colCyan
		if d.Manufacturer == "DRAKE" || d.Manufacturer == "MISC" || d.Manufacturer == "ARGO" {
			silCol = colAmber
		}
		drawShipSilhouette(hdc, artBox, d.HullType, silCol, rgb(0x0c, 0x24, 0x34))

		// Ship Title & Subsystem Tags
		shipName := d.Ship
		if shipName == "" {
			shipName = d.Title
		}
		drawText(hdc, strings.ToUpper(shipName), RECT{artBox.Right + sc(12), card.Top + sc(8), card.Right - sc(380), card.Top + sc(32)}, app.FontTitle, colText, DT_LEFT|DT_SINGLELINE)

		var pageNames []string
		for _, p := range d.Pages {
			pageNames = append(pageNames, p.Label)
		}
		subList := strings.Join(pageNames, " · ")
		meta := fmt.Sprintf("[%s]  %s   •   Subsysteme (%d): %s", d.Manufacturer, d.Role, d.PageCount, subList)
		drawText(hdc, meta, RECT{artBox.Right + sc(12), card.Top + sc(34), card.Right - sc(375), card.Bottom - sc(8)}, app.FontSmall, silCol, DT_LEFT|DT_SINGLELINE)

		// Action Buttons on Right of Ship Card: ★ FAVORIT | ► ÖFFNEN | ⚙ BAUKASTEN | KOPIE
		bx := card.Right - sc(12)
		if d.Source == "local" {
			btnDel := RECT{bx - sc(68), card.Top + sc(16), bx, card.Bottom - sc(16)}
			drawButton(hdc, CtrlDashDelBase+i, btnDel, "RESET", "danger")
			bx = btnDel.Left - sc(6)
		}

		btnEdit := RECT{bx - sc(98), card.Top + sc(16), bx, card.Bottom - sc(16)}
		drawButton(hdc, CtrlDashEditBase+i, btnEdit, "⚙ ANPASSEN", "cyan")
		bx = btnEdit.Left - sc(6)

		btnUse := RECT{bx - sc(115), card.Top + sc(16), bx, card.Bottom - sc(16)}
		if isActive {
			drawButton(hdc, CtrlDashUseLiveBase+i, btnUse, "● IM COCKPIT", "green")
		} else {
			drawButton(hdc, CtrlDashUseLiveBase+i, btnUse, "► ÖFFNEN", "cyan")
		}
		bx = btnUse.Left - sc(6)

		btnFav := RECT{bx - sc(94), card.Top + sc(16), bx, card.Bottom - sc(16)}
		if d.Favorite {
			drawButton(hdc, CtrlDashFavBase+i, btnFav, "★ HANGAR", "amber")
		} else {
			drawButton(hdc, CtrlDashFavBase+i, btnFav, "☆ MERKEN", "secondary")
		}

		y += rowH + sc(6)
	}
}

// ============================================================================
// TAB 3: TABLET / WLAN SERVER HOST LINK
// ============================================================================
func paintTabHostLink(hdc uintptr, cx, cw, h int32) {
	drawText(hdc, "TABLET & WLAN SHARD SERVER", RECT{cx, sc(22), cx + cw - sc(115), sc(50)}, app.FontHero, colText, DT_LEFT|DT_SINGLELINE)
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
	drawText(hdc, "SYSTEM & EINSTELLUNGEN", RECT{cx, sc(22), cx + cw - sc(115), sc(50)}, app.FontHero, colText, DT_LEFT|DT_SINGLELINE)
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

	// 0. 1-Tap Hangar Quick Bar Ship Switcher Chip Click
	if id >= CtrlLiveShipChipBase && id < CtrlLiveShipChipBase+500 {
		idx := id - CtrlLiveShipChipBase
		if idx >= 0 && idx < len(app.Dashboards) {
			setActiveDashboard(app.Dashboards[idx].ID)
			app.LivePageIdx = 0
		}
		pInvalidateRect.Call(app.Hwnd, 0, 0)
		return
	}

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

	// 3. Hangar Favorite Toggle (★)
	if id >= CtrlDashFavBase && id < CtrlDashFavBase+500 {
		idx := id - CtrlDashFavBase
		if idx >= 0 && idx < len(app.Dashboards) {
			isFav := toggleDashboardFavorite(app.Dashboards[idx].ID)
			if isFav {
				app.ToastMsg = fmt.Sprintf("★ '%s' wurde zu deinem Hangar hinzugefügt!", app.Dashboards[idx].Ship)
			} else {
				app.ToastMsg = fmt.Sprintf("'%s' aus dem Hangar entfernt.", app.Dashboards[idx].Ship)
			}
		}
		pInvalidateRect.Call(app.Hwnd, 0, 0)
		return
	}

	// 4. Hangar Ship List Actions (Open / Customize / Copy / Delete)
	if id >= CtrlDashUseLiveBase && id < CtrlDashUseLiveBase+500 {
		idx := id - CtrlDashUseLiveBase
		if idx < len(app.Dashboards) {
			setActiveDashboard(app.Dashboards[idx].ID)
			app.LivePageIdx = 0
			app.Tab = TabLiveCockpit
			app.ToastMsg = fmt.Sprintf("Schiff '%s' im Live-Cockpit geöffnet.", app.Dashboards[idx].Ship)
		}
		pInvalidateRect.Call(app.Hwnd, 0, 0)
		return
	}
	if id >= CtrlDashEditBase && id < CtrlDashEditBase+500 {
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
	if id >= CtrlDashDupBase && id < CtrlDashDupBase+500 {
		idx := id - CtrlDashDupBase
		if idx < len(app.Dashboards) {
			newID, err := duplicateDashboard(app.Dashboards[idx])
			if err != nil {
				app.ToastMsg = "Fehler: " + err.Error()
			} else {
				app.ToastMsg = fmt.Sprintf("Schiff-Kopie '%s.json' im Hangar erstellt.", newID)
			}
		}
		pInvalidateRect.Call(app.Hwnd, 0, 0)
		return
	}
	if id >= CtrlDashDelBase && id < CtrlDashDelBase+500 {
		idx := id - CtrlDashDelBase
		if idx < len(app.Dashboards) {
			d := app.Dashboards[idx]
			if d.Source == "local" && d.Path != "" {
				_ = os.Remove(d.Path)
				reloadDashboards()
				app.ToastMsg = fmt.Sprintf("Eigene Anpassung für '%s' zurückgesetzt.", d.Ship)
			}
		}
		pInvalidateRect.Call(app.Hwnd, 0, 0)
		return
	}

	switch id {
	case CtrlWinClose:
		stopServer()
		pPostQuitMessage.Call(0)
	case CtrlWinMax:
		toggleMaximizeWindow()
	case CtrlWinMin:
		pShowWindow.Call(app.Hwnd, SW_MINIMIZE)
	case CtrlInstSecondary:
		enablePortableOrInstalledMode()
		app.Mode = ModeLauncher
		app.Tab = TabLiveCockpit
		reloadDashboards()
		startServer()

	// Navigation Tabs
	case CtrlNavLiveCockpit:
		reloadDashboards()
		app.Tab = TabLiveCockpit
		app.ConfirmUninst = false
	case CtrlNavShards, CtrlLiveOpenHangar:
		reloadDashboards()
		app.Tab = TabShards
		app.ConfirmUninst = false
	case CtrlNavHostLink:
		app.Tab = TabHostLink
		app.ConfirmUninst = false
	case CtrlNavSettings:
		app.Tab = TabSettings
		app.ConfirmUninst = false

	// Hangar Filter & Search Controls
	case CtrlHangarSearchBox:
		app.ToastMsg = "Tippe einfach auf deiner Tastatur, um nach Schiffen, Herstellern oder Rollen zu suchen (Backspace löscht)."
	case CtrlHangarSearchClear:
		app.HangarSearch = ""
		app.HangarScrollIdx = 0
	case CtrlHangarFilterFav:
		app.HangarFilter = "hangar"
		app.HangarScrollIdx = 0
	case CtrlHangarFilterAll:
		app.HangarFilter = "all"
		app.HangarScrollIdx = 0
	case CtrlHangarFilterCombat:
		app.HangarFilter = "kampf"
		app.HangarScrollIdx = 0
	case CtrlHangarFilterMining:
		app.HangarFilter = "mining"
		app.HangarScrollIdx = 0
	case CtrlHangarFilterSalvage:
		app.HangarFilter = "salvage"
		app.HangarScrollIdx = 0
	case CtrlHangarFilterCargo:
		app.HangarFilter = "cargo"
		app.HangarScrollIdx = 0
	case CtrlHangarFilterExplore:
		app.HangarFilter = "exploration"
		app.HangarScrollIdx = 0
	case CtrlHangarPagePrev:
		app.HangarScrollIdx -= 5
		if app.HangarScrollIdx < 0 {
			app.HangarScrollIdx = 0
		}
	case CtrlHangarPageNext:
		filtered := getFilteredHangarShips()
		if app.HangarScrollIdx+5 < len(filtered) {
			app.HangarScrollIdx += 5
		}

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
	case CtrlLiveAlwaysOnTop:
		toggleAlwaysOnTop()
	case CtrlLiveGitPull:
		app.ToastMsg = "Lade neueste Version direkt von GitHub (Auto-Pull)..."
		go runGitHubLivePull(true)
	case CtrlLiveOpenEditor:
		if !app.ServerRunning {
			startServer()
		}
		url := fmt.Sprintf("http://127.0.0.1:%d/", app.Port)
		pShellExecuteW.Call(0, u16ptr("open"), u16ptr(url), 0, 0, SW_SHOWNORMAL)

	// 1-Click Subsystem Page Generators
	case CtrlAddSubMining:
		addSubsystemToActiveShip("mining")
	case CtrlAddSubSalvage:
		addSubsystemToActiveShip("salvage")
	case CtrlAddSubCombat:
		addSubsystemToActiveShip("kampf")
	case CtrlAddSubCargo:
		addSubsystemToActiveShip("cargo")
	case CtrlAddSubPower:
		addSubsystemToActiveShip("energie")

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

	// Hangar Manager Controls
	case CtrlBtnNewDash:
		newID, err := createNewDashboard()
		if err != nil {
			app.ToastMsg = "Fehler: " + err.Error()
		} else {
			app.ToastMsg = fmt.Sprintf("★ Neues Schiff '%s.json' im Hangar angelegt.", newID)
		}
	case CtrlBtnSyncSCBindings:
		app.ToastMsg = syncStarCitizenActionmaps()
	case CtrlBtnOpenDashFolder:
		_ = os.MkdirAll(app.UserDashDir, 0755)
		pShellExecuteW.Call(0, u16ptr("open"), u16ptr(app.UserDashDir), 0, 0, SW_SHOWNORMAL)
	case CtrlBtnReloadDash:
		app.ToastMsg = "Lade neueste Version & Schiffe von GitHub..."
		go runGitHubLivePull(true)

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

	case WM_SIZE:
		pInvalidateRect.Call(hwnd, 0, 0)
		return 0

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

		// Support interactive edge/corner window resizing when not maximized!
		if !isWindowMaximized() {
			var rc RECT
			pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
			w := rc.Right - rc.Left
			h := rc.Bottom - rc.Top
			b := sc(6)
			onLeft := pt.X < b
			onRight := pt.X >= w-b
			onTop := pt.Y < b
			onBottom := pt.Y >= h-b
			if onTop && onLeft {
				return HTTOPLEFT
			}
			if onTop && onRight {
				return HTTOPRIGHT
			}
			if onBottom && onLeft {
				return HTBOTTOMLEFT
			}
			if onBottom && onRight {
				return HTBOTTOMRIGHT
			}
			if onLeft {
				return HTLEFT
			}
			if onRight {
				return HTRIGHT
			}
			if onTop {
				return HTTOP
			}
			if onBottom {
				return HTBOTTOM
			}
		}

		for _, reg := range app.Regions {
			if ptInRect(pt.X, pt.Y, reg.Rect) {
				return HTCLIENT
			}
		}
		if pt.Y < sc(52) {
			return HTCAPTION
		}
		return HTCLIENT

	case WM_NCLBUTTONDBLCLK:
		if wParam == HTCAPTION {
			toggleMaximizeWindow()
			return 0
		}

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

	case WM_MOUSEWHEEL:
		if app.Mode == ModeLauncher && app.Tab == TabShards {
			delta := int16((wParam >> 16) & 0xFFFF)
			if delta < 0 {
				filtered := getFilteredHangarShips()
				if app.HangarScrollIdx+3 < len(filtered) {
					app.HangarScrollIdx += 3
					pInvalidateRect.Call(hwnd, 0, 0)
				}
			} else if delta > 0 {
				app.HangarScrollIdx -= 3
				if app.HangarScrollIdx < 0 {
					app.HangarScrollIdx = 0
				}
				pInvalidateRect.Call(hwnd, 0, 0)
			}
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

	case WM_CHAR:
		if app.Mode == ModeLauncher && app.Tab == TabShards {
			ch := rune(wParam)
			if ch == '\b' {
				if len(app.HangarSearch) > 0 {
					runes := []rune(app.HangarSearch)
					app.HangarSearch = string(runes[:len(runes)-1])
					app.HangarScrollIdx = 0
					pInvalidateRect.Call(hwnd, 0, 0)
				}
			} else if ch >= 32 && ch < 127 && len(app.HangarSearch) < 32 {
				app.HangarSearch += string(ch)
				if app.HangarFilter == "hangar" {
					app.HangarFilter = "all"
				}
				app.HangarScrollIdx = 0
				pInvalidateRect.Call(hwnd, 0, 0)
			}
			return 0
		}

	case WM_KEYDOWN:
		if wParam == VK_ESCAPE {
			if app.Mode == ModeLauncher && app.Tab == TabShards && app.HangarSearch != "" {
				app.HangarSearch = ""
				app.HangarScrollIdx = 0
				pInvalidateRect.Call(hwnd, 0, 0)
				return 0
			}
			stopServer()
			pPostQuitMessage.Call(0)
		} else if app.Mode == ModeLauncher && app.Tab == TabShards {
			if wParam == VK_NEXT || wParam == VK_DOWN {
				filtered := getFilteredHangarShips()
				if app.HangarScrollIdx+4 < len(filtered) {
					app.HangarScrollIdx += 4
					pInvalidateRect.Call(hwnd, 0, 0)
				}
			} else if wParam == VK_PRIOR || wParam == VK_UP {
				app.HangarScrollIdx -= 4
				if app.HangarScrollIdx < 0 {
					app.HangarScrollIdx = 0
				}
				pInvalidateRect.Call(hwnd, 0, 0)
			}
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

	app.Width = sc(1080)
	app.Height = sc(680)

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
		Style:     CS_HREDRAW | CS_VREDRAW | CS_DBLCLKS | CS_DROPSHADOW,
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
		uintptr(unsafe.Pointer(u16("SC Cockpit // GameGlass Hangar & Shard Hub"))),
		WS_POPUP|WS_VISIBLE|WS_SYSMENU|WS_MINIMIZEBOX|WS_MAXIMIZEBOX|WS_CLIPCHILDREN,
		uintptr(winX), uintptr(winY),
		uintptr(app.Width), uintptr(app.Height),
		0, 0, hInst, 0,
	)
	app.Hwnd = hwnd

	if app.Mode == ModeLauncher {
		reloadDashboards()
		startServer()
		go runGitHubLivePull(false)
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
