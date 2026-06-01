# Pavillio EDI Framework

Greenfield Go + Postgres EDI pipeline for multi-state Medicaid **837P** billing and **270/271** eligibility. Implements three architectural options from [RFC-EDI-001](docs/RFC-EDI-001.md); **Option 3** (CEL rules + AWS Lambda + Step Functions) is the recommended production path.

## Current progress

| Phase | Scope | Status |
|-------|-------|--------|
| [Phase 0](docs/spec/PHASE-0.md) | Canonical domain, dual HTTP services, dev comparison tooling | **Complete** |
| [Phase 1](docs/spec/PHASE-1.md) | Real `005010X222A1` 837P, engine parity, two-stage validation, dry-run submit | **Complete** |
| [Option 3 — Phase 1](docs/spec/PHASE-OPTION3.md) | CEL rules, shared pipeline, Lambda handlers, Step Functions, multi-state configs | **Complete** (slices 0.1–1.17) |
| [Option 3 — Phase 2](docs/plan/PHASE-OPTION3-TASKS.md#phase-2--eligibility--evv--observability) | EVV CEL library, 270/271 eligibility, observability, states 6–10 | **Complete** (slices 2.1–2.5) |
| [Option 3 — Phase 3](docs/plan/PHASE-OPTION3-TASKS.md#phase-3--self-service-future) | Config CRUD API, cache invalidation, payer CI matrix, SLO dashboards | **Planned** |

Full slice checklists: [Phase 0 tasks](docs/plan/PHASE-0-TASKS.md) · [Phase 1 tasks](docs/plan/PHASE-1-TASKS.md) · [Option 3 tasks](docs/plan/PHASE-OPTION3-TASKS.md)

### Option 3 deliverables

| Area | Status |
|------|--------|
| CEL env + evaluation (`internal/cel`) | Done |
| JSON Schema v1 + CEL payer config types | Done |
| Shared pipeline (`internal/pipeline`) | Done |
| Redis config cache + in-memory fallback | Done |
| Lambda handlers (load, rules, transformer, persist, parser, dispatch, dlq) | Done |
| 271 eligibility handler (`internal/lambda/eligibility`) | Done |
| `OutboundClaimWorkflow` — LoadClaim → Rules(pre) → Transform → Rules(post) → Persist | Done |
| `InboundAckWorkflow` — S3 277/999 → Parser → Persist | Done |
| EVV CEL library (`internal/cel/evvrules`) | Done |
| 270 outbound generation (`internal/edi/generate270.go`) | Done |
| 271 parser + eligibility persistence (`internal/lambda/eligibility`) | Done |
| Observability (ADR-011, `make observability-smoke`) | Done |
| ADR-010 Option 2 freeze + CI `make compare` gate | Done |
| Golden fixtures — 10 states (TX + FL/OH/PA/NY + CA/IL/GA/MI/NJ) | Done |

### Multi-state golden coverage

| State | Payer ID | Claim UUID | Config fixture |
|-------|----------|------------|----------------|
| TX (reference) | `TX-MCO-001` | `…0001` | [payer_config_837p_tx.json](docs/fixtures/payer_config_837p_tx.json) |
| FL | `FL-MCO-001` | `…0002` | [payer_config_837p_fl.json](docs/fixtures/payer_config_837p_fl.json) |
| OH | `OH-MCO-001` | `…0003` | [payer_config_837p_oh.json](docs/fixtures/payer_config_837p_oh.json) |
| PA | `PA-MCO-001` | `…0004` | [payer_config_837p_pa.json](docs/fixtures/payer_config_837p_pa.json) |
| NY | `NY-MCO-001` | `…0005` | [payer_config_837p_ny.json](docs/fixtures/payer_config_837p_ny.json) |
| CA | `CA-MCO-001` | `…0006` | [payer_config_837p_ca.json](docs/fixtures/payer_config_837p_ca.json) |
| IL | `IL-MCO-001` | `…0007` | [payer_config_837p_il.json](docs/fixtures/payer_config_837p_il.json) |
| GA | `GA-MCO-001` | `…0008` | [payer_config_837p_ga.json](docs/fixtures/payer_config_837p_ga.json) |
| MI | `MI-MCO-001` | `…0009` | [payer_config_837p_mi.json](docs/fixtures/payer_config_837p_mi.json) |
| NJ | `NJ-MCO-001` | `…0010` | [payer_config_837p_nj.json](docs/fixtures/payer_config_837p_nj.json) |

FL–NJ use synthetic fixtures until state companion guides arrive; goldens update when guides land.

### Remaining before production

From [Phase 1 success criteria](docs/plan/PHASE-OPTION3-TASKS.md#phase-1-success-criteria):

- OutboundClaimWorkflow E2E in AWS dev (SAM deploy against RDS)
- Redis cache hit >90% under load (staging)
- Remove `make compare` from CI once serverless golden suite covers all states in staging

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

## Workflow dashboard (Option 1 + Option 3)

Next.js UI at [`web/`](web/) visualizes outbound workflow steps for Option 1 and Option 3.

```bash
# After db-up migrate-up seed
make run-dashboard-api    # :8083 BFF

# terminal 2
cp web/.env.local.example web/.env.local
make run-web              # :3000 dashboard
```

| Mode | Backend path |
|------|----------------|
| Option 1 | `POST /api/claims/{id}/run?mode=option1` — rules pipeline with step trace |
| Option 3 (local) | `POST /api/claims/{id}/run?mode=option3` — in-process `workflow.Outbound` |
| Option 3 (SFN) | `POST /api/claims/{id}/run-sfn` + poll — requires `make sam-deploy-localstack` |

Default claim: `00000000-0000-4000-8000-000000000001`. Persist step is skipped by default (`skip_persist=true`).

## Seed claim UUID

`00000000-0000-4000-8000-000000000001` (payer `TX-MCO-001`)

## API (Options 1 & 2)

| Method | Path | Notes |
|--------|------|--------|
| GET | `/health` | Liveness |
| GET | `/claims/{id}/edi` | Generate 837P (pre/post validation) |
| POST | `/claims/{id}/submit?dry_run=true` | Persist EDI to `claims.x12_837`, status stays DRAFT |
| POST | `/claims/{id}/submit` (no dry_run) | `501 NOT IMPLEMENTED` |

## Architecture

Three options share domain, repository, mapping, and X12 builder code. Option 2 is **frozen** for regression only ([ADR-010](docs/decisions/ADR-010-option2-freeze.md)).

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

### Data flows ([ADR-009](docs/decisions/ADR-009-serverless-topology.md))

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

**Eligibility** (Phase 2) — 270 inquiry + 271 response:

```
270 generation (internal/edi) → outbound to payer
S3 (271) → Eligibility Lambda → eligibility_responses table
```

Local orchestration mirrors AWS in `internal/workflow/`; `cmd/workflow-local` drives Option 3 compare mode.

### Pipeline (all options)

```
LoadClaim → PreValidate → Transform → PostValidate → Persist
```

Rules and template engines must produce matching normalized EDI for the TX reference configuration. Option 3 adds CEL evaluation for `evv_rules`, `validation_rules`, and `business_rules` ([ADR-008](docs/decisions/ADR-008-cel-rules-language.md)).

Payer config shape (v1): [docs/schema/payer_config_v1.json](docs/schema/payer_config_v1.json)

### Project layout

```
pav/
├── cmd/
│   ├── rules-engine/          # Option 1 HTTP (:8081)
│   ├── template-engine/       # Option 2 HTTP (:8082)
│   ├── dashboard-api/         # Workflow BFF (:8083)
│   ├── workflow-local/        # Option 3 local orchestrator (make compare)
│   ├── gen-state-goldens/     # multi-state golden generator
│   └── lambda/                # Option 3 Lambda entrypoints
│       └── load/ rules/ transformer/ persist/
│           parser/ dispatch/ dlq/
├── internal/
│   ├── api/                   # chi HTTP handlers (Options 1 & 2)
│   │   └── dashboard/         # BFF for web workflow UI
│   ├── pipeline/              # shared Generate + event payloads + trace
│   ├── cel/                   # CEL env + rule evaluation
│   │   └── evvrules/          # standard EVV rule templates
│   ├── config/                # payer config loader + Redis cache
│   ├── validation/            # pre/post transform (CEL + legacy shim)
│   ├── workflow/              # outbound + inbound orchestration
│   ├── lambda/                # Lambda handler implementations (incl. eligibility)
│   ├── rules/ template/       # Option 1 & 2 transform engines
│   ├── mapping/ edi/          # segment assembly + X12/270 generation
│   ├── domain/ repository/    # entities + Postgres (pgx)
│   ├── queue/                 # DLQ publisher
│   ├── platform/observability/# structured logs, X-Ray hooks
│   └── states/                # multi-state golden harness
├── pkg/x12/                   # X12 builder + 277/999/271 parser
├── web/                       # Next.js workflow dashboard
├── infra/                     # AWS SAM template + Step Function ASL
│   ├── statemachine/          # outbound.asl.json, inbound.asl.json
│   └── observability/         # CloudWatch dashboard template
├── migrations/                # golang-migrate SQL
├── docs/                      # specs, plans, ADRs, fixtures, schema (see below)
└── seeds/dev/                 # local dev seed data
```

## Documentation (`docs/`)

```
docs/
├── RFC-EDI-001.md             # Architecture recommendation + phase roadmap
├── spec/
│   ├── PHASE-0.md             # Scaffold spec
│   ├── PHASE-1.md             # Real 837P + validation spec
│   └── PHASE-OPTION3.md       # CEL + serverless migration spec
├── plan/
│   ├── PHASE-0-TASKS.md       # Phase 0 slice checklist
│   ├── PHASE-1-TASKS.md       # Phase 1 slice checklist
│   └── PHASE-OPTION3-TASKS.md # Option 3 slice checklist + checkpoints
├── decisions/                 # Architecture Decision Records (ADR-001–011)
├── schema/
│   ├── payer_config_v1.json   # JSON Schema for payer_configs.config
│   ├── 001_canonical.sql      # DDL reference (canonical domain)
│   ├── 002_payer_configs.sql  # DDL reference (Option 1 configs)
│   └── 003_templates.sql      # DDL reference (Option 2 templates)
└── fixtures/
    ├── payer_config_837p_*.json   # per-state 837P configs
    ├── payer_config_270_tx.json   # TX 270 eligibility config
    ├── 837p_*_golden.x12          # per-state 837P goldens
    ├── 270_tx_golden.x12          # TX 270 golden
    ├── 271_tx_golden.x12          # TX 271 golden
    ├── 277_tx_golden.x12          # TX 277 ack golden
    └── 999_tx_golden.x12          # TX 999 ack golden
```

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
| [ADR-011](docs/decisions/ADR-011-observability.md) | Structured logs, X-Ray, DLQ alarm |

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
make observability-smoke                        # structured log + DLQ smoke
```
