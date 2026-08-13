"""Readable, dependency-free reference implementations of decay scores."""

from __future__ import annotations

import math

MIN_DURATION_MILLIS = 1_000
MAX_DURATION_MILLIS = 315_576_000_000


def exponential_score(
    *, last_accessed: int, now: int, half_life_millis: int
) -> float:
    """Return normalized exponential retention from Unix-epoch milliseconds."""
    if half_life_millis <= 0:
        raise ValueError("half_life_millis must be positive")
    if not MIN_DURATION_MILLIS <= half_life_millis <= MAX_DURATION_MILLIS:
        raise ValueError(
            "half_life_millis must be between "
            f"{MIN_DURATION_MILLIS} and {MAX_DURATION_MILLIS}"
        )
    elapsed = max(0, now - last_accessed)
    return math.exp2(-elapsed / half_life_millis)


def importance_weighted_power_law_score(
    *,
    last_accessed: int,
    now: int,
    scale_millis: int,
    exponent: float,
    importance: float,
) -> float:
    """Return importance-weighted power-law retention."""
    if scale_millis <= 0:
        raise ValueError("scale_millis must be positive")
    if not MIN_DURATION_MILLIS <= scale_millis <= MAX_DURATION_MILLIS:
        raise ValueError(
            "scale_millis must be between "
            f"{MIN_DURATION_MILLIS} and {MAX_DURATION_MILLIS}"
        )
    if not math.isfinite(exponent) or exponent <= 0:
        raise ValueError("exponent must be finite and positive")
    if not 0.1 <= exponent <= 10.0:
        raise ValueError("exponent must be between 0.1 and 10.0")
    if not math.isfinite(importance) or not 0 <= importance <= 1:
        raise ValueError("importance must be finite and in [0, 1]")
    elapsed = max(0, now - last_accessed)
    return importance * math.exp(-exponent * math.log1p(elapsed / scale_millis))
