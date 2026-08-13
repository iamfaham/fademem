"""In-memory scoring API and MemoryStore protocol for recolva."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Protocol, Sequence

from .reference import (
    exponential_score,
)
from .reference import (
    importance_weighted_power_law_score as power_law_score,
)


@dataclass(frozen=True)
class MemoryRecord:
    """A single memory with its access time and importance."""

    id: str
    last_accessed_ms: int
    importance: float = 1.0


@dataclass(frozen=True)
class Decision:
    """The deterministic retention decision for one memory."""

    id: str
    score: float
    prune: bool


class MemoryStore(Protocol):
    """Protocol for adapters that connect recolva to existing memory systems."""

    def get_memories(self) -> Sequence[MemoryRecord]:
        """Return all memories with id, last_accessed_ms, and importance."""
        ...

    def archive_memories(self, memory_ids: Sequence[str]) -> None:
        """Move memories to an archive (or mark as archived)."""
        ...

    def delete_memories(self, memory_ids: Sequence[str]) -> None:
        """Permanently remove memories."""
        ...


def score_memories(
    memories: Sequence[MemoryRecord],
    *,
    model: str,
    now: int,
    threshold: float,
    half_life_millis: int | None = None,
    scale_millis: int | None = None,
    exponent: float | None = None,
) -> list[Decision]:
    """Score a sequence of memory records and return prune decisions.

    Args:
        memories: Sequence of MemoryRecord objects.
        model: "exponential" or "power-law".
        now: Current time in Unix epoch milliseconds.
        threshold: Prune scores strictly below this value (equality retains).
        half_life_millis: Required for exponential model.
        scale_millis: Required for power-law model.
        exponent: Required for power-law model.

    Returns:
        List of Decision objects in the same order as the input memories.
    """
    if not 0 <= threshold <= 1:
        raise ValueError("threshold must be between 0 and 1")

    decisions: list[Decision] = []

    for memory in memories:
        if model == "exponential":
            if half_life_millis is None:
                raise ValueError("half_life_millis is required for exponential model")
            score = exponential_score(
                last_accessed=memory.last_accessed_ms,
                now=now,
                half_life_millis=half_life_millis,
            )
        elif model == "power-law":
            if scale_millis is None:
                raise ValueError("scale_millis is required for power-law model")
            if exponent is None:
                raise ValueError("exponent is required for power-law model")
            score = power_law_score(
                last_accessed=memory.last_accessed_ms,
                now=now,
                scale_millis=scale_millis,
                exponent=exponent,
                importance=memory.importance,
            )
        else:
            raise ValueError(f"model {model!r} is not implemented")

        decisions.append(
            Decision(id=memory.id, score=score, prune=score < threshold)
        )

    return decisions


def prune_memories(
    store: MemoryStore,
    *,
    model: str,
    now: int,
    threshold: float,
    half_life_millis: int | None = None,
    scale_millis: int | None = None,
    exponent: float | None = None,
    action: str = "archive",
) -> list[Decision]:
    """Score all memories in a store and prune (archive or delete) the ones below threshold.

    Args:
        store: A MemoryStore implementation.
        model: "exponential" or "power-law".
        now: Current time in Unix epoch milliseconds.
        threshold: Prune scores strictly below this value.
        half_life_millis: Required for exponential model.
        scale_millis: Required for power-law model.
        exponent: Required for power-law model.
        action: "archive" or "delete".

    Returns:
        List of Decision objects for all memories in the store.
    """
    if action not in ("archive", "delete"):
        raise ValueError(f"action {action!r} is not implemented")

    memories = store.get_memories()
    decisions = score_memories(
        memories,
        model=model,
        now=now,
        threshold=threshold,
        half_life_millis=half_life_millis,
        scale_millis=scale_millis,
        exponent=exponent,
    )

    to_prune = [d.id for d in decisions if d.prune]

    if to_prune:
        if action == "archive":
            store.archive_memories(to_prune)
        elif action == "delete":
            store.delete_memories(to_prune)
        else:
            raise ValueError(f"action {action!r} is not implemented")

    return decisions


__all__ = [
    "MemoryRecord",
    "Decision",
    "MemoryStore",
    "score_memories",
    "prune_memories",
]
