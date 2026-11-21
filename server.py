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

import io
import json
import os
import re
import shutil
import socket
import subprocess
import sys
import threading
import time
import urllib.error
import urllib.request
import zipfile
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import unquote

import sc_ships

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


# --- GitHub-Stand: gleicher Entscheid für Button, Web und Start-Badge ---

GITHUB_API = "https://api.github.com/repos/MOON-132/Star-Citizen-Cockpit/commits/main"
GITHUB_REPO_ZIP = "https://github.com/MOON-132/Star-Citizen-Cockpit/archive/{sha}.zip"
COMMIT_NAME = ".sc-cockpit-commit"
BADGE_TEXT = "Neue Version verfügbar"
_PRESERVE_NAMES = {"config.local.json", "state.json", COMMIT_NAME, ".sc-cockpit-update.lock", ".sc-cockpit-restart"}


def one_line(text: str, limit: int = 160) -> str:
    text = " ".join((text or "").split())
    if len(text) > limit:
        return text[: limit - 1] + "…"
    return text


def describe_update(local_sha: str, remote_sha: str, fetch_error: str) -> dict:
    """Vergleicht lokalen und GitHub-Commit.

    status: update | current | error
    badge: nur bei update (Start-Hinweis). Netzwerkfehler bleibt ohne Badge.
    """
    err = (fetch_error or "").strip()
    remote = (remote_sha or "").strip().lower()
    local = (local_sha or "").strip().lower()
    if err or not remote:
        return {
            "ok": False,
            "status": "error",
            "message": "Update-Fehler: " + (err or "GitHub-Stand unbekannt"),
            "local": local,
            "remote": remote,
            "badge": False,
            "badge_text": "",
        }
    if local != remote:
        return {
            "ok": True,
            "status": "update",
            "message": "Update verfügbar.",
            "local": local,
            "remote": remote,
            "badge": True,
            "badge_text": BADGE_TEXT,
        }
    return {
        "ok": True,
        "status": "current",
        "message": "SC Cockpit ist aktuell.",
        "local": local,
        "remote": remote,
        "badge": False,
        "badge_text": "",
    }


def _update_error(detail: str) -> dict:
    got = describe_update("", "", one_line(detail) or "unbekannt")
    return got


def http_get(url: str, timeout: float) -> bytes:
    req = urllib.request.Request(
        url,
        headers={"User-Agent": "SC-Cockpit", "Accept": "application/vnd.github+json"},
    )
    with urllib.request.urlopen(req, timeout=timeout) as res:
        return res.read()


def fetch_remote_sha(timeout: float = 20, url: str = "") -> tuple[str, str]:
    try:
        raw = http_get(url or GITHUB_API, timeout)
        data = json.loads(raw.decode("utf-8"))
        sha = str(data.get("sha") or "").strip()
        if len(sha) < 7 or any(c not in "0123456789abcdefABCDEF" for c in sha):
            return "", "GitHub-Antwort ohne Commit"
        return sha, ""
    except urllib.error.HTTPError as exc:
        return "", f"GitHub HTTP {exc.code}"
    except Exception:
        return "", "keine Verbindung zu GitHub"


def _git_env(root: Path) -> dict:
    env = os.environ.copy()
    env["GIT_CEILING_DIRECTORIES"] = str(root.resolve())
    return env


def read_local_sha(root: Path, timeout: float = 20) -> tuple[str, str]:
    git_meta = root / ".git"
    if git_meta.exists():
        git = shutil.which("git")
        if not git:
            return "", "git nicht gefunden"
        try:
            proc = subprocess.run(
                [git, "rev-parse", "HEAD"],
                cwd=root,
                capture_output=True,
                text=True,
                timeout=timeout,
                env=_git_env(root),
            )
        except subprocess.TimeoutExpired:
            return "", "git Zeitüberschreitung"
        if proc.returncode != 0:
            detail = (proc.stderr or proc.stdout or "git rev-parse fehlgeschlagen").strip()
            return "", one_line(detail) or "git rev-parse fehlgeschlagen"
        return proc.stdout.strip(), ""
    marker = root / COMMIT_NAME
    if marker.is_file():
        parts = marker.read_text(encoding="utf-8", errors="replace").strip().split()
        return (parts[0] if parts else ""), ""
    return "", ""


def _write_marker(root: Path, sha: str) -> None:
    (root / COMMIT_NAME).write_text(sha.strip().lower() + "\n", encoding="utf-8")


def _skip_rel(rel: str) -> bool:
    rel = rel.replace("\\", "/").lstrip("/")
    if not rel:
        return True
    parts = [p for p in rel.split("/")]
    if any(part in {"", ".", ".."} for part in parts):
        return True
    if parts[-1] in _PRESERVE_NAMES or parts[-1] == "config.local.json":
        return True
    if rel == "dashboards/local" or rel.startswith("dashboards/local/"):
        return True
    if parts[0] == ".git":
        return True
    return False


def _zip_prefix(names: list[str]) -> str:
    tops = set()
    has_slash = False
    for name in names:
        name = name.replace("\\", "/").lstrip("/")
        if not name or name.endswith("/"):
            continue
        tops.add(name.split("/", 1)[0])
        if "/" in name:
            has_slash = True
    if has_slash and len(tops) == 1:
        return next(iter(tops)) + "/"
    return ""


def write_program_file(target: Path, data: bytes) -> str:
    target.parent.mkdir(parents=True, exist_ok=True)
    try:
        if target.is_file() and target.read_bytes() == data:
            return "same"
    except OSError:
        pass
    part = Path(str(target) + ".part")
    part.write_bytes(data)
    try:
        os.replace(part, target)
    except PermissionError:
        try:
            part.unlink()
        except OSError:
            pass
        if target.suffix.lower() != ".exe":
            raise
        Path(str(target) + ".new").write_bytes(data)
        return "staged"
    if target.suffix.lower() in {".exe", ".bat", ".ps1"}:
        try:
            os.chmod(target, 0o755)
        except OSError:
            pass
    return "written"


def apply_repo_zip(root: Path, blob: bytes) -> dict:
    """Ersetzt Programmdateien. config.local.json, state.json und dashboards/local bleiben."""
    root_r = root.resolve()
    files = 0
    restart_exe = False
    with zipfile.ZipFile(io.BytesIO(blob)) as zf:
        names = [n for n in zf.namelist() if not n.endswith("/")]
        prefix = _zip_prefix(names)
        for info in zf.infolist():
            if info.is_dir():
                continue
            rel = info.filename.replace("\\", "/")
            if prefix and rel.startswith(prefix):
                rel = rel[len(prefix):]
            rel = rel.lstrip("/")
            if _skip_rel(rel):
                continue
            target = (root_r / rel).resolve()
            try:
                target.relative_to(root_r)
            except ValueError:
                continue
            how = write_program_file(target, zf.read(info))
            files += 1
            if rel.lower().endswith(".exe") and how in {"written", "staged"}:
                restart_exe = True
    mirror = _mirror_setup_exe(root_r)
    if mirror in {"written", "staged"}:
        restart_exe = True
        files += 1
    return {"files": files, "restart_exe": restart_exe}


def _mirror_setup_exe(root: Path) -> str:
    staged = Path(str(root / "SC-Cockpit-Setup.exe") + ".new")
    src = staged if staged.is_file() else root / "SC-Cockpit-Setup.exe"
    other = root / "SC-Cockpit.exe"
    if not src.is_file():
        return ""
    if not other.exists() and not Path(str(other) + ".new").exists():
        return ""
    if src.resolve() == other.resolve():
        return ""
    return write_program_file(other, src.read_bytes())


def _acquire_update_lock(root: Path) -> Path | None:
    lock = root / ".sc-cockpit-update.lock"
    if lock.exists():
        try:
            age = time.time() - lock.stat().st_mtime
        except OSError:
            age = 0
        if age < 300:
            return None
        try:
            lock.unlink()
        except OSError:
            return None
    try:
        fd = os.open(str(lock), os.O_CREAT | os.O_EXCL | os.O_WRONLY)
    except FileExistsError:
        return None
    try:
        os.write(fd, str(os.getpid()).encode())
    finally:
        os.close(fd)
    return lock


def _download_zip(sha: str) -> bytes:
    clean = sha.strip().lower()
    if len(clean) < 7 or any(c not in "0123456789abcdef" for c in clean):
        raise ValueError("Commit ungültig")
    return http_get(GITHUB_REPO_ZIP.format(sha=clean), 120)


def _apply_git(root: Path, remote: str) -> dict:
    git = shutil.which("git")
    if not git:
        return _update_error("git nicht gefunden")
    try:
        proc = subprocess.run(
            [git, "pull", "--ff-only"],
            cwd=root,
            capture_output=True,
            text=True,
            timeout=90,
            env=_git_env(root),
        )
    except subprocess.TimeoutExpired:
        return _update_error("git pull Zeitüberschreitung")
    except OSError as exc:
        return _update_error(str(exc))
    if proc.returncode != 0:
        lines = [ln.strip() for ln in (proc.stderr or proc.stdout or "").splitlines() if ln.strip()]
        return _update_error(lines[0] if lines else "git pull fehlgeschlagen")
    new_local, err = read_local_sha(root)
    if err:
        return _update_error(err)
    if new_local.lower() != remote.lower():
        return _update_error("git pull hat den GitHub-Stand nicht übernommen")
    _write_marker(root, new_local)
    return {
        "ok": True,
        "status": "updated",
        "message": "Update installiert.",
        "local": new_local.lower(),
        "remote": remote.lower(),
        "badge": False,
        "badge_text": "",
        "restart_exe": False,
    }


def _apply_zip(root: Path, remote: str) -> dict:
    blob = _download_zip(remote)
    if len(blob) < 1000 or blob[:2] != b"PK":
        return _update_error("ZIP von GitHub ist leer")
    info = apply_repo_zip(root, blob)
    if info["files"] <= 0:
        return _update_error("ZIP ohne Programmdateien")
    _write_marker(root, remote)
    return {
        "ok": True,
        "status": "updated",
        "message": "Update installiert.",
        "local": remote.lower(),
        "remote": remote.lower(),
        "badge": False,
        "badge_text": "",
        "files": info["files"],
        "restart_exe": info["restart_exe"],
    }


def github_update(root: Path, apply: bool = True, timeout: float = 20) -> dict:
    root = Path(root)
    if not root.is_dir():
        return _update_error("Ordner nicht gefunden")
    local, local_err = read_local_sha(root, timeout=timeout)
    if local_err:
        got = _update_error(local_err)
        return got
    remote, remote_err = fetch_remote_sha(timeout)
    decision = describe_update(local, remote, remote_err)
    if decision["status"] != "update" or not apply:
        return decision
    lock = _acquire_update_lock(root)
    if lock is None:
        return _update_error("Update läuft bereits")
    try:
        if (root / ".git").exists():
            return _apply_git(root, remote)
        return _apply_zip(root, remote)
    except Exception as exc:
        return _update_error(str(exc) or "Update fehlgeschlagen")
    finally:
        try:
            lock.unlink()
        except OSError:
            pass


def run_github_update_cli(apply: bool) -> int:
    args = sys.argv[1:]
    root = APP_ROOT
    if "--root" in args:
        idx = args.index("--root")
        if idx + 1 >= len(args):
            result = _update_error("--root fehlt")
        else:
            root = Path(args[idx + 1])
            result = None
    else:
        result = None
    if result is None:
        timeout = 8 if not apply else 20
        try:
            result = github_update(root, apply=apply, timeout=timeout)
        except Exception as exc:
            result = _update_error(str(exc) or "Update fehlgeschlagen")
    sys.stdout.write(json.dumps(result, ensure_ascii=False) + "\n")
    sys.stdout.flush()
    return 0 if result.get("status") != "error" else 1


def _exit_soon() -> None:
    time.sleep(0.5)
    os._exit(0)


def test_update_decisions() -> None:
    """Stale → Update + Badge, gleich → aktuell ohne Badge, Netzfehler → Fehler ohne Badge."""
    stale = describe_update("altcommit", "neucommit", "")
    if stale["status"] != "update" or "Update verfügbar" not in stale["message"] or not stale["badge"]:
        raise SystemExit(f"veraltet falsch: {stale}")
    if stale["badge_text"] != BADGE_TEXT:
        raise SystemExit(f"Badge-Text falsch: {stale}")
    unknown = describe_update("", "deadbeef", "")
    if unknown["status"] != "update" or not unknown["badge"]:
        raise SystemExit(f"ohne lokalen Stand falsch: {unknown}")
    same = describe_update("AbC", "abc", "")
    if same["status"] != "current" or "aktuell" not in same["message"] or same["badge"] or same["badge_text"]:
        raise SystemExit(f"aktuell falsch: {same}")
    down = describe_update("abc", "", "keine Verbindung zu GitHub")
    if down["status"] != "error" or down["badge"] or "aktuell" in down["message"].lower():
        raise SystemExit(f"Netzwerkfehler als aktuell gewertet: {down}")
    if not down["message"].startswith("Update-Fehler:"):
        raise SystemExit(f"Fehlertext fehlt: {down}")

    saved = GITHUB_API
    try:
        globals()["GITHUB_API"] = "http://127.0.0.1:1/nope"
        sha, err = fetch_remote_sha(timeout=3)
    finally:
        globals()["GITHUB_API"] = saved
    if sha or not err:
        raise SystemExit(f"lokaler Verbindungsfehler nicht gemeldet: {sha!r} {err!r}")
    live_down = describe_update("abc", sha, err)
    if live_down["status"] != "error" or live_down["badge"] or "aktuell" in live_down["message"].lower():
        raise SystemExit(f"Verbindungsfehler zeigt Badge oder aktuell: {live_down}")

    tmp = Path(tempfile_mkdtemp())
    sibling = tmp.parent / (tmp.name + "-user")
    try:
        (tmp / "server.py").write_text("OLD", encoding="utf-8")
        (tmp / "config.local.json").write_text('{"port": 9999}\n', encoding="utf-8")
        (tmp / "state.json").write_text('{"active":"mine"}\n', encoding="utf-8")
        local = tmp / "dashboards" / "local"
        local.mkdir(parents=True)
        (local / "my.json").write_text('{"id":"my"}\n', encoding="utf-8")
        (local / "state.json").write_text('{"active":"local"}\n', encoding="utf-8")
        sibling.mkdir()
        (sibling / "config.local.json").write_text("USER\n", encoding="utf-8")
        buf = io.BytesIO()
        with zipfile.ZipFile(buf, "w") as zf:
            zf.writestr("Star-Citizen-Cockpit-main/server.py", "NEW")
            zf.writestr("Star-Citizen-Cockpit-main/installer/qrcode/qr.go", "package qrcode\n")
            zf.writestr("Star-Citizen-Cockpit-main/config.local.json", "NUKE\n")
            zf.writestr("Star-Citizen-Cockpit-main/state.json", "NUKE\n")
            zf.writestr("Star-Citizen-Cockpit-main/dashboards/local/my.json", "NUKE\n")
            zf.writestr("Star-Citizen-Cockpit-main/../../evil.txt", "NUKE\n")
        info = apply_repo_zip(tmp, buf.getvalue())
        if (tmp / "server.py").read_text(encoding="utf-8") != "NEW":
            raise SystemExit("Programmdatei nicht ersetzt")
        if (tmp / "installer" / "qrcode" / "qr.go").read_text(encoding="utf-8") != "package qrcode\n":
            raise SystemExit("neue Datei fehlt")
        if (tmp / "config.local.json").read_text(encoding="utf-8") != '{"port": 9999}\n':
            raise SystemExit("config.local.json verändert")
        if (tmp / "state.json").read_text(encoding="utf-8") != '{"active":"mine"}\n':
            raise SystemExit("state.json verändert")
        if (local / "my.json").read_text(encoding="utf-8") != '{"id":"my"}\n':
            raise SystemExit("dashboards/local verändert")
        if (local / "state.json").read_text(encoding="utf-8") != '{"active":"local"}\n':
            raise SystemExit("lokale state.json verändert")
        if (sibling / "config.local.json").read_text(encoding="utf-8") != "USER\n":
            raise SystemExit("Benutzerordner verändert")
        if (tmp.parent / "evil.txt").exists() or (tmp.parent.parent / "evil.txt").exists():
            raise SystemExit("Zip-Slip")
        if info["files"] < 2:
            raise SystemExit(f"zu wenige Dateien: {info}")
    finally:
        shutil.rmtree(tmp, ignore_errors=True)
        shutil.rmtree(sibling, ignore_errors=True)

    fake = Path(tempfile_mkdtemp())
    try:
        (fake / ".git").mkdir()
        (fake / "config.local.json").write_text("KEEP", encoding="utf-8")
        broken = github_update(fake, apply=True, timeout=3)
        if broken["status"] != "error" or broken["badge"] or "aktuell" in broken["message"].lower():
            raise SystemExit(f"git-Fehler als aktuell gewertet: {broken}")
        if (fake / "config.local.json").read_text(encoding="utf-8") != "KEEP":
            raise SystemExit("git-Fehler hat config.local.json verändert")
    finally:
        shutil.rmtree(fake, ignore_errors=True)


def tempfile_mkdtemp() -> str:
    import tempfile
    return tempfile.mkdtemp(prefix="sc-upd-")


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


WIDGET_KINDS = {"button", "telemetry", "pair", "guard"}


def validate_layout(layout: dict) -> list[str]:
    errors = []
    count = 0
    skipped = 0
    for page, _group, button in iter_buttons(layout):
        kind = str(button.get("kind") or "button")
        if kind not in WIDGET_KINDS:
            # Retired live widgets (power/shield) and unknown kinds are ignored.
            skipped += 1
            continue
        count += 1
        if kind == "telemetry":
            continue
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
    if count == 0 and skipped == 0:
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


def _register_dashboard_data(data: dict, source: str, path: Path | None, items: list, index: dict) -> None:
    if not isinstance(data, dict) or not isinstance(data.get("pages"), list):
        return
    dash_id = str(data.get("id") or (path.stem if path else ""))
    if dash_id in RESERVED_IDS or not ID_RE.fullmatch(dash_id):
        log(f"Dashboard-Id ungültig: {dash_id}")
        return
    pages_list = data.get("pages") or []
    btn_count = sum(
        len(g.get("buttons") or [])
        for p in pages_list if isinstance(p, dict)
        for g in (p.get("groups") or []) if isinstance(g, dict)
    )
    ship_name = str(data.get("ship") or data.get("title") or dash_id)[:40]
    mfg = str(data.get("manufacturer") or "UEE")[:24]
    role = str(data.get("role") or "Multi-Role")[:40]
    hull_type = str(data.get("hull_type") or "")[:32]
    if not hull_type:
        low_r = (role + " " + dash_id).lower()
        if "mining" in low_r:
            hull_type = "mining_bubble"
        elif "salvage" in low_r:
            hull_type = "twin_industrial"
        elif "hauler" in low_r or "fracht" in low_r or "cargo" in low_r:
            hull_type = "crusader_wing"
        else:
            hull_type = "fighter_delta"
    bg = str(data.get("background") or "").strip()
    if not bg or bg.startswith("preset:"):
        bg = sc_ships.make_ship_svg_data_uri(ship_name, mfg, role, hull_type)
        data = dict(data)
        data["background"] = bg
        data.setdefault("bg_opacity", 0.48)
    subsystems = data.get("subsystems")
    if not isinstance(subsystems, list) or not subsystems:
        subsystems = [str(p.get("id") or "") for p in pages_list if isinstance(p, dict) and p.get("id")]
    entry = {
        "id": dash_id,
        "title": str(data.get("title") or dash_id)[:40],
        "ship": ship_name,
        "manufacturer": mfg,
        "role": role,
        "hull_type": hull_type,
        "subsystems": subsystems,
        "background": bg,
        "icon": str(data.get("icon") or "combat")[:20],
        "page_count": len(pages_list),
        "button_count": btn_count,
        "source": source,
        "path": path,
        "data": data,
    }
    if dash_id in index:
        items[index[dash_id]] = entry
    else:
        index[dash_id] = len(items)
        items.append(entry)


def _read_dashboard(path: Path, source: str, items: list, index: dict) -> None:
    try:
        data = load_json(path)
    except json.JSONDecodeError as exc:
        log(f"{path.name}: {exc}")
        return
    if data:
        _register_dashboard_data(data, source, path, items, index)


DEFAULT_FAVORITES = [
    "aegis-gladius",
    "misc-prospector",
    "drake-vulture",
    "crusader-c2-hercules",
    "drake-cutlass-black",
    "anvil-carrack",
    "rsi-constellation-andromeda",
    "aegis-avenger-titan",
]


def _load_state() -> dict:
    try:
        raw = load_json(state_path())
        if isinstance(raw, dict):
            return raw
    except (json.JSONDecodeError, OSError):
        pass
    return {}


def _save_state(st: dict) -> None:
    path = state_path()
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(st, indent=2) + "\n", encoding="utf-8")


def get_favorites() -> list[str]:
    st = _load_state()
    favs = st.get("favorites")
    if isinstance(favs, list):
        return [str(x) for x in favs if isinstance(x, str)]
    return list(DEFAULT_FAVORITES)


def toggle_favorite(dash_id: str, fav_state: bool | None = None) -> list[str]:
    st = _load_state()
    favs = get_favorites()
    if fav_state is True:
        if dash_id not in favs:
            favs.append(dash_id)
    elif fav_state is False:
        favs = [x for x in favs if x != dash_id]
    else:
        if dash_id in favs:
            favs = [x for x in favs if x != dash_id]
        else:
            favs.append(dash_id)
    st["favorites"] = favs
    _save_state(st)
    return favs


def catalog() -> list[dict]:
    items: list[dict] = []
    index: dict[str, int] = {}
    # 1. All 100+ Star Citizen Ships with personalized dashboards & SVG ship blueprints
    for ship_data in sc_ships.all_generated_ship_dashboards():
        _register_dashboard_data(ship_data, "builtin", None, items, index)
    # 2. Legacy/custom builtin dashboards (skip redundant legacy duplicates of Gladius/Prospector/Vulture/Hercules)
    legacy_aliases = {"standard", "mining", "salvage", "hauler"}
    folder = builtin_dir()
    if folder.is_dir():
        for path in sorted(folder.glob("*.json")):
            if path.stem not in legacy_aliases:
                _read_dashboard(path, "builtin", items, index)
    legacy_local = APP_ROOT / "layout.local.json"
    if legacy_local.is_file():
        _read_dashboard(legacy_local, "local", items, index)
    for path in sorted(local_dir().glob("*.json")):
        _read_dashboard(path, "local", items, index)
    fav_set = set(get_favorites())
    for item in items:
        item["favorite"] = item["id"] in fav_set or item["source"] == "local"
    return items


def active_id(items: list[dict] | None = None) -> str:
    items = catalog() if items is None else items
    ids = [item["id"] for item in items]
    st = _load_state()
    want = st.get("active")
    if want in ids:
        return str(want)
    if "aegis-gladius" in ids:
        return "aegis-gladius"
    if "standard" in ids:
        return "standard"
    return ids[0] if ids else ""


def set_active(dash_id: str) -> None:
    st = _load_state()
    st["active"] = dash_id
    _save_state(st)


LEGACY_ID_MAP = {
    "standard": "aegis-gladius",
    "mining": "misc-prospector",
    "salvage": "drake-vulture",
    "hauler": "crusader-c2-hercules",
}


def find_dashboard(dash_id: str) -> dict | None:
    items = catalog()
    for item in items:
        if item["id"] == dash_id:
            return item
    mapped = LEGACY_ID_MAP.get(dash_id)
    if mapped:
        for item in items:
            if item["id"] == mapped:
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
                kind = str(button.get("kind") or "button")
                if kind not in WIDGET_KINDS:
                    continue
                keys_raw = button.get("keys")
                if kind == "telemetry" and not keys_raw:
                    norm = ["f8"]
                else:
                    if not isinstance(keys_raw, list) or not 1 <= len(keys_raw) <= 4:
                        raise ValueError(f"Tasten fehlen bei {button.get('label') or 'Button'}")
                    norm = [normalize_key(str(key)) for key in keys_raw]
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
                entry = {
                    "id": bid,
                    "label": str(button.get("label") or bid)[:40],
                    "hint": str(button.get("hint") or "")[:60],
                    "keys": norm,
                    "mode": mode,
                    "color": color,
                    "wide": bool(button.get("wide")),
                    "danger": bool(button.get("danger")),
                }
                if kind != "button":
                    entry["kind"] = kind
                if button.get("sc"):
                    entry["sc"] = str(button["sc"])[:60]
                if button.get("keys2") and isinstance(button["keys2"], list):
                    norm2 = [normalize_key(str(k)) for k in button["keys2"][:4] if known_key(normalize_key(str(k)))]
                    if norm2:
                        entry["keys2"] = norm2
                if button.get("span") in {1, 2, 3, 4}:
                    entry["span"] = int(button["span"])
                buttons.append(entry)
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
    result = {"id": dash_id, "title": str(data.get("title") or dash_id)[:40], "pages": pages}
    for meta_key, max_len in (("ship", 40), ("manufacturer", 24), ("role", 40), ("icon", 20), ("hull_type", 32)):
        val = str(data.get(meta_key) or "").strip()
        if val:
            result[meta_key] = val[:max_len]
    if isinstance(data.get("subsystems"), list):
        result["subsystems"] = [str(s)[:24] for s in data["subsystems"][:12] if s]
    bg = str(data.get("background") or "").strip()
    if bg and len(bg) <= 250000:
        result["background"] = bg
    if "bg_opacity" in data:
        try:
            op = max(0.05, min(1.0, float(data["bg_opacity"])))
            result["bg_opacity"] = round(op, 2)
        except (ValueError, TypeError):
            pass
    return result


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


class TelemetryState:
    """Tracks key-echoed ship toggles and Star Citizen Game.log / process status. No power percentages."""

    def __init__(self) -> None:
        self.lock = threading.Lock()
        self.systems = {
            "flight_ready": False,
            "gear": True,
            "vtol": False,
            "lights": False,
            "decoupled": False,
            "cruise": False,
            "master_mode": "SCM",
        }
        self.sc_running = False
        self.sc_ship = "AEGIS / UEE COCKPIT"
        self.sc_zone = "Stanton System"
        self.last_action = "-"
        self.last_check = 0.0

    def on_keys_pressed(self, keys: list[str], label: str = "") -> None:
        combo = "+".join(keys)
        with self.lock:
            self.last_action = f"{label} ({combo})" if label else combo
            if combo == "ralt+r":
                self.systems["flight_ready"] = True
            elif combo == "n":
                self.systems["gear"] = not self.systems["gear"]
            elif combo == "k":
                self.systems["vtol"] = not self.systems["vtol"]
            elif combo == "l":
                self.systems["lights"] = not self.systems["lights"]
            elif combo == "c":
                self.systems["decoupled"] = not self.systems["decoupled"]
            elif combo == "lalt+c":
                self.systems["cruise"] = not self.systems["cruise"]
            elif combo == "b":
                self.systems["master_mode"] = "NAV / QUANTUM" if self.systems["master_mode"] == "SCM" else "SCM"

    def update(self, payload: dict) -> dict:
        with self.lock:
            if isinstance(payload.get("systems"), dict):
                for k, v in payload["systems"].items():
                    if k in self.systems:
                        self.systems[k] = v
        return self.snapshot()

    def _poll_sc_environment(self) -> None:
        now = time.monotonic()
        if now - self.last_check < 4.0:
            return
        self.last_check = now
        if sys.platform == "win32":
            try:
                class MEMORYSTATUSEX(ctypes.Structure):
                    _fields_ = [
                        ("dwLength", wintypes.DWORD),
                        ("dwMemoryLoad", wintypes.DWORD),
                        ("ullTotalPhys", ctypes.c_uint64),
                        ("ullAvailPhys", ctypes.c_uint64),
                        ("ullTotalPageFile", ctypes.c_uint64),
                        ("ullAvailPageFile", ctypes.c_uint64),
                        ("ullTotalVirtual", ctypes.c_uint64),
                        ("ullAvailVirtual", ctypes.c_uint64),
                        ("ullAvailExtendedVirtual", ctypes.c_uint64),
                    ]
                mem = MEMORYSTATUSEX()
                mem.dwLength = ctypes.sizeof(MEMORYSTATUSEX)
                ctypes.windll.kernel32.GlobalMemoryStatusEx(ctypes.byref(mem))
                self.ram_pct = int(mem.dwMemoryLoad)
            except Exception:
                self.ram_pct = 0

        # Scan common Star Citizen Game.log locations for active ship or location
        for base in _sc_candidate_dirs():
            gamelog = base / "Game.log"
            if gamelog.is_file():
                self.sc_running = (time.time() - gamelog.stat().st_mtime) < 300
                try:
                    with gamelog.open("rb") as fh:
                        fh.seek(0, os.SEEK_END)
                        size = fh.tell()
                        fh.seek(max(0, size - 24576), os.SEEK_SET)
                        tail = fh.read().decode("utf-8", errors="ignore")
                    ship_matches = re.findall(r"([A-Z]{3,5}_[A-Za-z0-9_]+)", tail)
                    for sm in reversed(ship_matches):
                        if any(p in sm for p in ("AEGS_", "ANVL_", "orig_", "ORIG_", "DRAK_", "MISC_", "RSI_", "CRUS_", "ARGO_", "CNOU_", "GAMA_", "MIRAI_")):
                            clean_ship = sm.split("_")[0] + " " + " ".join(sm.split("_")[1:3])
                            self.sc_ship = clean_ship.upper()
                            break
                except Exception:
                    pass
                break

    def snapshot(self) -> dict:
        self._poll_sc_environment()
        with self.lock:
            return {
                "systems": dict(self.systems),
                "sc_running": self.sc_running,
                "sc_ship": self.sc_ship,
                "sc_zone": self.sc_zone,
                "last_action": self.last_action,
                "ram_pct": getattr(self, "ram_pct", 0),
            }


def _sc_candidate_dirs() -> list[Path]:
    candidates = []
    for drive in ("C:", "D:", "E:", "F:"):
        candidates.append(Path(drive + "/Program Files/Roberts Space Industries/StarCitizen/LIVE"))
        candidates.append(Path(drive + "/Roberts Space Industries/StarCitizen/LIVE"))
        candidates.append(Path(drive + "/Games/Roberts Space Industries/StarCitizen/LIVE"))
        candidates.append(Path(drive + "/StarCitizen/LIVE"))
    return [c for c in candidates if c.is_dir()]


def read_sc_actionmaps() -> dict:
    """Reads user keybindings from Star Citizen's actionmaps.xml if installed (inspired by tablet-deck)."""
    import xml.etree.ElementTree as ET
    for live_dir in _sc_candidate_dirs():
        xml_path = live_dir / "user/client/0/Profiles/default/actionmaps.xml"
        if not xml_path.is_file():
            xml_path = live_dir / "USER/Client/0/Profiles/default/actionmaps.xml"
        if xml_path.is_file():
            bindings = {}
            try:
                root = ET.parse(xml_path).getroot()
                for action in root.iter("action"):
                    act_name = action.get("name") or ""
                    for rb in action.iter("rebind"):
                        inp = (rb.get("input") or "").strip()
                        if inp.startswith("kb1_") and inp[4:].strip():
                            raw_k = inp[4:].strip().lower()
                            parts = [normalize_key(p) for p in raw_k.split("+") if p.strip()]
                            if parts and all(known_key(p) for p in parts):
                                bindings[act_name] = parts
            except Exception as exc:
                return {"found": True, "path": str(xml_path), "error": str(exc), "bindings": {}}
            return {"found": True, "path": str(xml_path), "bindings": bindings}
    return {"found": False, "path": "", "bindings": {}}


CONFIG = load_config()
BOARD = Board(CONFIG["hold_timeout_s"])
TELEMETRY = TelemetryState()


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
        try:
            self.send_response(code)
            self.send_header("Content-Type", "application/json; charset=utf-8")
            self.send_header("Content-Length", str(len(body)))
            self.send_header("Cache-Control", "no-store, no-cache, must-revalidate")
            self.send_header("Connection", "close")
            self.end_headers()
            self.wfile.write(body)
        except (BrokenPipeError, ConnectionResetError, OSError):
            pass
        self.close_connection = True

    def _bytes(self, code: int, body: bytes, content_type: str) -> None:
        try:
            self.send_response(code)
            self.send_header("Content-Type", content_type)
            self.send_header("Content-Length", str(len(body)))
            self.send_header("Cache-Control", "no-store, no-cache, must-revalidate")
            self.send_header("Connection", "close")
            self.end_headers()
            self.wfile.write(body)
        except (BrokenPipeError, ConnectionResetError, OSError):
            pass
        self.close_connection = True

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
            html_bytes = file.read_bytes().replace(b"\r\n", b"\n")
            self._bytes(200, html_bytes, "text/html; charset=utf-8")
            return
        if path == "/favicon.ico":
            ico = APP_ROOT / "sc-cockpit.ico"
            if ico.is_file():
                self._bytes(200, ico.read_bytes(), "image/x-icon")
            else:
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
                "telemetry": TELEMETRY.snapshot(),
            })
            return
        if path == "/api/telemetry":
            self._json(200, TELEMETRY.snapshot())
            return
        if path == "/api/sc-bindings":
            self._json(200, read_sc_actionmaps())
            return
        if path == "/api/dashboards":
            items = catalog()
            favs = get_favorites()
            self._json(200, {
                "active": active_id(items),
                "favorites": favs,
                "dashboards": [
                    {
                        "id": item["id"],
                        "title": item["title"],
                        "ship": item.get("ship", item["title"]),
                        "manufacturer": item.get("manufacturer", "UEE"),
                        "role": item.get("role", "Multi-Role"),
                        "hull_type": item.get("hull_type", "fighter_delta"),
                        "subsystems": item.get("subsystems", []),
                        "background": item.get("background", ""),
                        "favorite": bool(item.get("favorite", False)),
                        "icon": item.get("icon", "combat"),
                        "page_count": item.get("page_count", len(item["data"].get("pages", []))),
                        "button_count": item.get("button_count", 0),
                        "source": item["source"],
                    }
                    for item in items
                ],
            })
            return
        if path == "/api/favorites":
            self._json(200, {"favorites": get_favorites()})
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
            data = self._read_json(600000)
            saved = save_dashboard(dash_id, data)
            # Automatically add user-customized ships to Hangar favorites
            toggle_favorite(dash_id, True)
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
        if path == "/api/favorites":
            try:
                data = self._read_json()
                dash_id = str(data.get("id") or "").strip()
                fav_state = data.get("favorite")
                if not isinstance(fav_state, bool):
                    fav_state = None
            except (ValueError, TypeError, json.JSONDecodeError) as exc:
                self._json(400, {"error": str(exc)})
                return
            if not dash_id or find_dashboard(dash_id) is None:
                self._json(404, {"error": "Schiff nicht gefunden"})
                return
            favs = toggle_favorite(dash_id, fav_state)
            self._json(200, {"ok": True, "id": dash_id, "favorite": dash_id in favs, "favorites": favs})
            return
        if path == "/api/git-pull":
            result = github_update(APP_ROOT, apply=True)
            if result.get("status") == "updated" and os.environ.get("SC_COCKPIT_LAUNCHER") == "1":
                try:
                    (APP_ROOT / ".sc-cockpit-restart").write_text("1", encoding="utf-8")
                    threading.Thread(target=_exit_soon, daemon=True).start()
                except OSError:
                    pass
            elif result.get("status") == "updated":
                result = dict(result)
                result["message"] = "Update installiert. Bitte Server neu starten."
            self._json(200, result)
            return
        if path == "/api/telemetry":
            try:
                data = self._read_json(32768)
                snap = TELEMETRY.update(data)
            except (ValueError, TypeError, json.JSONDecodeError) as exc:
                self._json(400, {"error": str(exc)})
                return
            self._json(200, snap)
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
            hold_ms = int(data.get("hold_ms") or (400 if mode == "hold" else 80))
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
            if mode in {"tap", "hold", "down"}:
                TELEMETRY.on_keys_pressed(keys, label)
        except (ValueError, OSError) as exc:
            log(f"FEHLER {mode} {label}: {exc}")
            self._json(500, {"error": str(exc)})
            return
        if mode != "down":
            log(f"{mode:<6} {label}  {'+'.join(keys)}")
        self._json(200, {"ok": True, "telemetry": TELEMETRY.snapshot()})


def serve(host: str = "0.0.0.0") -> None:
    items = catalog()
    if not items:
        raise SystemExit("Keine Dashboards in dashboards/ gefunden.")
    for item in items:
        for err in validate_layout(item["data"]):
            log(f"{item['id']}: {err}")
    try:
        server = ThreadingHTTPServer((host, CONFIG["port"]), Handler)
    except OSError as exc:
        log(f"Port {CONFIG['port']} ist belegt oder gesperrt ({exc}).")
        raise SystemExit(1) from exc
    log(f"Server lauscht auf {host}:{CONFIG['port']}")
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
    sample_dir = tmp / "dashboards"
    sample_dir.mkdir(parents=True, exist_ok=True)
    retired_power = "power"
    retired_shield = "shield"
    (sample_dir / "old-power-widget.json").write_text(json.dumps({
        "id": "old-power-widget",
        "title": "Altes Power-Widget",
        "pages": [{
            "id": "energie",
            "label": "Alt",
            "groups": [{
                "title": "Alt",
                "buttons": [
                    {"id": "gear", "label": "Fahrwerk", "keys": ["n"], "mode": "toggle", "color": "cyan"},
                    {"id": "live-pwr", "kind": retired_power, "label": "Energie-Kondensator", "hint": "F5 / F6 / F7 / F8", "keys": ["f8"], "mode": "tap", "color": "amber", "wide": True},
                    {"id": "live-shd", "kind": retired_shield, "label": "Schild-Matrix", "hint": "Sektoren", "keys": ["f8"], "mode": "tap", "color": "cyan", "wide": True},
                ],
            }],
        }],
    }, ensure_ascii=False), encoding="utf-8")
    items = catalog()
    if not any(item["id"] == "aegis-gladius" for item in items):
        log("aegis-gladius Dashboard fehlt")
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
        if len(ids) < 95 or "aegis-gladius" not in ids or "drake-vulture" not in ids or "misc-prospector" not in ids:
            raise SystemExit(f"Dashboards unvollständig ({len(ids)}): {ids}")
        fav_res = call("POST", "/api/favorites", {"id": "aegis-eclipse", "favorite": True})
        if not fav_res.get("favorite") or "aegis-eclipse" not in fav_res.get("favorites", []):
            raise SystemExit(f"Favoriten-Speicherung fehlgeschlagen: {fav_res}")
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
        call("POST", "/api/active", {"id": "misc-prospector"})
        if active_id() != "misc-prospector":
            raise SystemExit("aktives Dashboard nicht gesetzt")
        call("DELETE", "/api/dashboard/selftest")
        if find_dashboard("selftest") is not None:
            raise SystemExit("Dashboard nicht gelöscht")
        call("POST", "/api/press", {"keys": ["n"], "mode": "tap", "id": "test"})
        tele = call("GET", "/api/telemetry")
        if "power" in tele or "shields" in tele:
            raise SystemExit("Power-Dreieck-Telemetrie noch vorhanden")
        if "systems" not in tele or "sc_running" not in tele or "sc_ship" not in tele:
            raise SystemExit("telemetry unvollständig")
        old = call("GET", "/api/dashboard/old-power-widget")
        if old.get("id") != "old-power-widget":
            raise SystemExit("altes Power-Widget-Dashboard nicht ladbar")
        glad = find_dashboard("aegis-gladius")
        energie = next(p for p in glad["data"]["pages"] if p["id"] == "energie")
        if energie.get("label") != "Power Management":
            raise SystemExit(f"energie-Seite heißt {energie.get('label')!r}")
        for group in energie["groups"]:
            for button in group["buttons"]:
                if button.get("kind") in {"power", "shield"}:
                    raise SystemExit("energie-Seite hat noch ein Live-Widget")
        with urllib.request.urlopen(base + "/") as res:
            html = res.read().decode()
        if "Baukasten" not in html:
            raise SystemExit("index.html ohne Baukasten-Editor")
    finally:
        server.shutdown()
        server.server_close()
        os.environ.pop("SC_COCKPIT_HOME", None)
        shutil.rmtree(tmp, ignore_errors=True)
    test_update_decisions()
    log(f"self-test ok ({len(items)} dashboards, dry_run={DRY_RUN})")
    return 0


def main() -> None:
    try:
        sys.stdout.reconfigure(encoding="utf-8")
    except Exception:
        pass
    if "--github-update" in sys.argv or "--github-check" in sys.argv:
        raise SystemExit(run_github_update_cli(apply="--github-check" not in sys.argv))
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
