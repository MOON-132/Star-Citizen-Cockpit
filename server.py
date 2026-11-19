#!/usr/bin/env python3
"""SC Cockpit — lokaler Tastatur-Bridge für ein Tablet-Dashboard.

Nur die Python-Standardbibliothek. Auf dem Gaming-PC starten, im Browser
jedes Endgeräts die angezeigte Adresse öffnen. Tasten gehen per SendInput
(Scancode) an das fokussierte Fenster, also an Star Citizen.

Dashboards sind JSON-Dateien. Format und Regeln: docs/DASHBOARD.md.
Mitgelieferte liegen in dashboards/, eigene in dashboards/local/
oder im Benutzerordner des Installers.


"""

from __future__ import annotations

import json
import os
import re
import socket
import sys
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import unquote

APP_ROOT = Path(__file__).resolve().parent
VERSION = "0.2.0"
ID_RE = re.compile(r"^[a-z0-9][a-z0-9-]{0,40}$")
RESERVED_IDS = {"local", "state"}
COLORS = {"amber", "cyan", "red", "blue", "green", "slate"}

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
    candidates = []
    home = os.environ.get("SC_COCKPIT_HOME", "").strip()
    if home:
        candidates.append(Path(home) / "config.local.json")
    candidates.append(APP_ROOT / "config.local.json")
    for path in candidates:
        try:
            raw = load_json(path)
        except json.JSONDecodeError as exc:
            log(f"{path.name} ist kein gültiges JSON: {exc}")
            continue
        if isinstance(raw, dict):
            cfg.update({k: raw[k] for k in cfg if k in raw})
            break
    cfg["port"] = int(cfg["port"])
    cfg["token"] = str(cfg.get("token") or "")
    cfg["hold_timeout_s"] = float(cfg["hold_timeout_s"])
    return cfg


def user_root() -> Path:
    raw = os.environ.get("SC_COCKPIT_HOME", "").strip()
    if raw:
        path = Path(raw)
        path.mkdir(parents=True, exist_ok=True)
        return path
    return APP_ROOT


def builtin_dir() -> Path:
    return APP_ROOT / "dashboards"


def local_dir() -> Path:
    if user_root().resolve() == APP_ROOT.resolve():
        path = APP_ROOT / "dashboards" / "local"
    else:
        path = user_root() / "dashboards"
    path.mkdir(parents=True, exist_ok=True)
    return path


def state_path() -> Path:
    if user_root().resolve() == APP_ROOT.resolve():
        return local_dir() / "state.json"
    return user_root() / "state.json"


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


def slug(text: str) -> str:
    out = []
    for ch in str(text).strip().lower():
        if ch.isascii() and ch.isalnum():
            out.append(ch)
        elif ch in "-_ " and out and out[-1] != "-":
            out.append("-")
    return "".join(out).strip("-")[:40]


def _read_dashboard(path: Path, source: str, items: list, index: dict) -> None:
    try:
        data = load_json(path)
    except json.JSONDecodeError as exc:
        log(f"{path.name}: {exc}")
        return
    if not isinstance(data, dict) or not isinstance(data.get("pages"), list):
        return
    dash_id = str(data.get("id") or path.stem)
    if dash_id in RESERVED_IDS or not ID_RE.fullmatch(dash_id):
        log(f"Dashboard-Id ungültig: {dash_id}")
        return
    entry = {
        "id": dash_id,
        "title": str(data.get("title") or dash_id)[:40],
        "source": source,
        "path": path,
        "data": data,
    }
    if dash_id in index:
        items[index[dash_id]] = entry
    else:
        index[dash_id] = len(items)
        items.append(entry)


def catalog() -> list[dict]:
    items: list[dict] = []
    index: dict[str, int] = {}
    folder = builtin_dir()
    if folder.is_dir():
        for path in sorted(folder.glob("*.json")):
            _read_dashboard(path, "builtin", items, index)
    legacy = APP_ROOT / "layout.json"
    if legacy.is_file():
        _read_dashboard(legacy, "builtin", items, index)
    legacy_local = APP_ROOT / "layout.local.json"
    if legacy_local.is_file():
        _read_dashboard(legacy_local, "local", items, index)
    for path in sorted(local_dir().glob("*.json")):
        _read_dashboard(path, "local", items, index)
    return items


def active_id(items: list[dict] | None = None) -> str:
    items = catalog() if items is None else items
    ids = [item["id"] for item in items]
    try:
        raw = load_json(state_path())
    except (json.JSONDecodeError, OSError):
        raw = None
    want = raw.get("active") if isinstance(raw, dict) else None
    if want in ids:
        return str(want)
    if "standard" in ids:
        return "standard"
    return ids[0] if ids else ""


def set_active(dash_id: str) -> None:
    path = state_path()
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps({"active": dash_id}, indent=2) + "\n", encoding="utf-8")


def find_dashboard(dash_id: str) -> dict | None:
    for item in catalog():
        if item["id"] == dash_id:
            return item
    return None


def clean_dashboard(data: dict, dash_id: str) -> dict:
    if not isinstance(data, dict):
        raise ValueError("Dashboard muss ein Objekt sein")
    pages_in = data.get("pages")
    if not isinstance(pages_in, list) or not pages_in:
        raise ValueError("mindestens eine Seite")
    if len(pages_in) > 12:
        raise ValueError("höchstens 12 Seiten")
    pages = []
    seen_pages: set[str] = set()
    seen_buttons: set[str] = set()
    for i, page in enumerate(pages_in):
        if not isinstance(page, dict):
            raise ValueError("Seite ungültig")
        pid = slug(page.get("id") or f"seite-{i + 1}") or f"seite-{i + 1}"
        base = pid
        n = 2
        while pid in seen_pages:
            pid = f"{base}-{n}"[:40]
            n += 1
        seen_pages.add(pid)
        groups = []
        for group in page.get("groups") or []:
            if not isinstance(group, dict):
                continue
            buttons = []
            for button in group.get("buttons") or []:
                if not isinstance(button, dict):
                    continue
                keys = button.get("keys")
                if not isinstance(keys, list) or not 1 <= len(keys) <= 4:
                    raise ValueError(f"Tasten fehlen bei {button.get('label') or 'Button'}")
                norm = [normalize_key(str(key)) for key in keys]
                for key in norm:
                    if not known_key(key):
                        raise ValueError(f"unbekannte Taste: {key}")
                mode = button.get("mode") or "tap"
                if mode not in {"tap", "toggle", "hold"}:
                    raise ValueError("mode ungültig")
                color = button.get("color") or "slate"
                if color not in COLORS:
                    color = "slate"
                bid = slug(str(button.get("id") or button.get("label") or "button")) or "button"
                base_id = bid
                n = 2
                while bid in seen_buttons:
                    bid = f"{base_id}-{n}"[:40]
                    n += 1
                seen_buttons.add(bid)
                buttons.append({
                    "id": bid,
                    "label": str(button.get("label") or bid)[:40],
                    "hint": str(button.get("hint") or "")[:60],
                    "keys": norm,
                    "mode": mode,
                    "color": color,
                    "wide": bool(button.get("wide")),
                    "danger": bool(button.get("danger")),
                })
            if buttons:
                groups.append({
                    "title": str(group.get("title") or "Steuerung")[:40],
                    "buttons": buttons,
                })
        if not groups:
            raise ValueError(f"Seite {pid} hat keine Buttons")
        pages.append({
            "id": pid,
            "label": str(page.get("label") or pid)[:24],
            "groups": groups,
        })
    return {"id": dash_id, "title": str(data.get("title") or dash_id)[:40], "pages": pages}


def save_dashboard(dash_id: str, data: dict) -> dict:
    if dash_id in RESERVED_IDS or not ID_RE.fullmatch(dash_id):
        raise ValueError("Id ungültig. Kleinbuchstaben, Zahlen, Bindestrich.")
    cleaned = clean_dashboard(data, dash_id)
    path = local_dir() / f"{dash_id}.json"
    tmp = path.with_suffix(".tmp")
    tmp.write_text(json.dumps(cleaned, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    tmp.replace(path)
    return cleaned


def delete_dashboard(dash_id: str) -> bool:
    path = local_dir() / f"{dash_id}.json"
    if not path.is_file():
        return False
    path.unlink()
    return True


def public_dashboard(item: dict) -> dict:
    data = dict(item["data"])
    data["id"] = item["id"]
    data["_source"] = item["source"]
    data["_title"] = item["title"]
    return data


def lan_ips() -> list[str]:
    ips: list[str] = []
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.connect(("8.8.8.8", 80))
        primary = sock.getsockname()[0]
        sock.close()
        if not primary.startswith(("127.", "169.254.")):
            ips.append(primary)
    except OSError:
        pass
    try:
        for info in socket.getaddrinfo(socket.gethostname(), None, socket.AF_INET):
            ip = info[4][0]
            if not ip.startswith(("127.", "169.254.")) and ip not in ips:
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

    def _read_json(self, limit: int = 8192) -> dict:
        length = int(self.headers.get("Content-Length", "0") or 0)
        if length <= 0 or length > limit:
            raise ValueError("ungültiger Body")
        raw = self.rfile.read(length)
        data = json.loads(raw.decode("utf-8"))
        if not isinstance(data, dict):
            raise ValueError("JSON-Objekt erwartet")
        return data

    def _dash_id(self, path: str) -> str | None:
        prefix = "/api/dashboard/"
        if not path.startswith(prefix):
            return None
        dash_id = unquote(path[len(prefix):]).strip().strip("/")
        if not ID_RE.fullmatch(dash_id) or dash_id in RESERVED_IDS:
            return None
        return dash_id

    def do_GET(self) -> None:  # noqa: N802
        path = self.path.split("?", 1)[0]
        if path in {"/", "/index.html"}:
            file = APP_ROOT / "index.html"
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
            items = catalog()
            self._json(200, {
                "ok": True,
                "version": VERSION,
                "dry_run": DRY_RUN,
                "active": active_id(items),
                "dashboards": len(items),
                "held": sorted(BOARD.down),
            })
            return
        if path == "/api/dashboards":
            items = catalog()
            self._json(200, {
                "active": active_id(items),
                "dashboards": [
                    {"id": item["id"], "title": item["title"], "source": item["source"]}
                    for item in items
                ],
            })
            return
        if path == "/api/layout":
            items = catalog()
            current = active_id(items)
            item = find_dashboard(current) if current else None
            if item is None:
                self._json(404, {"error": "kein Dashboard"})
                return
            self._json(200, public_dashboard(item))
            return
        dash_id = self._dash_id(path)
        if dash_id:
            item = find_dashboard(dash_id)
            if item is None:
                self._json(404, {"error": "Dashboard nicht gefunden"})
                return
            self._json(200, public_dashboard(item))
            return
        self._json(404, {"error": "nicht gefunden"})

    def do_PUT(self) -> None:  # noqa: N802
        self._save_dashboard()

    def do_DELETE(self) -> None:  # noqa: N802
        path = self.path.split("?", 1)[0]
        if not self._authorized():
            self._json(401, {"error": "token"})
            return
        dash_id = self._dash_id(path)
        if not dash_id:
            self._json(404, {"error": "nicht gefunden"})
            return
        if not delete_dashboard(dash_id):
            self._json(404, {"error": "keine eigene Kopie"})
            return
        log(f"dashboard gelöscht  {dash_id}")
        self._json(200, {"ok": True, "id": dash_id})

    def _save_dashboard(self) -> None:
        path = self.path.split("?", 1)[0]
        if not self._authorized():
            self._json(401, {"error": "token"})
            return
        dash_id = self._dash_id(path)
        if not dash_id:
            self._json(404, {"error": "nicht gefunden"})
            return
        try:
            data = self._read_json(300000)
            saved = save_dashboard(dash_id, data)
            if data.get("activate", True):
                set_active(dash_id)
        except (ValueError, TypeError, json.JSONDecodeError, OSError) as exc:
            self._json(400, {"error": str(exc)})
            return
        log(f"dashboard gespeichert  {dash_id}")
        item = find_dashboard(dash_id)
        self._json(200, public_dashboard(item) if item else saved)

    def do_POST(self) -> None:  # noqa: N802
        path = self.path.split("?", 1)[0]
        if not self._authorized():
            self._json(401, {"error": "token"})
            return
        if self._dash_id(path):
            self._save_dashboard()
            return
        if path == "/api/active":
            try:
                data = self._read_json()
                dash_id = str(data.get("id") or "")
            except (ValueError, TypeError, json.JSONDecodeError) as exc:
                self._json(400, {"error": str(exc)})
                return
            if find_dashboard(dash_id) is None:
                self._json(404, {"error": "Dashboard nicht gefunden"})
                return
            set_active(dash_id)
            self._json(200, {"ok": True, "id": dash_id})
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
    items = catalog()
    if not items:
        raise SystemExit("Keine Dashboards in dashboards/ gefunden.")
    for item in items:
        for err in validate_layout(item["data"]):
            log(f"{item['id']}: {err}")
    server = ThreadingHTTPServer((host, CONFIG["port"]), Handler)
    ips = lan_ips()
    log("")
    log(f"SC Cockpit {VERSION}")
    log("Dashboards: " + ", ".join(item["title"] for item in items))
    log(f"Aktiv: {active_id(items) or '-'}")
    log(f"Eigene Dashboards: {local_dir()}")
    log("Format für neue Dashboards: docs/DASHBOARD.md")
    if DRY_RUN:
        log("Trockenlauf: kein Windows, Tasten werden nur angezeigt.")
    admin = is_admin()
    if admin is False:
        log("Hinweis: nicht als Administrator. Wenn Star Citizen als Admin läuft,")
        log("         die Startmenü-Verknüpfung 'SC Cockpit (Administrator)' nutzen.")
    if CONFIG["token"]:
        log("Token ist aktiv.")
    else:
        log("Kein Token. Nur im eigenen WLAN benutzen.")
    log("")
    for ip in ips:
        log(f"  Gerät:  http://{ip}:{CONFIG['port']}/")
    log("")
    log("Dieses Fenster offen lassen, dann ins Spiel wechseln.")
    log("Strg+C beendet den Server.")
    log("")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        log("Beendet.")
    finally:
        BOARD.release_all()
        server.server_close()


def self_test() -> int:
    import shutil
    import tempfile
    import urllib.error
    import urllib.request

    example = json.loads((APP_ROOT / "docs" / "beispiel.json").read_text(encoding="utf-8"))
    cleaned = clean_dashboard(example, example["id"])
    if cleaned["id"] != "beispiel" or not cleaned["pages"]:
        log("docs/beispiel.json ist kein gültiges Dashboard")
        return 1
    schema = json.loads((APP_ROOT / "docs" / "dashboard.schema.json").read_text(encoding="utf-8"))
    if schema.get("properties", {}).get("pages", {}).get("maxItems") != 12:
        log("docs/dashboard.schema.json passt nicht zum Server")
        return 1

    tmp = Path(tempfile.mkdtemp(prefix="sc-cockpit-"))
    os.environ["SC_COCKPIT_HOME"] = str(tmp)
    items = catalog()
    if not any(item["id"] == "standard" for item in items):
        log("standard-Dashboard fehlt")
        return 1
    for item in items:
        errors = validate_layout(item["data"])
        if errors:
            for err in errors:
                log(f"{item['id']}: {err}")
            return 1
    BOARD.tap(["n"], 15)
    BOARD.finger_down(["lshift"])
    BOARD.finger_up(["lshift"])
    BOARD.release_all()
    server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    port = server.server_address[1]
    base = f"http://127.0.0.1:{port}"

    def call(method: str, path: str, payload: dict | None = None) -> dict:
        data = None if payload is None else json.dumps(payload).encode()
        req = urllib.request.Request(base + path, data=data, method=method)
        if payload is not None:
            req.add_header("Content-Type", "application/json")
        try:
            with urllib.request.urlopen(req) as res:
                raw = res.read().decode()
                return json.loads(raw) if raw else {}
        except urllib.error.HTTPError as exc:
            detail = exc.read().decode()
            raise SystemExit(f"{method} {path} -> {exc.code} {detail}") from exc

    try:
        health = call("GET", "/api/health")
        if not health.get("ok") or health.get("version") != VERSION:
            raise SystemExit("health nicht ok")
        listed = call("GET", "/api/dashboards")
        ids = {item["id"] for item in listed["dashboards"]}
        if "standard" not in ids or "mining" not in ids:
            raise SystemExit(f"Dashboards unvollständig: {ids}")
        saved = call("PUT", "/api/dashboard/selftest", {
            "title": "Selftest",
            "pages": [{
                "id": "seite",
                "label": "Test",
                "groups": [{
                    "title": "X",
                    "buttons": [{
                        "id": "gear",
                        "label": "Fahrwerk",
                        "keys": ["n"],
                        "mode": "toggle",
                    }],
                }],
            }],
        })
        if saved.get("_source") != "local" or saved.get("id") != "selftest":
            raise SystemExit(f"Speichern fehlgeschlagen: {saved}")
        call("POST", "/api/active", {"id": "mining"})
        if active_id() != "mining":
            raise SystemExit("aktives Dashboard nicht gesetzt")
        call("DELETE", "/api/dashboard/selftest")
        if find_dashboard("selftest") is not None:
            raise SystemExit("Dashboard nicht gelöscht")
        call("POST", "/api/press", {"keys": ["n"], "mode": "tap", "id": "test"})
        with urllib.request.urlopen(base + "/") as res:
            html = res.read().decode()
        if "Anpassen" not in html:
            raise SystemExit("index.html ohne Editor")
    finally:
        server.shutdown()
        server.server_close()
        os.environ.pop("SC_COCKPIT_HOME", None)
        shutil.rmtree(tmp, ignore_errors=True)
    log(f"self-test ok ({len(items)} dashboards, dry_run={DRY_RUN})")
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
        log("Dashboard-Format: docs/DASHBOARD.md")
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
