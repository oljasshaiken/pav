# Spec: Pavillio EDI Framework — Phase 0

**Status:** Accepted  
**Author:** Olzhas Shaikenov  
**Date:** 2026-05-31  
**Related:** [RFC-EDI-001](../RFC-EDI-001.md), [PHASE-0 Tasks](../plan/PHASE-0-TASKS.md)

---

## Objective

Scaffold a greenfield Go + Postgres backend repository that houses **Option 1** (metadata-driven rules engine with JSONB configs) and **Option 2** (hybrid template + mapper overrides) from RFC-EDI-001, as **two parallel services** sharing a canonical domain model and repositories.

### Users

- **Backend engineers** building the EDI pipeline and onboarding new states
- **State onboarding engineers** adding payer-specific configuration without code changes (Phase 1+)

### User stories

1. As a backend engineer, I can start Postgres locally, run migrations, and seed synthetic data with one command sequence.
2. As a backend engineer, I can run the rules engine and template engine side-by-side and compare EDI output for the same claim.
3. As a backend engineer, I can add payer config (Option 1) or template override (Option 2) via Postgres rows without changing Go code (Phase 0: schema + loaders only; stubs for transform).

### What success looks like

Phase 0 proves **end-to-end wiring** — domain model, config storage, repository loaders, HTTP APIs, and placeholder EDI generation — before real X12 5010 segment logic. Both architectural options coexist for dev comparison.

---

## Tech Stack

| Tool | Version |
|------|---------|
| Go | 1.22+ |
| PostgreSQL | 16 |
| github.com/jackc/pgx/v5 | latest stable |
| golang-migrate | v4 |
| github.com/go-chi/chi/v5 | v5 |
| testcontainers-go | integration tests |

**Module path:** `github.com/pavillio/pav-edi` (confirm org before `go mod init`)

---

## Commands

### Prerequisites

```bash
# Install tooling (macOS)
brew install go migrate

# Or via Go install
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### Environment

```bash
cp .env.example .env
# Defaults:
# DATABASE_URL=postgres://pav:pav@localhost:5432/pav?sslmode=disable
# RULES_ENGINE_PORT=8081
# TEMPLATE_ENGINE_PORT=8082
```

### Database

```bash
make db-up
make migrate-up
make migrate-down    # rollback all migrations
make seed
```

Direct migrate invocation (without Makefile):

```bash
migrate -path migrations \
  -database "postgres://pav:pav@localhost:5432/pav?sslmode=disable" up
```

### Run services

```bash
make run-rules      # localhost:8081
make run-template   # localhost:8082
```

### Build

```bash
go build ./...
go build -o /dev/null ./cmd/rules-engine ./cmd/template-engine
```

### Test & compare

```bash
make test
make compare CLAIM_ID=<uuid-from-seed>   # requires Slice 6 seeds + both services running
```

**Verification by slice:**

- **Slices 4–5:** engines verified via testcontainers integration tests (`go test ./internal/api/... ./internal/rules/... ./internal/template/...`)
- **Slice 6 onward:** manual checks below work after `make seed` and starting both services

Manual API checks (after `make seed` and `make run-rules` / `make run-template`):

```bash
curl -s localhost:8081/health | jq .
curl -s localhost:8081/claims/<uuid>/edi | jq .
curl -s localhost:8082/claims/<uuid>/edi | jq .
diff <(curl -s localhost:8081/claims/<uuid>/edi) \
     <(curl -s localhost:8082/claims/<uuid>/edi)
```

### Format

```bash
go fmt ./...
go vet ./...
```

No coverage gate in Phase 0. All success criteria below must pass.

---

## Project Structure

```
pav/
├── cmd/
│   ├── rules-engine/          # Option 1 entrypoint (port 8081)
│   └── template-engine/       # Option 2 entrypoint (port 8082)
├── internal/
│   ├── api/                   # Shared HTTP routes and handlers
│   ├── domain/                # Canonical structs, ClaimContext, config types
│   ├── repository/            # Hand-written pgx queries (see ADR-002)
│   ├── rules/                 # Option 1 Engine interface + stub
│   ├── template/              # Option 2 Renderer interface + stub
│   ├── validation/            # Multi-stage pipeline stub
│   ├── submission/            # Submission/routing stub
│   └── platform/              # DB pool, env config, slog logging
├── pkg/x12/                   # X12 Document types (stub parser)
├── migrations/                # golang-migrate SQL files
├── seeds/dev/                 # Synthetic fixtures
├── docs/
│   ├── RFC-EDI-001.md
│   ├── spec/PHASE-0.md        # This file
│   └── decisions/             # ADR-001 through ADR-004
├── .github/workflows/test.yml
├── docker-compose.yml
├── Makefile
└── README.md
```

---

## Code Style

```go
// Errors: wrap with context
return fmt.Errorf("load claim context: %w", err)

// Context: always first parameter
func (r *Repo) LoadClaimContext(ctx context.Context, claimID uuid.UUID) (domain.ClaimContext, error)

// Logging: structured via slog; include claim_id when available
slog.Info("edi generated", "claim_id", claimID, "engine", "rules")

// Struct tags: db for repository scans, json for API
type Claim struct {
    ID     uuid.UUID `db:"id" json:"id"`
    Status string    `db:"status" json:"status"`
}
```

Conventions:

- Package names: lowercase, single word (`domain`, `rules`, `template`)
- Interfaces: small, engine-specific (`rules.Engine`, `template.Renderer`)
- No config embedded in `ClaimContext`; engines load config via repository helpers
- Placeholder EDI format: `{engine}:{claimID}:{configVersion}`

---

## Testing Strategy

| Level | Scope | Location | When |
|-------|-------|----------|------|
| Unit | JSON unmarshaling for payer config and mapper shapes | `internal/domain/*_test.go` | Slice 2–3 |
| Integration | Repository queries against Postgres | `internal/repository/*_test.go` | Slice 2–6 (testcontainers) |
| HTTP smoke | GET `/health`, GET `/claims/{id}/edi` | `internal/api/*_test.go` | Slice 4–6 |
| CI | Full test suite on PR/push | `.github/workflows/test.yml` | Slice 7 |

**Slice 4–5 vs Slice 6:** Slices 4–5 prove engines via **testcontainers** (fixture claim inserted in test). Manual `curl` and `make compare` require **Slice 6** seed data and running services.

Run before every commit: `make test`

---

## Boundaries

### Always do

- Run `make test` before committing
- Use synthetic PHI in seeds and tests (fake names, Medicaid IDs)
- Wrap errors with context; pass `context.Context` through call chains
- Run `go fmt ./...` on changed Go files

### Ask first

- Adding dependencies beyond chi, pgx, testcontainers
- Changing migration schema after initial merge
- Adding auth, TLS, or external service integrations

### Never do

- Commit real patient/caregiver data or credentials
- Inline secrets in `destination` JSONB (use `credentials_ref` env var names only)
- Remove or skip failing tests without explicit approval

---

## Locked Decisions

| Decision | Choice |
|----------|--------|
| Query layer | pgx/v5 hand-written queries (sqlc deferred — ADR-002) |
| Migrations | golang-migrate |
| Option 2 overrides | Mapper config (same shape as Option 1 `mappings`) |
| Claim ↔ visit | Visits linked via `claim_service_lines.visit_id` only (no `claims.visit_id`) |
| Config version | Latest active: `WHERE active ORDER BY config_version DESC LIMIT 1` |
| HTTP transform | `GET /claims/{claimID}/edi` |
| Shared HTTP | `internal/api` package used by both cmd entrypoints |
| Partitioning | Deferred to Phase 1 |

See `docs/decisions/ADR-*.md` for rationale.

---

## HTTP API Contract

Both services expose identical routes; only the injected engine differs.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness |

**GET /health → 200:**

```json
{ "status": "ok" }
```

| GET | `/claims/{claimID}/edi` | Generate placeholder EDI |

**200 response:**

```json
{
  "claim_id": "uuid",
  "engine": "rules",
  "config_version": 1,
  "edi": "rules:uuid:1",
  "generated_at": "2026-05-31T12:00:00Z"
}
```

**Error envelope:**

```json
{
  "error": {
    "code": "CLAIM_NOT_FOUND",
    "message": "claim not found"
  }
}
```

Status codes: `404` not found, `422` validation failure, `500` internal (no stack traces).

**Error codes:**

| Code | HTTP | When |
|------|------|------|
| `CLAIM_NOT_FOUND` | 404 | Claim ID not in database |
| `INVALID_CLAIM_ID` | 422 | Malformed UUID |
| `CONFIG_NOT_FOUND` | 422 | No active payer/template config |
| `INTERNAL_ERROR` | 500 | Unexpected failure |

Ports: rules-engine `8081`, template-engine `8082`. No auth/TLS in Phase 0.

---

## LoadClaimContext Rules

1. Load claim by ID; error if missing.
2. Load all `claim_service_lines`; error if none.
3. Load visits for each distinct `visit_id`; error if any missing.
4. Load patient from first visit; load authorization from first service line's `authorization_id` if set, else latest active auth for patient + claim `payer_id`. **Phase 0 assumes a single patient per claim** (first visit only).
5. Load agency from first visit's `agency_id`.
6. Do **not** embed payer/template config in `ClaimContext`.

## Template Resolution

`GetActiveTemplateOverride(state, payer_id, transaction_type)` joins `template_overrides` → `x12_templates` where override is active with highest `override_version`. Phase 0 uses template named **`837P-base`**.

## Stub Interfaces (Phase 0)

```go
// internal/validation — no-op
type Pipeline interface {
    Validate(ctx context.Context, doc x12.Document) error
}

// internal/submission — no-op; not called on GET in Phase 0
type Service interface {
    Submit(ctx context.Context, doc x12.Document) error
}
```

Call order: transform → validate → return EDI.

## make compare

Requires both services running. Fetches `GET /claims/{id}/edi` from :8081 and :8082, compares `edi` fields. Exit 0 if different non-empty strings; exit 1 if same or either request fails.

---

## Out of Scope (Phase 0)

- Real X12 5010 segment generation
- 270/271 transaction logic
- Table partitioning
- SFTP/AS2 submission, AWS deployment
- Option 3 DynamoDB adapter
- Frontend configuration UI
- HTTP authentication and TLS

---

## Success Criteria

Phase 0 is complete when **all** of the following pass:

1. [ ] `make db-up && make migrate-up && make seed` completes without error
2. [ ] `make migrate-down` rolls back all migrations cleanly
3. [ ] `GET /health` returns `200` on ports 8081 and 8082
4. [ ] `GET /claims/{claimID}/edi` returns distinct non-empty placeholder EDI from each engine
5. [ ] `make compare CLAIM_ID=<seed-claim>` shows different outputs from rules vs template engines
6. [ ] `make test` passes (unit + integration + smoke)
7. [ ] `.github/workflows/test.yml` passes on a clean checkout

---

## Open Questions

1. **Module path** — `github.com/pavillio/pav-edi` (confirmed for Phase 0)
2. **Seed claim UUID** — `00000000-0000-4000-8000-000000000001` (see README)

---

## Checkpoints

Pause at each checkpoint before starting the next slice group. Full checklist: [Implementation Plan — Checkpoints](../../.cursor/plans/pav_edi_repo_scaffold_563f1081.plan.md#checkpoints).

| Checkpoint | After slices | Gate |
|------------|--------------|------|
| **0: Documentation** | 0 | RFC, schema, fixtures, spec contracts |
| **A: Foundation** | 1–3 | Migrations, domain/repo tests, build |
| **B: Engines** | 4b–5 | API/rules/template tests; distinct placeholder EDI (automated) |
| **C: Dev tooling** | 6 | Seeds, `make compare`, full test suite |
| **D: Phase 0 complete** | 7 | All success criteria + ADRs + CI |

Full checklist: [PHASE-0-TASKS.md](../plan/PHASE-0-TASKS.md)
