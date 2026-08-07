import pytest

from decay.reference import exponential_score


def test_exponential_score_is_one_at_last_access() -> None:
    assert exponential_score(
        last_accessed=1_000_000,
        now=1_000_000,
        half_life_seconds=86_400,
    ) == 1.0


def test_exponential_score_halves_at_half_life() -> None:
    assert exponential_score(
        last_accessed=1_000_000,
        now=1_086_400,
        half_life_seconds=86_400,
    ) == 0.5


def test_exponential_score_clamps_future_access_to_one() -> None:
    assert exponential_score(
        last_accessed=1_086_400,
        now=1_000_000,
        half_life_seconds=86_400,
    ) == 1.0


def test_exponential_score_rejects_non_positive_half_life() -> None:
    with pytest.raises(ValueError, match="half_life_seconds must be positive"):
        exponential_score(
            last_accessed=1_000_000,
            now=1_000_000,
            half_life_seconds=0,
        )
