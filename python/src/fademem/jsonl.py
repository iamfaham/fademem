"""Read-only JSONL adapter for exponential memory decay scans."""

from __future__ import annotations

import json
import os
import tempfile
from dataclasses import dataclass
from pathlib import Path
from typing import Union

from . import exponential_score, power_law_score


@dataclass(frozen=True)
class Decision:
    """The deterministic retention decision for one JSONL memory record."""

    id: str
    score: float
    prune: bool


def scan_exponential_jsonl(
    path: Union[str, Path],
    *,
    now: int,
    half_life_millis: int,
    threshold: float,
) -> list[Decision]:
    """Read a JSONL store and return ordered, non-mutating exponential decisions."""
    if not 0 <= threshold <= 1:
        raise ValueError("threshold must be between 0 and 1")

    decisions: list[Decision] = []
    with Path(path).open(encoding="utf-8") as records:
        for line_number, line in enumerate(records, start=1):
            try:
                record = json.loads(line)
                memory_id = record["id"]
                last_accessed = record["last_accessed_ms"]
            except (json.JSONDecodeError, KeyError, TypeError) as error:
                raise ValueError(
                    f"invalid JSONL memory record on line {line_number}"
                ) from error
            if (
                not isinstance(memory_id, str)
                or not memory_id.strip()
                or not isinstance(last_accessed, int)
                or isinstance(last_accessed, bool)
            ):
                raise ValueError(f"invalid JSONL memory record on line {line_number}")

            score = exponential_score(
                last_accessed=last_accessed,
                now=now,
                half_life_millis=half_life_millis,
            )
            decisions.append(
                Decision(id=memory_id, score=score, prune=score < threshold)
            )
    return decisions


def scan_power_law_jsonl(
    path: Union[str, Path],
    *,
    now: int,
    scale_millis: int,
    exponent: float,
    threshold: float,
) -> list[Decision]:
    """Read a JSONL store and return ordered, non-mutating power-law decisions."""
    if not 0 <= threshold <= 1:
        raise ValueError("threshold must be between 0 and 1")

    decisions: list[Decision] = []
    with Path(path).open(encoding="utf-8") as records:
        for line_number, line in enumerate(records, start=1):
            try:
                record = json.loads(line)
                memory_id = record["id"]
                last_accessed = record["last_accessed_ms"]
                importance = record["importance"]
            except (json.JSONDecodeError, KeyError, TypeError) as error:
                raise ValueError(
                    f"invalid JSONL memory record on line {line_number}"
                ) from error
            if (
                not isinstance(memory_id, str)
                or not memory_id.strip()
                or not isinstance(last_accessed, int)
                or isinstance(last_accessed, bool)
                or not isinstance(importance, (int, float))
                or isinstance(importance, bool)
            ):
                raise ValueError(f"invalid JSONL memory record on line {line_number}")
            score = power_law_score(
                last_accessed=last_accessed,
                now=now,
                scale_millis=scale_millis,
                exponent=exponent,
                importance=float(importance),
            )
            decisions.append(Decision(id=memory_id, score=score, prune=score < threshold))
    return decisions


def archive_power_law_jsonl(
    input_path: Union[str, Path],
    archive_path: Union[str, Path],
    *,
    now: int,
    scale_millis: int,
    exponent: float,
    threshold: float,
) -> list[Decision]:
    """Atomically retain non-pruned records and archive power-law pruned records."""
    source = Path(input_path)
    archive = Path(archive_path)
    if source.resolve() == archive.resolve():
        raise ValueError("archive_path must not refer to input_path")
    decisions = scan_power_law_jsonl(
        source,
        now=now,
        scale_millis=scale_millis,
        exponent=exponent,
        threshold=threshold,
    )
    raw_records = source.read_text(encoding="utf-8").splitlines(keepends=True)
    retained = [raw for raw, decision in zip(raw_records, decisions) if not decision.prune]
    pruned = [raw for raw, decision in zip(raw_records, decisions) if decision.prune]
    _replace_text_atomically(archive, "".join(pruned))
    _replace_text_atomically(source, "".join(retained))
    return decisions


def archive_exponential_jsonl(
    input_path: Union[str, Path],
    archive_path: Union[str, Path],
    *,
    now: int,
    half_life_millis: int,
    threshold: float,
) -> list[Decision]:
    """Atomically retain non-pruned records and archive pruned JSONL records."""
    source = Path(input_path)
    archive = Path(archive_path)
    if source.resolve() == archive.resolve():
        raise ValueError("archive_path must not refer to input_path")

    decisions = scan_exponential_jsonl(
        source,
        now=now,
        half_life_millis=half_life_millis,
        threshold=threshold,
    )
    raw_records = source.read_text(encoding="utf-8").splitlines(keepends=True)
    retained = [raw for raw, decision in zip(raw_records, decisions) if not decision.prune]
    pruned = [raw for raw, decision in zip(raw_records, decisions) if decision.prune]
    _replace_text_atomically(archive, "".join(pruned))
    _replace_text_atomically(source, "".join(retained))
    return decisions


def delete_power_law_jsonl(
    input_path: Union[str, Path],
    *,
    now: int,
    scale_millis: int,
    exponent: float,
    threshold: float,
) -> list[Decision]:
    """Atomically replace a JSONL store with power-law retained records."""
    source = Path(input_path)
    decisions = scan_power_law_jsonl(
        source,
        now=now,
        scale_millis=scale_millis,
        exponent=exponent,
        threshold=threshold,
    )
    raw_records = source.read_text(encoding="utf-8").splitlines(keepends=True)
    retained = [raw for raw, decision in zip(raw_records, decisions) if not decision.prune]
    _replace_text_atomically(source, "".join(retained))
    return decisions


def delete_exponential_jsonl(
    input_path: Union[str, Path],
    *,
    now: int,
    half_life_millis: int,
    threshold: float,
) -> list[Decision]:
    """Atomically replace a JSONL store with its non-pruned records."""
    source = Path(input_path)
    decisions = scan_exponential_jsonl(
        source,
        now=now,
        half_life_millis=half_life_millis,
        threshold=threshold,
    )
    raw_records = source.read_text(encoding="utf-8").splitlines(keepends=True)
    retained = [raw for raw, decision in zip(raw_records, decisions) if not decision.prune]
    _replace_text_atomically(source, "".join(retained))
    return decisions


def _replace_text_atomically(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    descriptor, temporary_name = tempfile.mkstemp(
        prefix=f".{path.name}.decay-", dir=path.parent, text=True
    )
    temporary_path = Path(temporary_name)
    try:
        with os.fdopen(descriptor, "w", encoding="utf-8", newline="") as output:
            output.write(content)
            output.flush()
            os.fsync(output.fileno())
        os.replace(temporary_path, path)
    finally:
        temporary_path.unlink(missing_ok=True)
