import pytest

from recolva.reference import (
    exponential_score,
    importance_weighted_power_law_score,
)

ONE_DAY_MILLIS = 86_400_000


def test_exponential_score_is_one_at_last_access() -> None:
    assert exponential_score(
        last_accessed=1_000_000,
        now=1_000_000,
        half_life_millis=ONE_DAY_MILLIS,
    ) == 1.0


def test_exponential_score_halves_at_half_life() -> None:
    assert exponential_score(
        last_accessed=1_000_000,
        now=87_400_000,
        half_life_millis=ONE_DAY_MILLIS,
    ) == 0.5


def test_exponential_score_clamps_future_access_to_one() -> None:
    assert exponential_score(
        last_accessed=87_400_000,
        now=1_000_000,
        half_life_millis=ONE_DAY_MILLIS,
    ) == 1.0


def test_exponential_score_rejects_non_positive_half_life() -> None:
    with pytest.raises(ValueError, match="half_life_millis must be positive"):
        exponential_score(
            last_accessed=1_000_000,
            now=1_000_000,
            half_life_millis=0,
        )


@pytest.mark.parametrize("half_life_millis", (999, 315_576_000_001))
def test_exponential_score_rejects_out_of_range_half_life(
    half_life_millis: int,
) -> None:
    with pytest.raises(ValueError, match="half_life_millis must be between"):
        exponential_score(
            last_accessed=1_000_000,
            now=1_000_000,
            half_life_millis=half_life_millis,
        )


@pytest.mark.parametrize(
    ("last_accessed", "now", "importance", "expected"),
    [
        (1_000_000, 1_000_000, 1.0, 1.0),
        (1_000_000, 87_400_000, 1.0, 0.5),
        (1_000_000, 87_400_000, 0.5, 0.25),
    ],
    ids=("at-access", "base-scale", "importance-scales-score"),
)
def test_importance_weighted_power_law_score_follows_configured_curve(
    last_accessed: int, now: int, importance: float, expected: float
) -> None:
    assert importance_weighted_power_law_score(
        last_accessed=last_accessed,
        now=now,
        scale_millis=ONE_DAY_MILLIS,
        exponent=1.0,
        importance=importance,
    ) == expected


def test_importance_weighted_power_law_score_clamps_future_access_to_one() -> None:
    assert importance_weighted_power_law_score(
        last_accessed=87_400_000,
        now=1_000_000,
        scale_millis=ONE_DAY_MILLIS,
        exponent=1.0,
        importance=1.0,
    ) == 1.0


def test_importance_weighted_power_law_score_rejects_non_positive_scale() -> None:
    with pytest.raises(ValueError, match="scale_millis must be positive"):
        importance_weighted_power_law_score(
            last_accessed=1_000_000,
            now=1_000_000,
            scale_millis=0,
            exponent=1.0,
            importance=0.0,
        )


@pytest.mark.parametrize("scale_millis", (999, 315_576_000_001))
def test_importance_weighted_power_law_score_rejects_out_of_range_scale(
    scale_millis: int,
) -> None:
    with pytest.raises(ValueError, match="scale_millis must be between"):
        importance_weighted_power_law_score(
            last_accessed=1_000_000,
            now=1_000_000,
            scale_millis=scale_millis,
            exponent=1.0,
            importance=1.0,
        )


@pytest.mark.parametrize("exponent", (0.09, 10.01))
def test_importance_weighted_power_law_score_rejects_out_of_range_exponent(
    exponent: float,
) -> None:
    with pytest.raises(ValueError, match="exponent must be between"):
        importance_weighted_power_law_score(
            last_accessed=1_000_000,
            now=1_000_000,
            scale_millis=ONE_DAY_MILLIS,
            exponent=exponent,
            importance=1.0,
        )


@pytest.mark.parametrize(
    ("exponent", "importance"),
    [
        (0.0, 0.0),
        (-1.0, 0.0),
        (float("nan"), 0.0),
        (1.0, -0.1),
        (1.0, 1.1),
        (1.0, float("inf")),
    ],
    ids=(
        "zero-exponent",
        "negative-exponent",
        "nonfinite-exponent",
        "negative-importance",
        "importance-over-one",
        "nonfinite-importance",
    ),
)
def test_importance_weighted_power_law_score_rejects_invalid_float_parameters(
    exponent: float, importance: float
) -> None:
    with pytest.raises(ValueError):
        importance_weighted_power_law_score(
            last_accessed=1_000_000,
            now=1_000_000,
            scale_millis=ONE_DAY_MILLIS,
            exponent=exponent,
            importance=importance,
        )
