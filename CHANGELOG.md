# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-08-13

### Changed
- Renamed the project from **fademem** to **recolva**.
- Renamed the Python distribution/import package, Go module path, native-library filenames, CI artifact names, and public documentation to `recolva`.
- Earlier `fademem` releases remain available for existing users; all new development and releases use `recolva`.

## [0.1.0] - 2026-08-11

Initial release of **recolva** (formerly `decay-library`).

### Added
- Exponential decay scoring model (`exp2(-age_ms / half_life_ms)`)
- Importance-weighted power-law decay scoring model (`importance * exp(-exponent * log1p(age_ms / scale_ms))`)
- Fixed-width C ABI with `DecayScoreExponential` and `DecayScorePowerLaw` exports
- Python `recolva` package with native `ctypes` acceleration and pure-Python fallback
- JSONL reference-store adapter with scan, archive, and delete operations for both models
- In-memory scoring API (`score_memories`) for use without file I/O
- `MemoryStore` protocol for connecting recolva to existing memory systems
- `prune_memories` function for one-call score-and-prune against any store
- Standalone Go pruning-sweep CLI (`decay-sweep`) with dry-run, archive, and confirmed-delete modes
- Structured JSONL audit logging for all CLI mutation operations
- Bounded concurrent scoring with source-order preservation (worker-batched, one batch in memory)
- Sibling temporary file mutation safety with atomic replacement and cleanup on failure
- Path collision detection for input/archive/audit paths
- Native CI for Windows x86_64, Linux x86_64, and macOS ARM64
- Clean virtual-environment wheel install verification in CI
- Go test coverage reporting and Python linting in CI
- PyPI auto-publish workflow on GitHub Release
- MIT license
- README with install instructions, API reference, CLI usage, and platform support table
- CONTRIBUTING guide
- CI badges, `go vet`, and `ruff` linting in CI workflows

### Known limitations
- macOS Intel x86_64 is not supported (GitHub-hosted `macos-13` runner was unavailable)
- Python JSONL APIs read the entire file into memory (no streaming adapter yet)
- Archive mutation is not a cross-file atomic transaction
- Local Windows cgo builds require a 64-bit C compiler (GitHub CI runners provide this)
