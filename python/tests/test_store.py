"""Tests for the in-memory scoring API and MemoryStore protocol."""

from __future__ import annotations

from typing import Sequence

import pytest

from fademem.store import (
    MemoryRecord,
    prune_memories,
    score_memories,
)


def test_score_memories_exponential_returns_correct_scores() -> None:
    memories = [
        MemoryRecord(id="expired", last_accessed_ms=-85_400_000, importance=1.0),
        MemoryRecord(id="boundary", last_accessed_ms=1_000_000, importance=1.0),
        MemoryRecord(id="fresh", last_accessed_ms=87_400_000, importance=1.0),
    ]
    decisions = score_memories(
        memories,
        model="exponential",
        now=87_400_000,
        half_life_millis=86_400_000,
        threshold=0.5,
    )
    assert len(decisions) == 3
    assert decisions[0].id == "expired"
    assert decisions[0].prune is True
    assert decisions[1].id == "boundary"
    assert decisions[1].prune is False
    assert decisions[2].id == "fresh"
    assert decisions[2].prune is False


def test_score_memories_power_law_uses_importance() -> None:
    memories = [
        MemoryRecord(id="low", last_accessed_ms=1_000_000, importance=0.5),
        MemoryRecord(id="high", last_accessed_ms=1_000_000, importance=1.0),
    ]
    decisions = score_memories(
        memories,
        model="power-law",
        now=87_400_000,
        scale_millis=86_400_000,
        exponent=1.0,
        threshold=0.25,
    )
    assert decisions[0].id == "low"
    assert decisions[0].score == 0.25
    assert decisions[0].prune is False  # equality retains
    assert decisions[1].id == "high"
    assert decisions[1].score == 0.5
    assert decisions[1].prune is False


def test_score_memories_preserves_input_order() -> None:
    memories = [
        MemoryRecord(id="c", last_accessed_ms=87_400_000, importance=1.0),
        MemoryRecord(id="a", last_accessed_ms=-85_400_000, importance=1.0),
        MemoryRecord(id="b", last_accessed_ms=1_000_000, importance=1.0),
    ]
    decisions = score_memories(
        memories,
        model="exponential",
        now=87_400_000,
        half_life_millis=86_400_000,
        threshold=0.5,
    )
    assert [d.id for d in decisions] == ["c", "a", "b"]


def test_score_memories_empty_list_returns_empty() -> None:
    decisions = score_memories(
        [],
        model="exponential",
        now=87_400_000,
        half_life_millis=86_400_000,
        threshold=0.5,
    )
    assert decisions == []


def test_score_memories_rejects_invalid_threshold() -> None:
    with pytest.raises(ValueError, match="threshold"):
        score_memories(
            [MemoryRecord(id="x", last_accessed_ms=0)],
            model="exponential",
            now=0,
            half_life_millis=86_400_000,
            threshold=1.5,
        )


def test_score_memories_rejects_unknown_model() -> None:
    with pytest.raises(ValueError, match="model"):
        score_memories(
            [MemoryRecord(id="x", last_accessed_ms=0)],
            model="linear",
            now=0,
            threshold=0.5,
        )


def test_score_memories_rejects_missing_half_life_for_exponential() -> None:
    with pytest.raises(ValueError, match="half_life_millis"):
        score_memories(
            [MemoryRecord(id="x", last_accessed_ms=0)],
            model="exponential",
            now=0,
            threshold=0.5,
        )


def test_score_memories_rejects_missing_scale_for_power_law() -> None:
    with pytest.raises(ValueError, match="scale_millis"):
        score_memories(
            [MemoryRecord(id="x", last_accessed_ms=0)],
            model="power-law",
            now=0,
            threshold=0.5,
        )


def test_score_memories_default_importance_is_one() -> None:
    memory = MemoryRecord(id="x", last_accessed_ms=87_400_000)
    assert memory.importance == 1.0
    decisions = score_memories(
        [memory],
        model="power-law",
        now=87_400_000,
        scale_millis=86_400_000,
        exponent=1.0,
        threshold=0.25,
    )
    assert decisions[0].score == 1.0
    assert decisions[0].prune is False


class FakeStore:
    """A simple in-memory MemoryStore for testing."""

    def __init__(self, memories: list[MemoryRecord]) -> None:
        self._memories = list(memories)
        self.archived: list[str] = []
        self.deleted: list[str] = []

    def get_memories(self) -> Sequence[MemoryRecord]:
        return list(self._memories)

    def archive_memories(self, memory_ids: Sequence[str]) -> None:
        self.archived.extend(memory_ids)
        self._memories = [m for m in self._memories if m.id not in set(memory_ids)]

    def delete_memories(self, memory_ids: Sequence[str]) -> None:
        self.deleted.extend(memory_ids)
        self._memories = [m for m in self._memories if m.id not in set(memory_ids)]


def test_prune_memories_archives_below_threshold() -> None:
    store = FakeStore([
        MemoryRecord(id="expired", last_accessed_ms=-85_400_000, importance=1.0),
        MemoryRecord(id="fresh", last_accessed_ms=87_400_000, importance=1.0),
    ])
    decisions = prune_memories(
        store,
        model="exponential",
        now=87_400_000,
        half_life_millis=86_400_000,
        threshold=0.5,
        action="archive",
    )
    assert len(decisions) == 2
    assert store.archived == ["expired"]
    assert len(store._memories) == 1
    assert store._memories[0].id == "fresh"


def test_prune_memories_deletes_below_threshold() -> None:
    store = FakeStore([
        MemoryRecord(id="expired", last_accessed_ms=-85_400_000, importance=1.0),
        MemoryRecord(id="fresh", last_accessed_ms=87_400_000, importance=1.0),
    ])
    prune_memories(
        store,
        model="exponential",
        now=87_400_000,
        half_life_millis=86_400_000,
        threshold=0.5,
        action="delete",
    )
    assert store.deleted == ["expired"]
    assert store.archived == []


def test_prune_memories_no_prune_does_not_call_store() -> None:
    store = FakeStore([
        MemoryRecord(id="fresh", last_accessed_ms=87_400_000, importance=1.0),
    ])
    prune_memories(
        store,
        model="exponential",
        now=87_400_000,
        half_life_millis=86_400_000,
        threshold=0.5,
    )
    assert store.archived == []
    assert store.deleted == []


def test_prune_memories_rejects_invalid_action() -> None:
    store = FakeStore([
        MemoryRecord(id="expired", last_accessed_ms=-85_400_000, importance=1.0),
    ])
    with pytest.raises(ValueError, match="action"):
        prune_memories(
            store,
            model="exponential",
            now=87_400_000,
            half_life_millis=86_400_000,
            threshold=0.5,
            action="vaporize",
        )


def test_prune_memories_empty_store() -> None:
    store = FakeStore([])
    decisions = prune_memories(
        store,
        model="exponential",
        now=87_400_000,
        half_life_millis=86_400_000,
        threshold=0.5,
    )
    assert decisions == []
