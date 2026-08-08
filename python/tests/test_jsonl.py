from pathlib import Path

import pytest

from decay.jsonl import scan_exponential_jsonl


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
