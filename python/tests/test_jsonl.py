from pathlib import Path

import pytest

import recolva.jsonl as jsonl
from recolva.jsonl import (
    archive_exponential_jsonl,
    archive_power_law_jsonl,
    delete_exponential_jsonl,
    delete_power_law_jsonl,
    scan_exponential_jsonl,
    scan_power_law_jsonl,
)


def test_scan_exponential_jsonl_returns_ordered_decisions_without_mutating_input(
    tmp_path: Path,
) -> None:
    store = tmp_path / "memories.jsonl"
    original = (
        '{"id":"expired","last_accessed_ms":-85400000,"importance":1}\n'
        '{"id":"boundary","last_accessed_ms":1000000,"importance":1}\n'
        '{"id":"fresh","last_accessed_ms":87400000,"importance":1}\n'
    )
    store.write_text(original, encoding="utf-8")

    decisions = scan_exponential_jsonl(
        store,
        now=87_400_000,
        half_life_millis=86_400_000,
        threshold=0.5,
    )

    assert [(decision.id, decision.prune) for decision in decisions] == [
        ("expired", True),
        ("boundary", False),
        ("fresh", False),
    ]
    assert [decision.score for decision in decisions] == [0.25, 0.5, 1.0]
    assert store.read_text(encoding="utf-8") == original


def test_scan_power_law_jsonl_uses_importance_and_strict_threshold(tmp_path: Path) -> None:
    store = tmp_path / "memories.jsonl"
    store.write_text(
        '{"id":"expired","last_accessed_ms":-171800000,"importance":0.5}\n'
        '{"id":"boundary","last_accessed_ms":1000000,"importance":0.5}\n',
        encoding="utf-8",
    )

    decisions = scan_power_law_jsonl(
        store,
        now=87_400_000,
        scale_millis=86_400_000,
        exponent=1.0,
        threshold=0.25,
    )

    assert [(decision.id, decision.prune) for decision in decisions] == [
        ("expired", True),
        ("boundary", False),
    ]
    assert [decision.score for decision in decisions] == [0.125, 0.25]


def test_scan_exponential_jsonl_rejects_boolean_timestamp_with_line_context(
    tmp_path: Path,
) -> None:
    store = tmp_path / "memories.jsonl"
    store.write_text('{"id":"invalid","last_accessed_ms":true}\n', encoding="utf-8")

    with pytest.raises(ValueError, match="line 1"):
        scan_exponential_jsonl(
            store,
            now=87_400_000,
            half_life_millis=86_400_000,
            threshold=0.5,
        )


def test_scan_exponential_jsonl_rejects_empty_id_with_line_context(
    tmp_path: Path,
) -> None:
    store = tmp_path / "memories.jsonl"
    store.write_text('{"id":"","last_accessed_ms":0}\n', encoding="utf-8")

    with pytest.raises(ValueError, match="line 1"):
        scan_exponential_jsonl(
            store,
            now=87_400_000,
            half_life_millis=86_400_000,
            threshold=0.5,
        )


def test_scan_exponential_jsonl_rejects_whitespace_only_id_with_line_context(
    tmp_path: Path,
) -> None:
    store = tmp_path / "memories.jsonl"
    store.write_text('{"id":"  ","last_accessed_ms":0}\n', encoding="utf-8")

    with pytest.raises(ValueError, match="line 1"):
        scan_exponential_jsonl(
            store,
            now=87_400_000,
            half_life_millis=86_400_000,
            threshold=0.5,
        )


def test_archive_power_law_jsonl_replaces_input_and_writes_pruned_records(
    tmp_path: Path,
) -> None:
    store = tmp_path / "memories.jsonl"
    archive = tmp_path / "archive.jsonl"
    expired = '{"id":"expired","last_accessed_ms":-171800000,"importance":0.5}\n'
    boundary = '{"id":"boundary","last_accessed_ms":1000000,"importance":0.5}\n'
    store.write_text(expired + boundary, encoding="utf-8")

    decisions = archive_power_law_jsonl(
        store,
        archive,
        now=87_400_000,
        scale_millis=86_400_000,
        exponent=1.0,
        threshold=0.25,
    )

    assert [(decision.id, decision.prune) for decision in decisions] == [
        ("expired", True),
        ("boundary", False),
    ]
    assert store.read_text(encoding="utf-8") == boundary
    assert archive.read_text(encoding="utf-8") == expired


def test_archive_exponential_jsonl_replaces_input_and_writes_pruned_records(
    tmp_path: Path,
) -> None:
    store = tmp_path / "memories.jsonl"
    archive = tmp_path / "archive.jsonl"
    expired = '{"id":"expired","last_accessed_ms":-85400000,"importance":1}\n'
    boundary = '{"id":"boundary","last_accessed_ms":1000000,"importance":1}\n'
    fresh = '{"id":"fresh","last_accessed_ms":87400000,"importance":1}\n'
    store.write_text(expired + boundary + fresh, encoding="utf-8")

    decisions = archive_exponential_jsonl(
        store,
        archive,
        now=87_400_000,
        half_life_millis=86_400_000,
        threshold=0.5,
    )

    assert [(decision.id, decision.prune) for decision in decisions] == [
        ("expired", True),
        ("boundary", False),
        ("fresh", False),
    ]
    assert store.read_text(encoding="utf-8") == boundary + fresh
    assert archive.read_text(encoding="utf-8") == expired


def test_archive_exponential_jsonl_leaves_source_unchanged_when_archive_replace_fails(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    store = tmp_path / "memories.jsonl"
    archive = tmp_path / "archive.jsonl"
    original = '{"id":"expired","last_accessed_ms":-85400000}\n'
    store.write_text(original, encoding="utf-8")

    def fail_replace(source: Path, destination: Path) -> None:
        raise OSError("simulated archive replacement failure")

    monkeypatch.setattr(jsonl.os, "replace", fail_replace)

    with pytest.raises(OSError, match="simulated archive"):
        archive_exponential_jsonl(
            store,
            archive,
            now=87_400_000,
            half_life_millis=86_400_000,
            threshold=0.5,
        )

    assert store.read_text(encoding="utf-8") == original
    assert not list(tmp_path.glob(".archive.jsonl.decay-*"))


def test_archive_exponential_jsonl_rejects_archive_path_equal_to_input_without_mutation(
    tmp_path: Path,
) -> None:
    store = tmp_path / "memories.jsonl"
    original = '{"id":"expired","last_accessed_ms":-85400000,"importance":1}\n'
    store.write_text(original, encoding="utf-8")

    with pytest.raises(ValueError, match="archive_path"):
        archive_exponential_jsonl(
            store,
            store,
            now=87_400_000,
            half_life_millis=86_400_000,
            threshold=0.5,
        )

    assert store.read_text(encoding="utf-8") == original


def test_delete_power_law_jsonl_replaces_input_with_retained_records(
    tmp_path: Path,
) -> None:
    store = tmp_path / "memories.jsonl"
    expired = '{"id":"expired","last_accessed_ms":-171800000,"importance":0.5}\n'
    boundary = '{"id":"boundary","last_accessed_ms":1000000,"importance":0.5}\n'
    store.write_text(expired + boundary, encoding="utf-8")

    decisions = delete_power_law_jsonl(
        store,
        now=87_400_000,
        scale_millis=86_400_000,
        exponent=1.0,
        threshold=0.25,
    )

    assert [(decision.id, decision.prune) for decision in decisions] == [
        ("expired", True),
        ("boundary", False),
    ]
    assert store.read_text(encoding="utf-8") == boundary


def test_delete_exponential_jsonl_replaces_input_with_retained_records(
    tmp_path: Path,
) -> None:
    store = tmp_path / "memories.jsonl"
    expired = '{"id":"expired","last_accessed_ms":-85400000,"importance":1}\n'
    boundary = '{"id":"boundary","last_accessed_ms":1000000,"importance":1}\n'
    fresh = '{"id":"fresh","last_accessed_ms":87400000,"importance":1}\n'
    store.write_text(expired + boundary + fresh, encoding="utf-8")

    decisions = delete_exponential_jsonl(
        store,
        now=87_400_000,
        half_life_millis=86_400_000,
        threshold=0.5,
    )

    assert [(decision.id, decision.prune) for decision in decisions] == [
        ("expired", True),
        ("boundary", False),
        ("fresh", False),
    ]
    assert store.read_text(encoding="utf-8") == boundary + fresh
