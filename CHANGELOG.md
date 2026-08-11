# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-08-11

### Added
- Exponential decay scoring model (`exp2(-age_ms / half_life_ms)`)
- Importance-weighted power-law decay scoring model (`importance * exp(-exponent * log1p(age_ms / scale_ms))`)
- Fixed-width C ABI with `DecayScoreExponential` and `DecayScorePowerLaw` exports
- Python `memory-decay` package with native `ctypes` acceleration and pure-Python fallback
- JSONL reference-store adapter with scan, archive, and delete operations for both models
- Standalone Go pruning-sweep CLI (`decay-sweep`) with dry-run, archive, and confirmed-delete modes
- Structured JSONL audit logging for all CLI mutation operations
- Bounded concurrent scoring with source-order preservation (worker-batched, one batch in memory)
- Sibling temporary file mutation safety with atomic replacement and cleanup on failure
- Path collision detection for input/archive/audit paths
- Native CI for Windows x86_64, Linux x86_64, and macOS ARM64
- Clean virtual-environment wheel install verification in CI
- MIT license
- README with install instructions, API reference, CLI usage, and platform support table
- CONTRIBUTING guide
- CI badges, `go vet`, and `ruff` linting in CI workflows

### Known limitations
- macOS Intel x86_64 is not supported (GitHub-hosted `macos-13` runner was unavailable)
- Python JSONL APIs read the entire file into memory (no streaming adapter yet)
- Archive mutation is not a cross-file atomic transaction
- Local Windows cgo builds require a 64-bit C compiler (GitHub CI runners provide this)
