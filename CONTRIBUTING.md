# Contributing

Thanks for your interest in contributing to recolva. This document covers the basics.

## Development setup

### Go

```bash
# Run tests
go test ./cmd/decay-sweep ./internal/ffi ./internal/sweep ./pkg/decay

# Lint
go vet ./...

# Build the native shared library
go build -buildmode=c-shared -o dist/librecolva.so ./native
```

### Python

```bash
cd python

# Run tests
uv run --with pytest --with hatchling pytest

# Lint
uv run --with ruff ruff check src tests ../scripts

# Build a wheel (requires staged native library)
uv build --wheel
```

## Testing approach

This project follows test-driven development (TDD):

1. **RED**: Write a failing test for the new behavior.
2. **GREEN**: Implement the minimal code to make it pass.
3. **REFACTOR**: Clean up while keeping tests green.

Run focused tests first, then the full suite before committing.

## Before submitting a PR

- `go test ./...` passes (non-cgo packages)
- `go vet ./...` passes
- `pytest` passes
- `ruff check` passes
- No secrets, credentials, or personal paths in the diff
- New behavior has tests
- Commit messages follow conventional commits (e.g., `feat:`, `fix:`, `docs:`, `ci:`)

## CI

PRs trigger the same CI workflows that run on `main`:

- Windows x86_64 cgo build, wheel, and clean install
- Linux x86_64 and macOS ARM64 native build, wheel, and clean install

CI must pass before merge.

## Scope

v1 scope is fixed: two scoring models, fixed-width C ABI, Python wrapper, JSONL adapter, and CLI. New store adapters, decay models, and distribution channels belong in future versions. If you are unsure whether a change is in scope, open an issue first.
