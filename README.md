# Pavillio EDI Framework

Greenfield Go + Postgres EDI pipeline for multi-state Medicaid **837P** billing. Implements three architectural options from [RFC-EDI-001](docs/RFC-EDI-001.md); **Option 3** (CEL rules + AWS Lambda + Step Functions) is the recommended production path.

## Current progress

| Phase | Scope | Status |
|-------|-------|--------|
| [Phase 0](docs/spec/PHASE-0.md) | Canonical domain, dual HTTP services, dev comparison tooling | **Complete** |
| [Phase 1](docs/spec/PHASE-1.md) | Real `005010X222A1` 837P, engine parity, two-stage validation, dry-run submit | **Complete** |
| [Option 3](docs/spec/PHASE-OPTION3.md) | CEL rules, shared pipeline, Lambda handlers, Step Functions, multi-state configs | **Phase 1 complete** (slices 0.1–1.17) |

### Option 3 slice status

See [PHASE-OPTION3-TASKS.md](docs/plan/PHASE-OPTION3-TASKS.md) for full acceptance criteria.

| Area | Status |
|------|--------|
| CEL env + evaluation (`internal/cel`) | Done |
| JSON Schema v1 + CEL payer config types | Done |
| Shared pipeline (`internal/pipeline`) | Done |
| Redis config cache + in-memory fallback | Done |
| Lambda handlers (load, rules, transformer, persist, parser, dispatch, dlq) | Done |
| `OutboundClaimWorkflow` — LoadClaim → Rules(pre) → Transform → Rules(post) → Persist | Done |
| `InboundAckWorkflow` — S3 277/999 → Parser → Persist | Done |
| Golden fixtures for TX, FL, OH, PA, NY | Done |
| ADR-010 Option 2 freeze + CI `make compare` gate | Done |

## Prerequisites

- Go 1.22+
- Docker (Postgres via compose; testcontainers in tests)
- `python3` (for `make compare`)
- [AWS SAM CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/install-sam-cli.html) (`brew install aws-sam-cli`) for LocalStack deploy
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

# terminal 3 — all three engines must return identical EDI (Options 1, 2, 3)
make compare CLAIM_ID=00000000-0000-4000-8000-000000000001
```

`make compare` hits Option 1 (`:8081`) and Option 2 (`:8082`) over HTTP, then runs Option 3 via `cmd/workflow-local` in **compare dry-run** mode (`COMPARE_DRY_RUN=1`, no DB writes). All use the same `GENERATED_AT` timestamp (default `2026-05-31T12:00:00Z`) for deterministic envelope fields.

Run engines in **separate terminals** — `make run-rules` and `make run-template` block until stopped.

`make seed` loads claim data from `seeds/dev/seed.sql`, then applies TX reference payer/template fixtures via `scripts/seed-configs.sh`.

### Option 3 / serverless (local)

```bash
# Outbound workflow against docker Postgres (after db-up migrate-up seed)
make run-outbound-workflow

# LocalStack stack (S3, SQS FIFO, Redis)
make localstack-up
make invoke-transformer          # handler-only golden test
make sam-deploy-localstack       # optional SAM deploy to LocalStack
make start-outbound-sfn          # Step Functions execution
```

## Seed claim UUID

`00000000-0000-4000-8000-000000000001` (payer `TX-MCO-001`)

## API (Options 1 & 2)

| Method | Path | Notes |
|--------|------|--------|
| GET | `/health` | Liveness |
| GET | `/claims/{id}/edi` | Generate 837P (pre/post validation) |
| POST | `/claims/{id}/submit?dry_run=true` | Persist EDI to `claims.x12_837`, status stays DRAFT |
| POST | `/claims/{id}/submit` (no dry_run) | `501 NOT_IMPLEMENTED` |

## Architecture

Three options share domain, repository, mapping, and X12 builder code. Option 2 is **frozen** for regression only.

```
                    ┌─────────────────────────────────────────┐
                    │           Shared core (Go)              │
                    │  domain · repository · mapping · x12    │
                    │  validation · cel · pipeline · config   │
                    └─────────────────────────────────────────┘
                          │              │              │
              Option 1    │    Option 2  │    Option 3  │
              (local)     │   (frozen)   │ (serverless) │
                          ▼              ▼              ▼
                   cmd/rules-engine  cmd/template-engine  cmd/lambda/*
                   :8081 HTTP        :8082 HTTP          Step Functions
                          │              │              │
                          └──── make compare ────────────┘
```

### Architectural options

| Option | Entry point | Config | Role |
|--------|-------------|--------|------|
| **1** | `cmd/rules-engine` (:8081) | `payer_configs` JSONB | Local HTTP adapter over shared pipeline |
| **2** | `cmd/template-engine` (:8082) | `x12_templates` + mapper overrides | Frozen — parity regression via `make compare` |
| **3** | `cmd/lambda/*` + Step Functions | Same JSONB + CEL rules | Production path ([ADR-009](docs/decisions/ADR-009-serverless-topology.md)) |

### Step Functions ([ADR-009](docs/decisions/ADR-009-serverless-topology.md))

**OutboundClaimWorkflow** — claim → 837P in Postgres + S3:

```
LoadClaim → Rules(pre) → Transform → Rules(post) → Persist
                │                              │
                └── validation failure ──► SQS FIFO DLQ
```

**InboundAckWorkflow** — payer ack → ledger:

```
S3 (277/999) → Parser → Persist (claims.response_277)
```

Local orchestration mirrors AWS in `internal/workflow/`; `cmd/workflow-local` drives Option 3 compare mode.

### Project layout

```
pav/
├── cmd/
│   ├── rules-engine/          # Option 1 HTTP (:8081)
│   ├── template-engine/       # Option 2 HTTP (:8082)
│   ├── workflow-local/        # Option 3 local orchestrator (make compare)
│   ├── lambda/                # Option 3 Lambda entrypoints
│   │   ├── load/ rules/ transformer/ persist/
│   │   ├── parser/ dispatch/ dlq/
│   └── gen-state-goldens/     # multi-state golden generator
├── internal/
│   ├── api/                   # chi HTTP handlers (Options 1 & 2)
│   ├── pipeline/              # shared Generate + event payloads
│   ├── cel/                   # CEL env + rule evaluation
│   ├── config/                # payer config loader + Redis cache
│   ├── validation/            # pre/post transform (CEL + legacy shim)
│   ├── workflow/              # outbound + inbound orchestration
│   ├── lambda/                # Lambda handler implementations
│   ├── rules/ template/       # Option 1 & 2 transform engines
│   ├── mapping/ edi/          # segment assembly + X12 generation
│   ├── domain/ repository/    # entities + Postgres (pgx)
│   ├── queue/                 # DLQ publisher
│   └── states/                # FL/OH/PA/NY golden harness
├── pkg/x12/                   # X12 builder + 277/999 parser
├── infra/                     # AWS SAM template + Step Function ASL
├── migrations/                # golang-migrate SQL
├── docs/                      # specs, plans, ADRs, fixtures, schema
└── seeds/dev/                 # local dev seed data
```

### Pipeline (all options)

`LoadClaim → PreValidate → Transform → PostValidate → Persist`

Rules and template engines must produce matching normalized EDI for the TX reference configuration. Option 3 adds CEL evaluation for `evv_rules`, `validation_rules`, and `business_rules` ([ADR-008](docs/decisions/ADR-008-cel-rules-language.md)).

## Documentation

### Specs & plans

| Doc | Description |
|-----|-------------|
| [RFC-EDI-001](docs/RFC-EDI-001.md) | Architecture recommendation and phase roadmap |
| [PHASE-0 Spec](docs/spec/PHASE-0.md) | Scaffold — domain, dual services, comparison tooling |
| [PHASE-1 Spec](docs/spec/PHASE-1.md) | Real 837P, validation, dry-run submit |
| [PHASE-OPTION3 Spec](docs/spec/PHASE-OPTION3.md) | CEL + serverless migration |
| [Phase 0 tasks](docs/plan/PHASE-0-TASKS.md) | Slice checklist |
| [Phase 1 tasks](docs/plan/PHASE-1-TASKS.md) | Slice checklist |
| [Option 3 tasks](docs/plan/PHASE-OPTION3-TASKS.md) | Slice checklist + checkpoints |

### Architecture decisions

| ADR | Topic |
|-----|-------|
| [ADR-001](docs/decisions/ADR-001-parallel-services.md) | Parallel Option 1 & 2 services |
| [ADR-002](docs/decisions/ADR-002-sqlc-stack.md) | pgx repository stack |
| [ADR-003](docs/decisions/ADR-003-mapper-config-overrides.md) | Template mapper overrides |
| [ADR-004](docs/decisions/ADR-004-defer-partitioning.md) | Defer table partitioning |
| [ADR-005](docs/decisions/ADR-005-engine-parity.md) | Engine parity requirement |
| [ADR-006](docs/decisions/ADR-006-dry-run-submission.md) | Dry-run submit semantics |
| [ADR-007](docs/decisions/ADR-007-internal-x12-builder.md) | Internal X12 builder |
| [ADR-008](docs/decisions/ADR-008-cel-rules-language.md) | CEL for rules evaluation |
| [ADR-009](docs/decisions/ADR-009-serverless-topology.md) | Lambda + Step Functions topology |
| [ADR-010](docs/decisions/ADR-010-option2-freeze.md) | Option 2 template engine freeze |

### Schema & fixtures

| Path | Description |
|------|-------------|
| [docs/schema/](docs/schema/) | SQL DDL references + [payer_config_v1.json](docs/schema/payer_config_v1.json) |
| [docs/fixtures/](docs/fixtures/) | Golden X12, payer configs, templates (TX, FL, OH, PA, NY) |

## Test

```bash
make test
```

Requires Docker for testcontainers integration tests.

CI runs `go test ./...` and a separate **compare** job (`scripts/ci-compare.sh`) that exercises `make compare` against a docker-compose Postgres instance.

Key verify targets:

```bash
make compare                                    # three-engine parity
go test ./internal/workflow/... -count=1        # outbound + inbound workflows
go test ./internal/states/... -count=1          # multi-state golden harness
go test ./internal/cel/... -bench=. -count=1    # CEL benchmark gate
```
