"""Typed ctypes bindings for the fixed-width memory-decay C ABI."""

from __future__ import annotations

import ctypes
from pathlib import Path
from typing import Callable, Protocol


class _Loader(Protocol):
    def __call__(self, name: str) -> object: ...


class NativeDecayLibrary:
    """A loaded native decay library with Pythonic score methods."""

    def __init__(self, path: Path, loader: _Loader = ctypes.CDLL) -> None:
        self._library = loader(str(path))
        self._exponential = self._library.DecayScoreExponential
        self._exponential.argtypes = [
            ctypes.c_int64,
            ctypes.c_int64,
            ctypes.c_int64,
            ctypes.POINTER(ctypes.c_double),
        ]
        self._exponential.restype = ctypes.c_int32

        self._power_law = self._library.DecayScorePowerLaw
        self._power_law.argtypes = [
            ctypes.c_int64,
            ctypes.c_int64,
            ctypes.c_int64,
            ctypes.c_double,
            ctypes.c_double,
            ctypes.POINTER(ctypes.c_double),
        ]
        self._power_law.restype = ctypes.c_int32

    def exponential_score(
        self, *, last_accessed: int, now: int, half_life_millis: int
    ) -> float:
        """Return an exponential score through the native ABI."""
        score = ctypes.c_double()
        self._raise_for_status(
            self._exponential(
                last_accessed,
                now,
                half_life_millis,
                ctypes.byref(score),
            )
        )
        return score.value

    def power_law_score(
        self,
        *,
        last_accessed: int,
        now: int,
        scale_millis: int,
        exponent: float,
        importance: float,
    ) -> float:
        """Return an importance-weighted power-law score through the native ABI."""
        score = ctypes.c_double()
        self._raise_for_status(
            self._power_law(
                last_accessed,
                now,
                scale_millis,
                exponent,
                importance,
                ctypes.byref(score),
            )
        )
        return score.value

    @staticmethod
    def _raise_for_status(status: int) -> None:
        if status == 0:
            return
        if status == 1:
            raise ValueError("native decay library received invalid arguments")
        if status == 2:
            raise RuntimeError("native decay library received a null output pointer")
        raise RuntimeError(f"native decay library returned unknown status {status}")
