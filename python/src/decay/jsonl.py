"""Read-only JSONL adapter for exponential memory-decay scans."""

from __future__ import annotations

import json
from dataclasses import dataclass
from pathlib import Path
from typing import Union

from . import exponential_score


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
            if not isinstance(memory_id, str) or not isinstance(last_accessed, int):
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
