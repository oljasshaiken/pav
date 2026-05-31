# Pavillio EDI Framework (Phase 0)

Greenfield Go + Postgres scaffold for **Option 1** (rules engine) and **Option 2** (template engine) from RFC-EDI-001.

## Prerequisites

- Go 1.22+
- Docker (Postgres via compose + testcontainers in tests)
- `python3` (for `make compare`)
- [golang-migrate](https://github.com/golang-migrate/migrate) optional — Makefile falls back to Docker

## Quickstart

```bash
cp .env.example .env
make db-up
make migrate-up
make seed

# terminal 1
make run-rules

# terminal 2
make run-template

# terminal 3
make compare CLAIM_ID=00000000-0000-4000-8000-000000000001
```

## Seed claim UUID

`00000000-0000-4000-8000-000000000001`

## Docs

- [PHASE-0 Spec](docs/spec/PHASE-0.md)
- [RFC-EDI-001](docs/RFC-EDI-001.md)
- [Schema reference](docs/schema/)
- [Implementation tasks](docs/plan/PHASE-0-TASKS.md)

## Test

```bash
make test
```

Requires Docker for testcontainers integration tests.

## Architecture

Two parallel HTTP services share `internal/domain`, `internal/repository`, and `internal/api`:

- `cmd/rules-engine` → `:8081` (Option 1 JSONB payer configs)
- `cmd/template-engine` → `:8082` (Option 2 template + mapper overrides)

Phase 0 returns placeholder EDI: `{engine}:{claimID}:{configVersion}`.
