# Pavillio EDI Framework

Greenfield Go + Postgres EDI pipeline for multi-state Medicaid **837P** billing and **270/271** eligibility. The recommended production path is **Option 3** (metadata-driven configs + CEL rules + AWS Lambda + Step Functions). **Option 1** is the same config model as a local HTTP service for dev and testing.

> **Note:** Specs, ADRs, golden fixtures, and JSON Schema live under `docs/` locally (that folder is not published to GitHub). This README is the public architecture reference.

## Current progress

| Phase | Scope | Status |
|-------|-------|--------|
| Phase 0 | Canonical domain, dual HTTP services, dev comparison tooling | **Complete** |
| Phase 1 | Real `005010X222A1` 837P, engine parity, two-stage validation, dry-run submit | **Complete** |
| Option 3 — Phase 1 | CEL rules, shared pipeline, Lambda handlers, Step Functions, multi-state configs | **Complete** |
| Option 3 — Phase 2 | EVV CEL library, 270/271 eligibility, observability, states 6–10 | **Complete** |
| Option 3 — Phase 3 | Config CRUD API, cache invalidation, payer CI matrix, SLO dashboards | **Planned** |

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

| State | Payer ID | Claim UUID (suffix) |
|-------|----------|---------------------|
| TX (reference) | `TX-MCO-001` | `…0001` |
| FL | `FL-MCO-001` | `…0002` |
| OH | `OH-MCO-001` | `…0003` |
| PA | `PA-MCO-001` | `…0004` |
| NY | `NY-MCO-001` | `…0005` |
| CA | `CA-MCO-001` | `…0006` |
| IL | `IL-MCO-001` | `…0007` |
| GA | `GA-MCO-001` | `…0008` |
| MI | `MI-MCO-001` | `…0009` |
| NJ | `NJ-MCO-001` | `…0010` |

Full UUIDs follow the pattern `00000000-0000-4000-8000-00000000000N`. Per-state payer configs and golden X12 files are exercised by `go test ./internal/states/...` (fixtures are in local `docs/fixtures/`, not in the public repo).

### Remaining before production

- OutboundClaimWorkflow E2E in AWS dev (SAM deploy against RDS)
- Redis config cache wired in all Lambda paths; >90% hit rate under load in staging
- Remove `make compare` from CI once serverless golden suite covers all states in staging
- Config admin API (Phase 3) and external routing (SFTP/clearinghouse destinations)

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

Three options share domain, repository, mapping, and X12 builder code. Option 2 is **frozen** for regression only (no new payer onboarding in the template engine).

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
| **3** | `cmd/lambda/*` + Step Functions | Same JSONB + CEL rules | Production path — Lambda + SFN + Redis + SQS |

### Multi-state configurability

**The problem.** Medicaid billing at national scale means every state and payer can require different X12 implementation guides, trading-partner IDs, segment layouts, EVV rules, and validation logic — for **claims (837P)**, **eligibility (270/271)**, **acknowledgments (277/999)**, and more. Billing-code differences (e.g. home health vs personal care HCPCS) often show up as different required fields or edits, not as a separate product surface.

**The approach.** Pavillio stores payer-specific behavior as **versioned JSON config in Postgres**, not as per-state Go code. Two options implement the same idea at different runtimes:

| | Option 1 | Option 3 |
|---|----------|----------|
| **Answers** | *What* is configurable | *How* configs run at scale |
| **Runtime** | Single HTTP process (`:8081`) | Lambda + Step Functions + queue |
| **Config shape** | `payer_configs.config` JSONB | **Identical** JSONB |
| **Onboarding** | Publish JSON + tests | Same JSON + tests |

Option 2 (template + mapper overrides on `:8082`) was an alternate way to express **field mappings** only. It is frozen for `make compare` regression and is not used to onboard new states.

#### Config lookup

Each active config row in `payer_configs` ([migration](migrations/000002_option1_payer_configs.up.sql)) is keyed by:

```
(state, payer_id, transaction_type, config_version)
```

Examples: `TX` + `TX-MCO-001` + `837P` for professional claims; `TX` + `TX-MCO-001` + `270` for eligibility inquiry. The `destination` JSONB column holds routing metadata (clearinghouse / SFTP targets) — schema is in place; full delivery integration is future work.

Billing codes are **not** a separate config key. Procedure codes and service context live on the claim; payer-specific behavior is expressed with **CEL rules** over that data (e.g. “require diagnosis when service type is home health”).

#### What goes in a payer config

| Layer | JSON field | Purpose |
|-------|------------|---------|
| Format | `x12_version` | Implementation guide (e.g. `005010X222A1` for 837P) |
| Trading partners | `envelope` | ISA/GS/ST sender, receiver, functional IDs |
| Field layout | `mappings` | X12 loop/element → claim context path (`patient.last_name`, `service_line.procedure_code`, …) |
| Pre-bill checks | `validation_rules` | CEL on claim context before X12 is built |
| EVV mandates | `evv_rules` | CEL on visit/GPS/signature data |
| Post-build checks | `business_rules` | CEL on generated segments after transform |

Types are defined in [`internal/domain/config.go`](internal/domain/config.go). Resolution from paths to values is in [`internal/mapping`](internal/mapping); X12 assembly is in [`internal/edi`](internal/edi) and [`pkg/x12`](pkg/x12).

#### Example (TX 837P, abbreviated)

```json
{
  "x12_version": "005010X222A1",
  "envelope": {
    "isa": { "sender_id": "PAVILLIO", "receiver_id": "TX_MCO" },
    "gs":  { "functional_id": "HC", "application_receiver": "TX_MCO" },
    "st":  { "transaction_set_id": "837" }
  },
  "mappings": {
    "patient": {
      "loop_2010BA": {
        "NM103": "patient.last_name",
        "NM109": "patient.medicaid_id"
      }
    },
    "service_line": {
      "loop_2400": {
        "SV101": "service_line.procedure_code",
        "SV104": "service_line.units"
      }
    }
  },
  "evv_rules": [{
    "id": "evv_verified",
    "cel": "visit.evv_status == \"VERIFIED\"",
    "message": "EVV visit must be verified before billing"
  }],
  "validation_rules": [{
    "id": "diagnosis_required",
    "cel": "authorization.service_type != \"home_health\" || size(service_line.diagnosis_codes) > 0",
    "message": "diagnosis_code required for home health claims"
  }]
}
```

**CEL** (Common Expression Language) lets ops and onboarding engineers add payer rules in config without redeploying Go. Rules are evaluated in [`internal/cel`](internal/cel) with typed bindings from [`internal/domain`](internal/domain) entities (patient, claim, visit, service line, etc.).

#### Option 1 — configurability model (local HTTP)

Option 1 is **`cmd/rules-engine`** on port **8081**. For each claim it runs the shared pipeline in one request:

```
LoadClaim → PreValidate (CEL) → Transform (mappings → 837P) → PostValidate (CEL) → optional Persist
```

Implementation: [`internal/pipeline/generate.go`](internal/pipeline/generate.go), [`internal/rules/engine.go`](internal/rules/engine.go), [`internal/api/server.go`](internal/api/server.go).

**Onboarding a new state or payer (Option 1 path):**

1. Add a `payer_configs` row (or seed JSON) for `(state, payer_id, transaction_type)`.
2. Set `envelope`, `mappings`, and CEL rule arrays for that payer’s companion guide.
3. Add a golden X12 expected output and a test in [`internal/states`](internal/states) (ten states scaffolded today).
4. Run `make compare` to confirm Option 1, Option 2 (regression), and Option 3 local workflow still match for reference payers.

No Go change is required unless you need a **new mapping primitive** the resolver does not support yet.

**Limits today:** synchronous HTTP only (scale by adding replicas); config cache and admin publish API are planned; ten reference states, not fifty; external file delivery (SFTP) not wired.

#### Option 3 — configurability platform (production)

Option 3 runs the **same config JSON** as Option 1, split into steps for independent scaling ([`infra/statemachine/outbound.asl.json`](infra/statemachine/outbound.asl.json)):

```
LoadClaim → Rules(pre) → Transform → Rules(post) → Persist
                │                              │
                └── validation failure ──► SQS FIFO DLQ
```

| Step | Lambda / code | Reads from config |
|------|---------------|-------------------|
| Load | [`internal/lambda/load`](internal/lambda/load) | Full row from `payer_configs` |
| Rules (pre/post) | [`internal/lambda/rules`](internal/lambda/rules) | `validation_rules`, `evv_rules`, `business_rules` |
| Transform | [`internal/lambda/transformer`](internal/lambda/transformer) | `envelope`, `mappings` → calls `rules.Transform837P` |
| Persist | [`internal/lambda/persist`](internal/lambda/persist) | Writes EDI to Postgres + S3 outbound bucket |

Supporting infrastructure ([`infra/template.yaml`](infra/template.yaml)): **Redis** for hot config cache ([`internal/config`](internal/config)), **SQS FIFO** for claim intake and **DLQ** for failed validations, **S3** for outbound 837P and inbound 277/999/271, **CloudWatch** structured logs (`workflow_step`, `dlq_alert`).

Other transaction flows use the same config pattern:

- **270 outbound** — [`internal/edi/generate270.go`](internal/edi/generate270.go) with `transaction_type = 270`
- **271 inbound** — [`internal/lambda/eligibility`](internal/lambda/eligibility) parses S3 payloads into `eligibility_responses`
- **277/999 inbound** — `InboundAckWorkflow` → parser → `claims.response_277`

Local dev runs the same orchestration without AWS via [`cmd/workflow-local`](cmd/workflow-local/main.go) and the [workflow dashboard](#workflow-dashboard-option-1--option-3) (`make run-dashboard-api` + `make run-web`).

**Onboarding a new state or payer (Option 3 path):** same config JSON and golden tests as Option 1 — Option 3 does not define a second config format. Production readiness is about **throughput** (cache hit rate, Lambda pool reuse, RDS Proxy, AWS E2E), not different metadata.

**Limits today:** full AWS staging E2E still open; Redis not wired on every code path; SFTP/clearinghouse destinations deferred; Phase 3 self-service config API not built.

### Data flows (Option 3)

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

Rules and template engines must produce matching normalized EDI for the TX reference configuration. Option 3 evaluates `evv_rules`, `validation_rules`, and `business_rules` via CEL in the rules Lambda steps.

Config types: [`internal/domain/config.go`](internal/domain/config.go) · CEL evaluation: [`internal/cel`](internal/cel)

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
├── seeds/dev/                 # local dev seed data
└── docs/                      # local only (gitignored): specs, ADRs, fixtures, schema
```

### Local documentation (`docs/`)

The `docs/` tree (specs, task plans, ADRs, JSON Schema, per-state fixtures and goldens) is **not published** with the GitHub repo. Clone with internal access or copy from a teammate to get fixtures for `make seed` / `scripts/seed-configs.sh`. Architecture summary lives in this README; code is the other source of truth under `internal/` and `infra/`.

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
