#!/usr/bin/env python3
"""
Headroom Pure Python Setup

Mengganti extension Rust headroom._core (.so) dengan shim Python murni
biar gak crash (SIGILL) di CPU KVM lawas yang kurang AVX2.
"""

import os
import shutil
import subprocess
import sys
import venv

HERE = os.path.dirname(os.path.abspath(__file__))

VENV_DIR = os.path.expanduser("~/headroom-env")
ACTIVATE = os.path.join(VENV_DIR, "bin", "activate")
PYTHON = os.path.join(VENV_DIR, "bin", "python")


def run(cmd, **kwargs):
    print(f"$ {cmd}", flush=True)
    subprocess.check_call(cmd, shell=True, **kwargs)


def main():
    # 1. Stop docker container
    print("=== Stop container headroom-nginx ===")
    subprocess.call("docker stop headroom-nginx 2>/dev/null", shell=True)
    subprocess.call("docker rm headroom-nginx 2>/dev/null", shell=True)

    # 2. Buat venv kalo belum ada
    if not os.path.isdir(VENV_DIR):
        print(f"\n=== Buat venv di {VENV_DIR} ===")
        venv.create(VENV_DIR, with_pip=True, symlinks=False)
    else:
        print(f"venv udah ada di {VENV_DIR}, skip")

    # 3. Install / upgrade headroom
    print("\n=== Install headroom ===")
    run(f"{PYTHON} -m pip install --upgrade pip")
    run(f"{PYTHON} -m pip install --upgrade headroom")

    # 4. Cari lokasi package headroom
    print("\n=== Cari package headroom ===")
    result = subprocess.check_output(
        [
            PYTHON,
            "-c",
            "import importlib.util; s=importlib.util.find_spec('headroom');"
            "p=(s.submodule_search_locations or [''])[0]; print(p)",
        ],
        text=True,
    ).strip()
    pkg_dir = result
    print(f"Headroom package dir: {pkg_dir}")

    core_dir = os.path.join(pkg_dir, "_core")
    print(f"Core dir: {core_dir}")

    # 5. Hapus .so
    print("\n=== Hapus _core*.so ===")
    for f in os.listdir(pkg_dir):
        if f.startswith("_core") and f.endswith(".so"):
            path = os.path.join(pkg_dir, f)
            os.remove(path)
            print(f"  Deleted: {path}")

    # 6. Hapus folder _core kalo ada, bikin ulang
    if os.path.isdir(core_dir):
        shutil.rmtree(core_dir)
        print(f"  Removed existing _core/ dir")

    os.makedirs(core_dir, exist_ok=True)

    # 7. Tulis shim
    print("\n=== Tulis core shim ===")
    shim_path = os.path.join(core_dir, "__init__.py")

    shim_code = r'''"""
Fallback shim untuk headroom._core

Digunakan sebagai fallback Python module ketika extension Rust (.so)
gagal dimuat (mis. SIGILL di CPU KVM lawas).
"""
from __future__ import annotations

import dataclasses
from typing import Any


# error_detection.py
def keyword_registry_snapshot() -> dict:
    return {}

def content_has_error_indicators(text: str) -> bool:
    return False

def score_line(line: str, context: str = "text") -> tuple[str | None, float, float]:
    return (None, 0.0, 0.0)


# tag_protector.py
def is_html_tag(tag: str) -> bool:
    return False

def known_html_tag_names() -> frozenset:
    return frozenset()

def protect_tags(text: str) -> str:
    return text

def restore_tags(text: str) -> str:
    return text


# text_crusher.py
@dataclasses.dataclass
class TextCrusherConfig:
    pass

class TextCrusher:
    def __init__(self, config: TextCrusherConfig | None = None) -> None:
        pass
    def __repr__(self) -> str:
        return "<TextCrusher (Fallback Shim)>"
    def __call__(self, *args: Any, **kwargs: Any) -> Any:
        if args and isinstance(args[0], str):
            return args[0]
        return ""


# SmartCrusher
@dataclasses.dataclass
class SmartCrusherConfig:
    pass

class SmartCrusher:
    def __init__(self, config: SmartCrusherConfig | None = None) -> None:
        pass
    def __repr__(self) -> str:
        return "<SmartCrusher (Fallback Shim)>"
    def __call__(self, *args: Any, **kwargs: Any) -> Any:
        if args and isinstance(args[0], str):
            return args[0]
        return ""


# Content detection
def detect_content_type(content: str) -> str:
    return "text"


# LogCompressor
@dataclasses.dataclass
class LogCompressorConfig:
    pass

class LogCompressor:
    def __init__(self, config: LogCompressorConfig | None = None) -> None:
        pass
    def __repr__(self) -> str:
        return "<LogCompressor (Fallback Shim)>"
    def __call__(self, *args: Any, **kwargs: Any) -> Any:
        if args and isinstance(args[0], str):
            return args[0]
        return ""

def detect_log_format(*args: Any, **kwargs: Any) -> str:
    return "unknown"


# SearchCompressor
@dataclasses.dataclass
class SearchCompressorConfig:
    pass

class SearchCompressor:
    def __init__(self, config: SearchCompressorConfig | None = None) -> None:
        pass
    def __repr__(self) -> str:
        return "<SearchCompressor (Fallback Shim)>"
    def __call__(self, *args: Any, **kwargs: Any) -> Any:
        if args and isinstance(args[0], str):
            return args[0]
        return ""

def parse_search_lines(*args: Any, **kwargs: Any) -> list:
    return []


# DiffCompressor
@dataclasses.dataclass
class DiffCompressorConfig:
    pass

class DiffCompressor:
    def __init__(self, config: DiffCompressorConfig | None = None) -> None:
        pass
    def __repr__(self) -> str:
        return "<DiffCompressor (Fallback Shim)>"
    def __call__(self, *args: Any, **kwargs: Any) -> Any:
        if args and isinstance(args[0], str):
            return args[0]
        return ""
'''

    with open(shim_path, "w") as f:
        f.write(shim_code)
    print(f"  Written: {shim_path}")

    # 8. Set HEADROOM_REQUIRE_RUST_CORE=false di activate script
    print("\n=== Set env var di activate script ===")
    marker = "# HEADROOM_PURE_PYTHON"
    line = f'export HEADROOM_REQUIRE_RUST_CORE=false  {marker}'
    with open(ACTIVATE, "r") as f:
        content = f.read()
    if marker not in content:
        content += "\n" + line + "\n"
        with open(ACTIVATE, "w") as f:
            f.write(content)
        print(f"  Added to {ACTIVATE}")

    # 9. Juga set di .profile biar kalo login langsung
    profile = os.path.expanduser("~/.profile")
    with open(profile, "a") as f:
        f.write(f"\n# Headroom pure python mode\nexport HEADROOM_REQUIRE_RUST_CORE=false\n")

    # 10. Setup systemd user service untuk auto-start
    print("\n=== Setup systemd user service ===")
    systemd_dir = os.path.expanduser("~/.config/systemd/user")
    os.makedirs(systemd_dir, exist_ok=True)

    service_path = os.path.join(systemd_dir, "headroom-proxy.service")
    service_content = f"""[Unit]
Description=Headroom Proxy (Pure Python Mode)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
Environment=HEADROOM_REQUIRE_RUST_CORE=false
ExecStart={VENV_DIR}/bin/headroom proxy --port 8787 --host 0.0.0.0
Restart=always
RestartSec=5

[Install]
WantedBy=default.target
"""
    with open(service_path, "w") as f:
        f.write(service_content)
    print(f"  Created: {service_path}")

    # Reload systemd user daemon & enable
    subprocess.call(["systemctl", "--user", "daemon-reload"], stderr=subprocess.DEVNULL)
    result = subprocess.call(["systemctl", "--user", "enable", "headroom-proxy.service"],
                             stderr=subprocess.DEVNULL)
    if result == 0:
        print("  systemd user service enabled ✅")
        print("  Mulai sekarang: systemctl --user start headroom-proxy")
        print("  Cek status   : systemctl --user status headroom-proxy")
        print("  Lihat log    : journalctl --user -u headroom-proxy -f")
    else:
        print("  ⚠️  systemd user service gagal di-enable (mungkin bukan Linux/systemd)")
        print("  Manual: systemctl --user enable headroom-proxy.service")

    print(f"\n{'='*50}")
    print("✅ Selesai!")
    print(f"   Venv    : {VENV_DIR}")
    print(f"   Aktifkan: source {ACTIVATE}")
    print(f"   Jalankan: headroom proxy --port 8787 --host 0.0.0.0")
    print(f"   Auto-start: systemctl --user start headroom-proxy")
    print(f"   Auto-restart kalau mati / reboot: otomatis (systemd)")
    print(f"{'='*50}")


if __name__ == "__main__":
    main()
