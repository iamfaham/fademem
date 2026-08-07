# Backlog

Items here are intentionally outside the v1 release scope. New non-v1 requests and review findings must be recorded here rather than silently omitted.

## Store adapters
- PostgreSQL / pgvector adapter with transactional archive/delete behavior.
- Neo4j adapter with graph-aware archival policy.
- Qdrant, Chroma, LanceDB, and generic SQL adapters.
- Restore tooling and store-specific archival destinations.

## Lifecycle integration
- Access-event ingestion and automatic `last_accessed` refresh hooks.
- Per-tenant and per-memory-class retention policies/exemptions.
- Metrics exporters, dashboards, alerting, and retention reports.

## Policy and model evolution
- Additional deterministic decay models.
- Parameter calibration/simulation tooling.
- Policy versioning and migration support.

## Distribution and ecosystem
- Signed releases, SBOM/provenance, PyPI publication, Homebrew/Scoop packages.
- Additional language bindings beyond Go and Python.
- Windows ARM64 support after a successful native build/install/CLI CI run.
