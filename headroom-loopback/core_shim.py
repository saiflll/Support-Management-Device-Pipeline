"""
Fallback shim untuk headroom._core

Digunakan sebagai fallback Python module ketika extension Rust (.so) 
gagal dimuat, misalnya karena SIGILL di CPU KVM lawas yang tidak mendukung
instruksi modern (AVX2, dll).

Mengembalikan nilai default yang aman agar aplikasi tidak crash.
"""
from __future__ import annotations

import dataclasses
from typing import Any

# ==========================================
# error_detection.py
# ==========================================

def keyword_registry_snapshot() -> dict:
    # Mengembalikan snapshot kosong
    return {}

def content_has_error_indicators(text: str) -> bool:
    # Asumsikan tidak ada error jika tidak bisa dideteksi
    return False

def score_line(line: str, context: str = "text") -> tuple[str | None, float, float]:
    # Return format: (reason, score, threshold)
    return (None, 0.0, 0.0)

# ==========================================
# tag_protector.py
# ==========================================

def is_html_tag(tag: str) -> bool:
    return False

def known_html_tag_names() -> frozenset:
    # Harus return set atau frozenset kosong
    return frozenset()

def protect_tags(text: str) -> str:
    # Lewati teks tanpa perubahan
    return text

def restore_tags(text: str) -> str:
    # Lewati teks tanpa perubahan
    return text

# ==========================================
# text_crusher.py & SmartCrusher
# ==========================================

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

# ==========================================
# Utils & Content Detection
# ==========================================

def detect_content_type(content: str) -> str:
    return "text"

# ==========================================
# Log & Search Compressors
# ==========================================

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

# ==========================================
# DiffCompressor
# ==========================================

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
