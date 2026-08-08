import pytest

import decay
from decay._native import NativeLibraryUnavailableError


class FakeNativeLibrary:
    def exponential_score(self, **kwargs: object) -> float:
        assert kwargs == {
            "last_accessed": 1_000_000,
            "now": 87_400_000,
            "half_life_millis": 86_400_000,
        }
        return 0.5

    def power_law_score(self, **kwargs: object) -> float:
        assert kwargs == {
            "last_accessed": 1_000_000,
            "now": 87_400_000,
            "scale_millis": 86_400_000,
            "exponent": 1.0,
            "importance": 0.5,
        }
        return 0.25


def test_public_exponential_score_uses_native_library(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(decay, "_native_library", FakeNativeLibrary())

    assert decay.exponential_score(
        last_accessed=1_000_000,
        now=87_400_000,
        half_life_millis=86_400_000,
    ) == 0.5


def test_public_power_law_score_uses_native_library(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(decay, "_native_library", FakeNativeLibrary())

    assert decay.power_law_score(
        last_accessed=1_000_000,
        now=87_400_000,
        scale_millis=86_400_000,
        exponent=1.0,
        importance=0.5,
    ) == 0.25


def test_public_exponential_score_falls_back_when_native_library_is_unavailable(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    def unavailable() -> object:
        raise NativeLibraryUnavailableError("native library unavailable")

    monkeypatch.setattr(decay, "_get_native_library", unavailable)

    assert decay.exponential_score(
        last_accessed=1_000_000,
        now=87_400_000,
        half_life_millis=86_400_000,
    ) == 0.5


def test_public_power_law_score_falls_back_when_native_library_is_unavailable(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    def unavailable() -> object:
        raise NativeLibraryUnavailableError("native library unavailable")

    monkeypatch.setattr(decay, "_get_native_library", unavailable)

    assert decay.power_law_score(
        last_accessed=1_000_000,
        now=87_400_000,
        scale_millis=86_400_000,
        exponent=1.0,
        importance=0.5,
    ) == 0.25
