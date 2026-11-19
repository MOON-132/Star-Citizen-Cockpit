#!/usr/bin/env python3
"""
Builds the standalone Windows installer executable `SC-Cockpit-Setup.exe`
with a Star Citizen MFD icon, Win32 DPI-aware manifest, and embedded payload.zip.
Can be run on Linux (cross-compiling to Windows amd64) or Windows.
"""

import math
import os
import struct
import subprocess
import zipfile
import zlib
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
INSTALLER_DIR = ROOT / "installer"
OUTPUT_EXE = ROOT / "SC-Cockpit-Setup.exe"


def make_png_rgba(size: int) -> bytes:
    """Render a crisp Star Citizen MFD-styled icon (Amber & Cyan HUD reticle on #0b0e13)."""
    rows = []
    cx = (size - 1) / 2.0
    cy = (size - 1) / 2.0
    half = size / 2.0
    cut = size * 0.20  # Chamfered octagonal corners

    for y in range(size):
        row = bytearray([0])  # PNG filter type 0
        for x in range(size):
            dx = abs(x - cx)
            dy = abs(y - cy)

            # Octagonal boundary check
            dist_edge = max(dx, dy)
            dist_diag = (dx + dy) / 1.41421356
            outer = half - 1.5
            diag_outer = (size * 0.63)

            if dist_edge > outer or (dx + dy) > (size - cut):
                row.extend((0, 0, 0, 0))
                continue

            # Base background #0b0e13
            r, g, b, a = 0x0B, 0x0E, 0x13, 255

            # Outer chamfer border (Amber #e2a23a)
            border_dist = min(outer - dist_edge, (size - cut) - (dx + dy))
            if border_dist < max(1.8, size * 0.055):
                r, g, b = 0xE2, 0xA2, 0x3A
            else:
                # Inner tactical HUD ring (Cyan #3ec8c4)
                rad = math.hypot(x - cx, y - cy)
                ring_r = size * 0.31
                ring_w = max(1.3, size * 0.045)
                if abs(rad - ring_r) <= ring_w and not (dx < size * 0.08 or dy < size * 0.08):
                    r, g, b = 0x3E, 0xC8, 0xC4

                # Center quantum diamond / amber core
                diamond = dx + dy
                if diamond <= size * 0.16:
                    r, g, b = 0xE2, 0xA2, 0x3A
                elif abs(diamond - size * 0.21) < max(1.1, size * 0.03):
                    r, g, b = 0x8A, 0x62, 0x20

                # Left vertical Star Citizen Cockpit brand bar accent
                if (size * 0.13 <= x <= size * 0.21) and (size * 0.24 <= y <= size * 0.76):
                    r, g, b = 0xE2, 0xA2, 0x3A

            row.extend((r, g, b, a))
        rows.append(bytes(row))

    raw = b"".join(rows)

    def chunk(tag: bytes, data: bytes) -> bytes:
        crc = zlib.crc32(tag + data) & 0xFFFFFFFF
        return struct.pack(">I", len(data)) + tag + data + struct.pack(">I", crc)

    ihdr = struct.pack(">IIBBBBB", size, size, 8, 6, 0, 0, 0)
    idat = zlib.compress(raw, 9)
    return b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", ihdr) + chunk(b"IDAT", idat) + chunk(b"IEND", b"")


def build_ico(path: Path) -> None:
    sizes = [16, 32, 48, 64, 256]
    pngs = [make_png_rgba(s) for s in sizes]
    header = struct.pack("<HHH", 0, 1, len(sizes))
    offset = 6 + 16 * len(sizes)
    entries = []
    for s, data in zip(sizes, pngs):
        w_byte = 0 if s == 256 else s
        entries.append(struct.pack("<BBBBHHII", w_byte, w_byte, 0, 0, 1, 32, len(data), offset))
        offset += len(data)
    path.write_bytes(header + b"".join(entries) + b"".join(pngs))


def build_payload_zip(ico_path: Path, zip_path: Path) -> None:
    top_files = [
        "server.py",
        "index.html",
        "LICENSE",
        "README.md",
        "config.local.json.example",
        "install.ps1",
        "install.bat",
        "uninstall.ps1",
        "uninstall.bat",
    ]
    with zipfile.ZipFile(zip_path, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as zf:
        for name in top_files:
            src = ROOT / name
            if src.exists():
                zf.write(src, arcname=name)

        # Include custom SC icon so desktop/start menu shortcuts get the SC icon
        zf.write(ico_path, arcname="sc-cockpit.ico")

        for folder_name in ("dashboards", "docs"):
            folder = ROOT / folder_name
            for p in sorted(folder.rglob("*")):
                if p.is_dir():
                    continue
                rel = p.relative_to(ROOT).as_posix()
                # Skip user-local dashboards in dashboards/local/
                if rel.startswith("dashboards/local/"):
                    continue
                zf.write(p, arcname=rel)


def build_coff_syso(ico_path: Path, syso_path: Path) -> None:
    """
    Generates a Windows x86_64 COFF (.syso) file containing a `.rsrc` section with:
      - RT_ICON (3) & RT_GROUP_ICON (14) so the .exe has the Star Citizen HUD icon
      - RT_MANIFEST (24) with Per-Monitor DPI awareness and asInvoker execution level
    """
    ico_bytes = ico_path.read_bytes()
    _, _, count = struct.unpack_from("<HHH", ico_bytes, 0)
    icon_blobs = []
    grp_entries = []
    for i in range(count):
        w, h, cc, rsv, planes, bitcnt, size, off = struct.unpack_from("<BBBBHHII", ico_bytes, 6 + i * 16)
        icon_blobs.append(ico_bytes[off : off + size])
        grp_entries.append(struct.pack("<BBBBHHIH", w, h, cc, rsv, planes, bitcnt, size, i + 1))

    grp_icon_data = struct.pack("<HHH", 0, 1, count) + b"".join(grp_entries)

    manifest_xml = b"""<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0" xmlns:asmv3="urn:schemas-microsoft-com:asm.v3">
  <assemblyIdentity version="0.2.0.0" processorArchitecture="amd64" name="MOON132.SCCockpit.Setup" type="win32"/>
  <description>SC Cockpit Star Citizen Touch Dashboard Installer</description>
  <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3">
    <security>
      <requestedPrivileges>
        <requestedExecutionLevel level="asInvoker" uiAccess="false"/>
      </requestedPrivileges>
    </security>
  </trustInfo>
  <asmv3:application>
    <asmv3:windowsSettings>
      <dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">true/pm</dpiAware>
      <dpiAwareness xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">PerMonitorV2,PerMonitor</dpiAwareness>
    </asmv3:windowsSettings>
  </asmv3:application>
</assembly>"""

    # Resource Types: 3 (RT_ICON), 14 (RT_GROUP_ICON), 24 (RT_MANIFEST)
    # Build .rsrc directory tree:
    # Level 1 (Root): 3 entries (Type 3, Type 14, Type 24)
    # Level 2 (IDs):
    #   - Type 3 has `count` entries (IDs 1..count)
    #   - Type 14 has 1 entry (ID 1)
    #   - Type 24 has 1 entry (ID 1)
    # Level 3 (Langs):
    #   - count + 2 language directories (each with 1 entry, lang 0x0409)
    # Data Entries: count + 2 (16 bytes each)

    total_items = count + 2
    lvl1_size = 16 + 3 * 8
    lvl2_icon_size = 16 + count * 8
    lvl2_grp_size = 16 + 1 * 8
    lvl2_man_size = 16 + 1 * 8
    lvl3_total_size = total_items * (16 + 1 * 8)
    data_entries_size = total_items * 16

    off_lvl1 = 0
    off_lvl2_icon = off_lvl1 + lvl1_size
    off_lvl2_grp = off_lvl2_icon + lvl2_icon_size
    off_lvl2_man = off_lvl2_grp + lvl2_grp_size
    off_lvl3 = off_lvl2_man + lvl2_man_size
    off_data_entries = off_lvl3 + lvl3_total_size
    off_blobs = (off_data_entries + data_entries_size + 7) & ~7

    HI = 0x80000000
    rsrc = bytearray()

    def dir_hdr(num_id: int) -> bytes:
        return struct.pack("<IIHHHH", 0, 0, 0, 0, 0, num_id)

    def dir_ent(id_val: int, target_off: int, is_subdir: bool = True) -> bytes:
        return struct.pack("<II", id_val, (target_off | HI) if is_subdir else target_off)

    # Level 1
    rsrc += dir_hdr(3)
    rsrc += dir_ent(3, off_lvl2_icon, True)
    rsrc += dir_ent(14, off_lvl2_grp, True)
    rsrc += dir_ent(24, off_lvl2_man, True)

    # Level 2: RT_ICON (1..count)
    rsrc += dir_hdr(count)
    for i in range(count):
        rsrc += dir_ent(i + 1, off_lvl3 + i * 24, True)

    # Level 2: RT_GROUP_ICON (ID 1)
    rsrc += dir_hdr(1)
    rsrc += dir_ent(1, off_lvl3 + count * 24, True)

    # Level 2: RT_MANIFEST (ID 1)
    rsrc += dir_hdr(1)
    rsrc += dir_ent(1, off_lvl3 + (count + 1) * 24, True)

    # Level 3: Language dirs
    for i in range(total_items):
        rsrc += dir_hdr(1)
        rsrc += dir_ent(0x0409, off_data_entries + i * 16, False)

    # Prepare blobs and their offsets
    all_blobs = icon_blobs + [grp_icon_data, manifest_xml]
    blob_area = bytearray()
    data_entries = bytearray()
    relocs = []

    cur_blob_off = off_blobs
    for idx, bdata in enumerate(all_blobs):
        de_off = off_data_entries + idx * 16
        relocs.append(de_off)  # RVA field at start of IMAGE_RESOURCE_DATA_ENTRY
        data_entries += struct.pack("<IIII", cur_blob_off, len(bdata), 0, 0)
        padded = bdata + b"\x00" * ((8 - (len(bdata) % 8)) % 8)
        blob_area += padded
        cur_blob_off += len(padded)

    rsrc += data_entries
    while len(rsrc) < off_blobs:
        rsrc.append(0)
    rsrc += blob_area

    # Wrap in COFF x86_64 object (.syso)
    # COFF Header (20 bytes) + 1 Section Header (40 bytes) + raw rsrc + relocs + 1 symbol (18 bytes) + string table (4 bytes)
    raw_ptr = 20 + 40
    reloc_ptr = raw_ptr + len(rsrc)
    sym_ptr = reloc_ptr + len(relocs) * 10

    coff_hdr = struct.pack(
        "<HHIIIHH",
        0x8664,        # IMAGE_FILE_MACHINE_AMD64
        1,             # NumberOfSections
        0,             # TimeDateStamp
        sym_ptr,       # PointerToSymbolTable
        1,             # NumberOfSymbols
        0,             # SizeOfOptionalHeader
        0,             # Characteristics
    )

    sec_hdr = struct.pack(
        "<8sIIIIIIHHI",
        b".rsrc\x00\x00\x00",
        0,             # VirtualSize
        0,             # VirtualAddress
        len(rsrc),     # SizeOfRawData
        raw_ptr,       # PointerToRawData
        reloc_ptr,     # PointerToRelocations
        0,             # PointerToLinenumbers
        len(relocs),   # NumberOfRelocations
        0,             # NumberOfLinenumbers
        0x40000040,    # IMAGE_SCN_CNT_INITIALIZED_DATA | IMAGE_SCN_MEM_READ
    )

    reloc_bytes = bytearray()
    for r_off in relocs:
        # Type 3 = IMAGE_REL_AMD64_ADDR32NB, SymbolTableIndex = 0
        reloc_bytes += struct.pack("<IIH", r_off, 0, 3)

    sym_table = struct.pack("<8sIhHBB", b".rsrc\x00\x00\x00", 0, 1, 0, 3, 0)
    str_table = struct.pack("<I", 4)

    syso_path.write_bytes(coff_hdr + sec_hdr + bytes(rsrc) + bytes(reloc_bytes) + sym_table + str_table)


def main() -> None:
    ico_path = INSTALLER_DIR / "sc-cockpit.ico"
    zip_path = INSTALLER_DIR / "payload.zip"
    syso_path = INSTALLER_DIR / "rsrc_windows_amd64.syso"

    print("[1/4] Generiere Star Citizen MFD App-Icon (sc-cockpit.ico)...")
    build_ico(ico_path)

    print("[2/4] Packe Anwendungsdateien & Dashboards in payload.zip...")
    build_payload_zip(ico_path, zip_path)

    print("[3/4] Erzeuge Win32 COFF Resource-Objekt (.syso mit Icon & DPI-Manifest)...")
    build_coff_syso(ico_path, syso_path)

    print(f"[4/4] Kompiliere nativen Windows-Installer -> {OUTPUT_EXE.name}...")
    env = os.environ.copy()
    env["GOOS"] = "windows"
    env["GOARCH"] = "amd64"
    env["CGO_ENABLED"] = "0"

    subprocess.run(
        [
            "go",
            "build",
            "-trimpath",
            "-ldflags=-H=windowsgui -s -w",
            "-o",
            str(OUTPUT_EXE),
            ".",
        ],
        cwd=str(INSTALLER_DIR),
        env=env,
        check=True,
    )

    size_kb = OUTPUT_EXE.stat().st_size / 1024.0
    print(f"Fertig! {OUTPUT_EXE} ({size_kb:.1f} KB)")


if __name__ == "__main__":
    main()
