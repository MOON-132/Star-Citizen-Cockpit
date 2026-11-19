#!/usr/bin/env python3
"""SC Cockpit — lokaler Tastatur-Bridge für ein Tablet-Dashboard.

Nur die Python-Standardbibliothek. Auf dem Gaming-PC starten, im Browser
des Tablets die angezeigte Adresse öffnen. Tasten gehen per SendInput
(Scancode) an das fokussierte Fenster, also an Star Citizen.

Keine Pip-Pakete. Kein Account. Kein Internet nötig nach dem Start.
"""

from __future__ import annotations

import json
import socket
import sys
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

ROOT = Path(__file__).resolve().parent
VERSION = "0.1.0"

# Virtual-Key-Codes. Buchstaben werden auf Windows per MapVirtualKey in den
# Scancode des aktiven Tastaturlayouts übersetzt (wichtig für DE: Y/Z).
NAMED_VK = {
    "backspace": 0x08,
    "tab": 0x09,
    "enter": 0x0D,
    "return": 0x0D,
    "pause": 0x13,
    "capslock": 0x14,
    "escape": 0x1B,
    "esc": 0x1B,
    "space": 0x20,
    "pageup": 0x21,
    "pagedown": 0x22,
    "end": 0x23,
    "home": 0x24,
    "left": 0x25,
    "up": 0x26,
    "right": 0x27,
    "down": 0x28,
    "insert": 0x2D,
    "delete": 0x2E,
    "lwin": 0x5B,
    "rwin": 0x5C,
    "num0": 0x60,
    "num1": 0x61,
    "num2": 0x62,
    "num3": 0x63,
    "num4": 0x64,
    "num5": 0x65,
    "num6": 0x66,
    "num7": 0x67,
    "num8": 0x68,
    "num9": 0x69,
    "nummul": 0x6A,
    "numadd": 0x6B,
    "numsub": 0x6D,
    "numdot": 0x6E,
    "numdiv": 0x6F,
    "numenter": 0x0D,
    "f1": 0x70,
    "f2": 0x71,
    "f3": 0x72,
    "f4": 0x73,
    "f5": 0x74,
    "f6": 0x75,
    "f7": 0x76,
    "f8": 0x77,
    "f9": 0x78,
    "f10": 0x79,
    "f11": 0x7A,
    "f12": 0x7B,
    "lshift": 0xA0,
    "rshift": 0xA1,
    "lctrl": 0xA2,
    "rctrl": 0xA3,
    "lalt": 0xA4,
    "ralt": 0xA5,
}
for _i, _ch in enumerate("0123456789"):
    NAMED_VK[_ch] = 0x30 + _i
for _i, _ch in enumerate("abcdefghijklmnopqrstuvwxyz"):
    NAMED_VK[_ch] = 0x41 + _i

# Fallback-Scancodes (US Set 1), falls MapVirtualKey nichts liefert.
US_SCAN = {
    "escape": 0x01, "1": 0x02, "2": 0x03, "3": 0x04, "4": 0x05, "5": 0x06,
    "6": 0x07, "7": 0x08, "8": 0x09, "9": 0x0A, "0": 0x0B,
    "backspace": 0x0E, "tab": 0x0F,
    "q": 0x10, "w": 0x11, "e": 0x12, "r": 0x13, "t": 0x14, "y": 0x15,
    "u": 0x16, "i": 0x17, "o": 0x18, "p": 0x19,
    "enter": 0x1C, "return": 0x1C, "lctrl": 0x1D,
    "a": 0x1E, "s": 0x1F, "d": 0x20, "f": 0x21, "g": 0x22, "h": 0x23,
    "j": 0x24, "k": 0x25, "l": 0x26,
    "lshift": 0x2A, "z": 0x2C, "x": 0x2D, "c": 0x2E, "v": 0x2F, "b": 0x30,
    "n": 0x31, "m": 0x32, "rshift": 0x36, "lalt": 0x38, "space": 0x39,
    "f1": 0x3B, "f2": 0x3C, "f3": 0x3D, "f4": 0x3E, "f5": 0x3F, "f6": 0x40,
    "f7": 0x41, "f8": 0x42, "f9": 0x43, "f10": 0x44, "f11": 0x57, "f12": 0x58,
    "num7": 0x47, "num8": 0x48, "num9": 0x49, "numsub": 0x4A,
    "num4": 0x4B, "num5": 0x4C, "num6": 0x4D, "numadd": 0x4E,
    "num1": 0x4F, "num2": 0x50, "num3": 0x51, "num0": 0x52, "numdot": 0x53,
}

EXTENDED_NAMES = {
    "rctrl", "ralt", "rwin", "insert", "delete", "home", "end",
    "pageup", "pagedown", "left", "right", "up", "down",
    "numdiv", "numenter",
}

ALIASES = {
    "ctrl": "lctrl",
    "control": "lctrl",
    "lcontrol": "lctrl",
    "rcontrol": "rctrl",
    "shift": "lshift",
    "alt": "lalt",
    "rightalt": "ralt",
    "leftalt": "lalt",
    "altgr": "ralt",
    "win": "lwin",
    "windows": "lwin",
    "return": "enter",
    "esc": "escape",
    "spacebar": "space",
    "pgup": "pageup",
    "pgdn": "pagedown",
    "del": "delete",
    "ins": "insert",
    "arrowup": "up",
    "arrowdown": "down",
    "arrowleft": "left",
    "arrowright": "right",
    "mouseleft": "mouse1",
    "lmb": "mouse1",
    "leftclick": "mouse1",
    "mouse_left": "mouse1",
    "mouseright": "mouse2",
    "rmb": "mouse2",
    "rightclick": "mouse2",
    "middlemouse": "mouse3",
    "mmb": "mouse3",
    "middleclick": "mouse3",
    "wheelup": "wheelup",
    "wheeldown": "wheeldown",
}

MOUSE = {
    "mouse1": ("left",),
    "mouse2": ("right",),
    "mouse3": ("middle",),
    "wheelup": ("wheel", 120),
    "wheeldown": ("wheel", -120),
}

DRY_RUN = sys.platform != "win32"
user32 = None

if sys.platform == "win32":
    import ctypes
    from ctypes import wintypes

    user32 = ctypes.WinDLL("user32", use_last_error=True)
    ULONG_PTR = ctypes.c_size_t

    INPUT_MOUSE = 0
    INPUT_KEYBOARD = 1
    KEYEVENTF_EXTENDEDKEY = 0x0001
    KEYEVENTF_KEYUP = 0x0002
    KEYEVENTF_SCANCODE = 0x0008
    MOUSEEVENTF_LEFTDOWN = 0x0002
    MOUSEEVENTF_LEFTUP = 0x0004
    MOUSEEVENTF_RIGHTDOWN = 0x0008
    MOUSEEVENTF_RIGHTUP = 0x0010
    MOUSEEVENTF_MIDDLEDOWN = 0x0020
    MOUSEEVENTF_MIDDLEUP = 0x0040
    MOUSEEVENTF_WHEEL = 0x0800
    MAPVK_VK_TO_VSC_EX = 4

    class KEYBDINPUT(ctypes.Structure):
        _fields_ = (
            ("wVk", wintypes.WORD),
            ("wScan", wintypes.WORD),
            ("dwFlags", wintypes.DWORD),
            ("time", wintypes.DWORD),
            ("dwExtraInfo", ULONG_PTR),
        )

    class MOUSEINPUT(ctypes.Structure):
        _fields_ = (
            ("dx", wintypes.LONG),
            ("dy", wintypes.LONG),
            ("mouseData", wintypes.DWORD),
            ("dwFlags", wintypes.DWORD),
            ("time", wintypes.DWORD),
            ("dwExtraInfo", ULONG_PTR),
        )

    class HARDWAREINPUT(ctypes.Structure):
        _fields_ = (
            ("uMsg", wintypes.DWORD),
            ("wParamL", wintypes.WORD),
            ("wParamH", wintypes.WORD),
        )

    class INPUT_UNION(ctypes.Union):
        _fields_ = (
            ("ki", KEYBDINPUT),
            ("mi", MOUSEINPUT),
            ("hi", HARDWAREINPUT),
        )

    class INPUT(ctypes.Structure):
        _fields_ = (
            ("type", wintypes.DWORD),
            ("union", INPUT_UNION),
        )

    user32.SendInput.argtypes = (wintypes.UINT, ctypes.POINTER(INPUT), ctypes.c_int)
    user32.SendInput.restype = wintypes.UINT
    user32.MapVirtualKeyW.argtypes = (wintypes.UINT, wintypes.UINT)
    user32.MapVirtualKeyW.restype = wintypes.UINT
    user32.VkKeyScanW.argtypes = (wintypes.WCHAR,)
    user32.VkKeyScanW.restype = ctypes.c_short


def log(msg: str) -> None:
    print(msg, flush=True)


def load_json(path: Path) -> dict | None:
    if not path.is_file():
        return None
    with path.open(encoding="utf-8") as fh:
        return json.load(fh)


def load_config() -> dict:
    cfg = {"port": 8765, "token": "", "hold_timeout_s": 4}
    try:
        raw = load_json(ROOT / "config.local.json")
    except json.JSONDecodeError as exc:
        log(f"config.local.json ist kein gültiges JSON: {exc}")
        raw = None
    if isinstance(raw, dict):
        cfg.update({k: raw[k] for k in cfg if k in raw})
    cfg["port"] = int(cfg["port"])
    cfg["token"] = str(cfg.get("token") or "")
    cfg["hold_timeout_s"] = float(cfg["hold_timeout_s"])
    return cfg


def layout_path() -> tuple[Path, str]:
    local = ROOT / "layout.local.json"
    if local.is_file():
        return local, "local"
    return ROOT / "layout.json", "default"


def load_layout() -> tuple[dict, str]:
    path, source = layout_path()
    data = load_json(path)
    if not isinstance(data, dict) or "pages" not in data:
        raise SystemExit(f"Layout fehlt oder ist ungültig: {path}")
    return data, source


def normalize_key(name: str) -> str:
    key = str(name).strip().lower().replace(" ", "")
    return ALIASES.get(key, key)


def known_key(name: str) -> bool:
    key = normalize_key(name)
    if key in MOUSE or key in NAMED_VK:
        return True
    # Einzelnes Zeichen (z. B. ü) wird auf Windows über VkKeyScan aufgelöst.
    return len(key) == 1


def _vk_for(key: str) -> int | None:
    if key in NAMED_VK:
        return NAMED_VK[key]
    if sys.platform == "win32" and len(key) == 1 and user32 is not None:
        res = user32.VkKeyScanW(key)
        if res == -1:
            return None
        return res & 0xFF
    return None


def describe_key(name: str) -> str:
    key = normalize_key(name)
    if key in MOUSE:
        return f"mouse:{key}"
    vk = _vk_for(key)
    if sys.platform == "win32" and vk is not None and user32 is not None:
        mapped = user32.MapVirtualKeyW(vk, MAPVK_VK_TO_VSC_EX)
        scan = mapped & 0xFF
        extended = bool(mapped & 0xFF00) or key in EXTENDED_NAMES
        return f"vk={vk:#x} scan={scan:#x} ext={int(extended)}"
    return f"name:{key}"


def emit(name: str, up: bool) -> None:
    key = normalize_key(name)
    if DRY_RUN or user32 is None:
        log(f"    {'up  ' if up else 'down'} {key}")
        return
    if key in MOUSE:
        kind = MOUSE[key]
        if kind[0] == "wheel":
            flags = MOUSEEVENTF_WHEEL
            data = kind[1]
            if up:
                return
        else:
            data = 0
            table = {
                "left": (MOUSEEVENTF_LEFTDOWN, MOUSEEVENTF_LEFTUP),
                "right": (MOUSEEVENTF_RIGHTDOWN, MOUSEEVENTF_RIGHTUP),
                "middle": (MOUSEEVENTF_MIDDLEDOWN, MOUSEEVENTF_MIDDLEUP),
            }
            flags = table[kind[0]][1 if up else 0]
        inp = INPUT(type=INPUT_MOUSE)
        inp.union.mi = MOUSEINPUT(0, 0, data, flags, 0, 0)
    else:
        vk = _vk_for(key)
        if vk is None:
            raise ValueError(f"unbekannte Taste: {name}")
        mapped = user32.MapVirtualKeyW(vk, MAPVK_VK_TO_VSC_EX)
        scan = mapped & 0xFF
        extended = bool(mapped & 0xFF00) or key in EXTENDED_NAMES
        if scan == 0:
            scan = US_SCAN.get(key, 0)
        if scan == 0:
            raise ValueError(f"kein Scancode für: {name}")
        flags = KEYEVENTF_SCANCODE
        if extended:
            flags |= KEYEVENTF_EXTENDEDKEY
        if up:
            flags |= KEYEVENTF_KEYUP
        inp = INPUT(type=INPUT_KEYBOARD)
        inp.union.ki = KEYBDINPUT(0, scan, flags, 0, 0)
    sent = user32.SendInput(1, ctypes.byref(inp), ctypes.sizeof(INPUT))
    if sent != 1:
        err = ctypes.get_last_error()
        raise OSError(f"SendInput fehlgeschlagen ({err})")


class Board:
    """Gedrückte Tasten plus kurze Halte-Lizenz, damit ein totes Tablet nicht hängen bleibt."""

    def __init__(self, lease_s: float) -> None:
        self.lease_s = lease_s
        self.lock = threading.Lock()
        self.down: set[str] = set()
        self.lease: dict[str, float] = {}

    def tap(self, keys: list[str], gap_ms: int) -> None:
        self._press(keys)
        time.sleep(max(gap_ms, 1) / 1000)
        self._release(keys)

    def hold_for(self, keys: list[str], hold_ms: int) -> None:
        self.finger_down(keys)
        time.sleep(max(hold_ms, 1) / 1000)
        self.finger_up(keys)

    def finger_down(self, keys: list[str]) -> None:
        expiry = time.monotonic() + self.lease_s
        with self.lock:
            for key in keys:
                self.lease[key] = expiry
        self._press(keys)

    def finger_up(self, keys: list[str]) -> None:
        with self.lock:
            for key in keys:
                self.lease.pop(key, None)
        self._release(keys)

    def release_all(self) -> list[str]:
        with self.lock:
            keys = list(self.down)
            self.lease.clear()
            self.down.clear()
        for key in reversed(keys):
            emit(key, up=True)
        return keys

    def sweep(self) -> None:
        now = time.monotonic()
        with self.lock:
            expired = [k for k, exp in self.lease.items() if exp <= now]
            for key in expired:
                self.lease.pop(key, None)
        if expired:
            self._release(expired)

    def _press(self, keys: list[str]) -> None:
        with self.lock:
            for key in keys:
                if key not in self.down:
                    emit(key, up=False)
                    self.down.add(key)

    def _release(self, keys: list[str]) -> None:
        with self.lock:
            for key in reversed(keys):
                if key in self.lease or key not in self.down:
                    continue
                emit(key, up=True)
                self.down.discard(key)


def iter_buttons(layout: dict):
    for page in layout.get("pages", []):
        for group in page.get("groups", []):
            for button in group.get("buttons", []):
                yield page, group, button


def validate_layout(layout: dict) -> list[str]:
    errors = []
    count = 0
    for page, _group, button in iter_buttons(layout):
        count += 1
        label = button.get("label", button.get("id", "?"))
        keys = button.get("keys")
        if not isinstance(keys, list) or not keys:
            errors.append(f"{page.get('id')}/{label}: keys fehlt")
            continue
        for key in keys:
            if not known_key(str(key)):
                errors.append(f"{page.get('id')}/{label}: unbekannte Taste {key!r}")
        mode = button.get("mode", "tap")
        if mode not in {"tap", "toggle", "hold"}:
            errors.append(f"{page.get('id')}/{label}: mode {mode!r} ungültig")
    if count == 0:
        errors.append("Layout enthält keine Buttons")
    return errors


def lan_ips() -> list[str]:
    ips: list[str] = []
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.connect(("8.8.8.8", 80))
        ips.append(sock.getsockname()[0])
        sock.close()
    except OSError:
        pass
    try:
        for info in socket.getaddrinfo(socket.gethostname(), None, socket.AF_INET):
            ip = info[4][0]
            if not ip.startswith("127.") and ip not in ips:
                ips.append(ip)
    except OSError:
        pass
    return ips or ["127.0.0.1"]


def is_admin() -> bool | None:
    if sys.platform != "win32":
        return None
    try:
        return bool(ctypes.windll.shell32.IsUserAnAdmin())
    except Exception:
        return None


CONFIG = load_config()
BOARD = Board(CONFIG["hold_timeout_s"])


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, fmt: str, *args) -> None:
        return

    def _authorized(self) -> bool:
        token = CONFIG["token"]
        if not token:
            return True
        got = self.headers.get("X-Token", "")
        if len(got) != len(token):
            return False
        diff = 0
        for a, b in zip(got, token):
            diff |= ord(a) ^ ord(b)
        return diff == 0

    def _json(self, code: int, payload: dict) -> None:
        body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Cache-Control", "no-store")
        self.end_headers()
        self.wfile.write(body)

    def _bytes(self, code: int, body: bytes, content_type: str) -> None:
        self.send_response(code)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Cache-Control", "no-store")
        self.end_headers()
        self.wfile.write(body)

    def _read_json(self) -> dict:
        length = int(self.headers.get("Content-Length", "0") or 0)
        if length <= 0 or length > 8192:
            raise ValueError("ungültiger Body")
        raw = self.rfile.read(length)
        data = json.loads(raw.decode("utf-8"))
        if not isinstance(data, dict):
            raise ValueError("JSON-Objekt erwartet")
        return data

    def do_GET(self) -> None:  # noqa: N802
        path = self.path.split("?", 1)[0]
        if path in {"/", "/index.html"}:
            file = ROOT / "index.html"
            if not file.is_file():
                self._json(500, {"error": "index.html fehlt"})
                return
            self._bytes(200, file.read_bytes(), "text/html; charset=utf-8")
            return
        if path == "/favicon.ico":
            self._bytes(204, b"", "image/x-icon")
            return
        if not self._authorized():
            self._json(401, {"error": "token"})
            return
        if path == "/api/health":
            _layout, source = layout_path()
            self._json(200, {
                "ok": True,
                "version": VERSION,
                "dry_run": DRY_RUN,
                "layout": source,
                "held": sorted(BOARD.down),
            })
            return
        if path == "/api/layout":
            try:
                layout, source = load_layout()
            except (json.JSONDecodeError, OSError, SystemExit) as exc:
                self._json(500, {"error": str(exc)})
                return
            layout = dict(layout)
            layout["_source"] = source
            self._json(200, layout)
            return
        self._json(404, {"error": "nicht gefunden"})

    def do_POST(self) -> None:  # noqa: N802
        path = self.path.split("?", 1)[0]
        if not self._authorized():
            self._json(401, {"error": "token"})
            return
        if path == "/api/release":
            released = BOARD.release_all()
            log(f"release  {released or '-'}")
            self._json(200, {"ok": True, "released": released})
            return
        if path != "/api/press":
            self._json(404, {"error": "nicht gefunden"})
            return
        try:
            data = self._read_json()
            keys = [normalize_key(k) for k in data.get("keys", [])]
            mode = str(data.get("mode") or "tap")
            if mode == "toggle":
                mode = "tap"
            if mode not in {"tap", "hold", "down", "up"}:
                raise ValueError("mode ungültig")
            if not keys or len(keys) > 4:
                raise ValueError("1 bis 4 Tasten")
            for key in keys:
                if not known_key(key):
                    raise ValueError(f"unbekannte Taste: {key}")
            hold_ms = int(data.get("hold_ms") or (400 if mode == "hold" else 45))
            hold_ms = max(1, min(hold_ms, 10000))
        except (ValueError, TypeError, json.JSONDecodeError) as exc:
            self._json(400, {"error": str(exc)})
            return
        label = str(data.get("id") or "+".join(keys))
        try:
            if mode == "down":
                BOARD.finger_down(keys)
            elif mode == "up":
                BOARD.finger_up(keys)
            elif mode == "hold":
                BOARD.hold_for(keys, hold_ms)
            else:
                BOARD.tap(keys, hold_ms)
        except (ValueError, OSError) as exc:
            log(f"FEHLER {mode} {label}: {exc}")
            self._json(500, {"error": str(exc)})
            return
        if mode != "down":
            log(f"{mode:<6} {label}  {'+'.join(keys)}")
        self._json(200, {"ok": True})


def serve(host: str = "0.0.0.0") -> None:
    layout, source = load_layout()
    errors = validate_layout(layout)
    if errors:
        for err in errors:
            log(f"Layout: {err}")
        raise SystemExit(1)
    buttons = sum(1 for _ in iter_buttons(layout))
    server = ThreadingHTTPServer((host, CONFIG["port"]), Handler)
    ips = lan_ips()
    log("")
    log(f"SC Cockpit {VERSION}")
    log(f"Layout: {buttons} Buttons ({source})")
    if DRY_RUN:
        log("Trockenlauf: kein Windows, Tasten werden nur angezeigt.")
    admin = is_admin()
    if admin is False:
        log("Hinweis: nicht als Administrator. Wenn Star Citizen als Admin läuft,")
        log("         start.bat per Rechtsklick 'Als Administrator ausführen'.")
    if CONFIG["token"]:
        log("Token ist aktiv.")
    else:
        log("Kein Token. Nur im eigenen WLAN benutzen.")
    log("")
    for ip in ips:
        log(f"  Tablet:  http://{ip}:{CONFIG['port']}/")
    log("")
    log("Strg+C beendet den Server. Dieses Fenster offen lassen.")
    log("")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        log("Beendet.")
    finally:
        BOARD.release_all()
        server.server_close()


def self_test() -> int:
    layout, source = load_layout()
    errors = validate_layout(layout)
    if errors:
        for err in errors:
            log(err)
        return 1
    BOARD.tap(["n"], 15)
    BOARD.finger_down(["lshift"])
    BOARD.finger_down(["lshift"])
    BOARD.finger_up(["lshift"])
    BOARD.release_all()
    server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    port = server.server_address[1]
    try:
        import urllib.request

        with urllib.request.urlopen(f"http://127.0.0.1:{port}/api/health") as res:
            health = json.loads(res.read().decode())
        if not health.get("ok"):
            raise SystemExit("health nicht ok")
        req = urllib.request.Request(
            f"http://127.0.0.1:{port}/api/press",
            data=json.dumps({"keys": ["n"], "mode": "tap", "id": "test"}).encode(),
            headers={"Content-Type": "application/json"},
        )
        with urllib.request.urlopen(req) as res:
            body = json.loads(res.read().decode())
        if not body.get("ok"):
            raise SystemExit("press nicht ok")
        with urllib.request.urlopen(f"http://127.0.0.1:{port}/") as res:
            html = res.read().decode()
        if "SC Cockpit" not in html:
            raise SystemExit("index.html unvollständig")
    finally:
        server.shutdown()
        server.server_close()
    log(f"self-test ok ({source}, dry_run={DRY_RUN})")
    return 0


def main() -> None:
    try:
        sys.stdout.reconfigure(encoding="utf-8")
    except Exception:
        pass
    if "--self-test" in sys.argv:
        raise SystemExit(self_test())
    if "--help" in sys.argv:
        log("python server.py            Server starten")
        log("python server.py --self-test  Layout und HTTP prüfen")
        return
    sweeper = threading.Thread(target=_sweep_loop, daemon=True)
    sweeper.start()
    serve()


def _sweep_loop() -> None:
    while True:
        time.sleep(0.4)
        try:
            BOARD.sweep()
        except Exception as exc:
            log(f"sweep: {exc}")


if __name__ == "__main__":
    main()
