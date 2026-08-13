import ctypes
from pathlib import Path
from typing import List, Optional

import pytest

from recolva._ffi import NativeDecayLibrary


class FakeFunction:
    def __init__(self, *, status: int, score: Optional[float] = None) -> None:
        self.status = status
        self.score = score
        self.argtypes: Optional[List[object]] = None
        self.restype: Optional[object] = None

    def __call__(self, *args: object) -> int:
        if self.score is not None:
            out_score = args[-1]
            out_score._obj.value = self.score  # type: ignore[attr-defined]
        return self.status


class FakeLibrary:
    def __init__(self, exponential: FakeFunction, power_law: FakeFunction) -> None:
        self.DecayScoreExponential = exponential
        self.DecayScorePowerLaw = power_law


def test_native_library_calls_fixed_width_exponential_abi() -> None:
    exponential = FakeFunction(status=0, score=0.5)
    library = FakeLibrary(exponential, FakeFunction(status=0, score=0.25))
    bridge = NativeDecayLibrary(Path("unused.dll"), loader=lambda _: library)

    assert bridge.exponential_score(
        last_accessed=1_000_000,
        now=87_400_000,
        half_life_millis=86_400_000,
    ) == 0.5
    assert exponential.argtypes == [
        ctypes.c_int64,
        ctypes.c_int64,
        ctypes.c_int64,
        ctypes.POINTER(ctypes.c_double),
    ]
    assert exponential.restype is ctypes.c_int32


def test_native_library_calls_fixed_width_power_law_abi() -> None:
    power_law = FakeFunction(status=0, score=0.25)
    library = FakeLibrary(FakeFunction(status=0, score=0.5), power_law)
    bridge = NativeDecayLibrary(Path("unused.dll"), loader=lambda _: library)

    assert bridge.power_law_score(
        last_accessed=1_000_000,
        now=87_400_000,
        scale_millis=86_400_000,
        exponent=1.0,
        importance=0.5,
    ) == 0.25
    assert power_law.argtypes == [
        ctypes.c_int64,
        ctypes.c_int64,
        ctypes.c_int64,
        ctypes.c_double,
        ctypes.c_double,
        ctypes.POINTER(ctypes.c_double),
    ]
    assert power_law.restype is ctypes.c_int32


def test_native_library_maps_invalid_argument_status_to_value_error() -> None:
    library = FakeLibrary(
        FakeFunction(status=1),
        FakeFunction(status=0, score=0.25),
    )
    bridge = NativeDecayLibrary(Path("unused.dll"), loader=lambda _: library)

    with pytest.raises(ValueError, match="invalid arguments"):
        bridge.exponential_score(
            last_accessed=1_000_000,
            now=1_000_000,
            half_life_millis=999,
        )
