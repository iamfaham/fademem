"""Readable, dependency-free reference implementations of decay scores."""

from __future__ import annotations

import math


def exponential_score(
    *, last_accessed: int, now: int, half_life_seconds: int
) -> float:
    """Return normalized exponential retention."""
    if half_life_seconds <= 0:
        raise ValueError("half_life_seconds must be positive")
    elapsed = max(0, now - last_accessed)
    return math.exp2(-elapsed / half_life_seconds)
