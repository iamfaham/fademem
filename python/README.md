# recolva

[![Windows CI](https://github.com/iamfaham/recolva/actions/workflows/cgo-smoke.yml/badge.svg)](https://github.com/iamfaham/recolva/actions/workflows/cgo-smoke.yml)
[![Linux and macOS CI](https://github.com/iamfaham/recolva/actions/workflows/native-wheel-smoke.yml/badge.svg)](https://github.com/iamfaham/recolva/actions/workflows/native-wheel-smoke.yml)

Deterministic, fully local memory-retention scoring and pruning for AI-agent long-term-memory stores.

> **Project rename:** Recolva is the successor to the earlier `fademem` package. New installations should use `recolva`; the earlier package remains available only for existing users.

`recolva` gives you two composable decay models, a fixed-width C ABI for cross-language use, a Python package with native acceleration and a pure-Python fallback, and a standalone JSONL sweep CLI with safe mutation modes and structured audit logs. No hosted service, no LLM, no embeddings, no database. Everything runs in-process on your machine.

## What it does

AI agents accumulate long-term memories. Not all of them stay relevant. `recolva` applies deterministic scoring to decide which memories to retain and which to prune, based on how recently they were accessed and how important they are. You supply the current time, the model parameters, and a threshold. The library returns a score in `[0.0, 1.0]` for each memory. Scores below your threshold are candidates for pruning.

## Scoring models

### Exponential

```text
score = exp2(-age_ms / half_life_ms)
```

Time since last access decays exponentially. `half_life_ms` controls how fast memories fade. At one half-life, the score is 0.5. At two, 0.25. And so on.

### Importance-weighted power-law

```text
score = importance * exp(-exponent * log1p(age_ms / scale_ms))
```

`importance` (in `[0, 1]`) sets the initial eligibility of a memory. `scale_ms` controls the time horizon. `exponent` controls how aggressively age reduces the score. The power-law decay is slower than exponential for old memories, which better matches how agents access historical context.

### Semantics

- Time values are signed Unix UTC epoch milliseconds (`int64` / `int64_t` / Python `int`).
- Future timestamps clamp `age_ms` to zero (score stays at full importance or 1.0).
- Pruning is strict: `score < threshold` prunes. Equality retains. This makes behavior stable at the boundary.
- All scoring is deterministic and stateless. No clock, filesystem, environment, or network reads.

## Install (Python)

The package is published on PyPI:

```bash
pip install recolva
```

Or build from source with `uv`:

```bash
cd python
uv build --wheel
pip install dist/*.whl
```

### Quickstart

```python
from recolva import exponential_score, power_law_score
from recolva.jsonl import scan_exponential_jsonl, archive_exponential_jsonl, delete_exponential_jsonl

# Score a single memory
score = exponential_score(
    last_accessed=1_000_000,
    now=87_400_000,
    half_life_millis=86_400_000,
)
# score == 0.5

score = power_law_score(
    last_accessed=1_000_000,
    now=87_400_000,
    scale_millis=86_400_000,
    exponent=1.0,
    importance=0.5,
)
# score == 0.25

# Scan a JSONL store (no mutation)
decisions = scan_exponential_jsonl(
    "memories.jsonl",
    now=87_400_000,
    half_life_millis=86_400_000,
    threshold=0.5,
)
for d in decisions:
    print(f"{d.id}: score={d.score:.4f} prune={d.prune}")

# Archive pruned records to a separate file
archive_exponential_jsonl(
    "memories.jsonl",
    "archive.jsonl",
    now=87_400_000,
    half_life_millis=86_400_000,
    threshold=0.5,
)

# Delete pruned records in place
delete_exponential_jsonl(
    "memories.jsonl",
    now=87_400_000,
    half_life_millis=86_400_000,
    threshold=0.5,
)
```

Power-law variants (`scan_power_law_jsonl`, `archive_power_law_jsonl`, `delete_power_law_jsonl`) accept `scale_millis`, `exponent`, and `threshold` instead of `half_life_millis`.

### In-memory scoring API

For use with existing memory systems (no file I/O required):

```python
from recolva import MemoryRecord, score_memories, prune_memories

# Score memories directly from your application
memories = [
    MemoryRecord(id="msg-1", last_accessed_ms=87_400_000, importance=0.8),
    MemoryRecord(id="msg-2", last_accessed_ms=-85_400_000, importance=0.5),
]

decisions = score_memories(
    memories,
    model="power-law",
    now=87_400_000,
    scale_millis=86_400_000,
    exponent=1.0,
    threshold=0.25,
)

for d in decisions:
    print(f"{d.id}: score={d.score:.4f} prune={d.prune}")
```

### Store adapter protocol

Implement the `MemoryStore` protocol to connect recolva to your existing memory system:

```python
from recolva import MemoryStore, prune_memories

class MyMemoryStore(MemoryStore):
    def get_memories(self):
        # Return memories from LangChain, Mem0, your database, etc.
        ...
    def archive_memories(self, memory_ids):
        # Move to archive or mark as archived
        ...
    def delete_memories(self, memory_ids):
        # Permanently remove
        ...

# Score and prune in one call
decisions = prune_memories(
    MyMemoryStore(),
    model="power-law",
    now=87_400_000,
    scale_millis=86_400_000,
    exponent=1.0,
    threshold=0.25,
    action="archive",  # or "delete"
)
```

### Native acceleration

The Python package bundles a platform-native shared library (Go `c-shared` build) and calls it through `ctypes`. If the native library is unavailable, it transparently falls back to a pure-Python reference implementation with identical results.

## Install (Go CLI)

```bash
go build -o decay-sweep ./cmd/decay-sweep
```

Or install directly:

```bash
go install github.com/iamfaham/recolva/cmd/decay-sweep@latest
```

### CLI usage

```bash
# Dry-run: scan and report, no file changes
decay-sweep \
  --input memories.jsonl \
  --mode dry-run \
  --model exponential \
  --now-ms 87400000 \
  --half-life-ms 86400000 \
  --threshold 0.5 \
  --workers 4

# Archive: move pruned records to a separate file
decay-sweep \
  --input memories.jsonl \
  --archive archive.jsonl \
  --audit audit.jsonl \
  --mode archive \
  --model power-law \
  --now-ms 87400000 \
  --scale-ms 86400000 \
  --exponent 1.0 \
  --threshold 0.25 \
  --workers 4

# Delete: remove pruned records in place (requires --confirm-delete)
decay-sweep \
  --input memories.jsonl \
  --audit audit.jsonl \
  --mode delete \
  --confirm-delete \
  --model exponential \
  --now-ms 87400000 \
  --half-life-ms 86400000 \
  --threshold 0.5 \
  --workers 4
```

### CLI flags

| Flag | Default | Description |
|---|---|---|
| `--input` | (required) | JSONL memory-store file |
| `--mode` | `dry-run` | `dry-run`, `archive`, or `delete` |
| `--model` | `exponential` | `exponential` or `power-law` |
| `--archive` | (empty) | Archive output file (required for `archive` mode) |
| `--audit` | (empty) | Structured JSONL audit output |
| `--confirm-delete` | `false` | Required for `delete` mode |
| `--now-ms` | `0` | Evaluation time in Unix epoch milliseconds |
| `--half-life-ms` | `0` | Exponential half-life in milliseconds |
| `--scale-ms` | `0` | Power-law scale in milliseconds |
| `--exponent` | `0` | Power-law exponent |
| `--threshold` | `0` | Prune scores strictly below this value |
| `--workers` | `1` | Bounded concurrent score calculations (max 1024) |
| `--version` | `false` | Print version and exit |

## JSONL record format

Each line is a JSON object with three fields:

```json
{"id": "unique-memory-id", "last_accessed_ms": 87400000, "importance": 0.8}
```

- `id`: non-blank string
- `last_accessed_ms`: signed integer, Unix UTC epoch milliseconds
- `importance`: float in `[0, 1]` (used by power-law model; ignored by exponential)

## Native C ABI

The Go engine exports two functions through a `c-shared` library:

```c
int32_t DecayScoreExponential(int64_t last_accessed, int64_t now, int64_t half_life_ms, double* out_score);
int32_t DecayScorePowerLaw(int64_t last_accessed, int64_t now, int64_t scale_ms, double exponent, double importance, double* out_score);
```

- Only fixed-width scalars cross the boundary: `int64_t`, `double`, `int32_t`.
- No Go pointers, strings, structs, callbacks, or caller-visible allocations.
- Status codes: `0 = OK`, `1 = invalid argument`, `2 = null output`.
- The caller owns the `double*` output pointer.

## Platform support

| Platform | Native library | Wheel | CI verified |
|---|---|---|---|
| Windows x86_64 | `recolva.dll` | `py3-none-win_amd64` | Yes |
| Linux x86_64 | `librecolva.so` | `py3-none-manylinux_2_28_x86_64` | Yes |
| macOS ARM64 | `librecolva.dylib` | `py3-none-macosx_14_0_arm64` | Yes |
| macOS Intel x86_64 | Not supported in v0.1.0 (GitHub-hosted runner unavailable) | — | — |

Native libraries are built on GitHub-hosted runners. No cross-compilation is claimed.

## Mutation safety

- **Dry-run**: no file changes.
- **Archive**: writes retained and pruned records to sibling temporary files, syncs, then atomically replaces the archive file first, then the input file.
- **Delete**: writes retained records to a sibling temporary file, syncs, then atomically replaces the input file.
- **Audit**: streamed to a sibling temporary file, synced, and atomically replaced.
- Path collisions between input, archive, and audit are rejected before any mutation.
- The CLI processes records in source order with bounded concurrent scoring (one batch of `max(1, workers) * 4` records in memory at a time).

## Development

```bash
# Go tests
go test ./cmd/decay-sweep ./internal/ffi ./internal/sweep ./pkg/decay

# Python tests
cd python
uv run --with pytest --with hatchling pytest

# Build native library
go build -buildmode=c-shared -o dist/librecolva.so ./native

# Stage and build wheel
python scripts/stage_native.py --source dist/librecolva.so --target-directory python/src/recolva/_native
cd python && uv build --wheel
```

## License

MIT