"""Local memory-decay scoring through the packaged native library."""

from __future__ import annotations

from typing import Optional

from ._ffi import NativeDecayLibrary
from ._native import bundled_library_path

_native_library: Optional[NativeDecayLibrary] = None


def _get_native_library() -> NativeDecayLibrary:
    global _native_library
    if _native_library is None:
        _native_library = NativeDecayLibrary(bundled_library_path())
    return _native_library


def exponential_score(
    *, last_accessed: int, now: int, half_life_millis: int
) -> float:
    """Return an exponential retention score through the native library."""
    return _get_native_library().exponential_score(
        last_accessed=last_accessed,
        now=now,
        half_life_millis=half_life_millis,
    )


def power_law_score(
    *,
    last_accessed: int,
    now: int,
    scale_millis: int,
    exponent: float,
    importance: float,
) -> float:
    """Return an importance-weighted power-law score through the native library."""
    return _get_native_library().power_law_score(
        last_accessed=last_accessed,
        now=now,
        scale_millis=scale_millis,
        exponent=exponent,
        importance=importance,
    )


__all__ = ["exponential_score", "power_law_score"]
