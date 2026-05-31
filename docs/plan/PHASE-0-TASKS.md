# Phase 0 Implementation Tasks

In-repo mirror of the implementation plan. See [PHASE-0 spec](../spec/PHASE-0.md).

## Slices

| Slice | Summary | Verify |
|-------|---------|--------|
| 0 | RFC, schema, fixtures, spec | Files exist; JSON valid |
| 1 | git, go.mod, migration 001, domain, docker-compose | `make db-up && make migrate-up && make migrate-down && go build ./...` |
| 2 | migration 002, payer config repo | `go test ./internal/domain/... ./internal/repository/... -run PayerConfig` |
| 3 | migration 003, templates | `go test ./internal/domain/... ./internal/repository/... -run Template` |
| 4a | LoadClaimContext, rules stub, platform | `go test ./internal/repository/... -run ClaimContext` |
| 4b | internal/api, cmd/rules-engine | `go test ./internal/api/... ./internal/rules/...` (includes distinct-engine test) |
| 5 | template engine, cmd/template-engine | `go test ./internal/template/...` |
| 6 | seeds, compare, validation/submission stubs | `make seed && make compare CLAIM_ID=... && make test` |
| 7 | ADRs, README, CI | CI green |

## Checkpoints

- **0:** After slice 0 — docs complete
- **A:** After slices 1–3 — foundation
- **B:** After slices 4b–5 — engines
- **C:** After slice 6 — dev tooling
- **D:** After slice 7 — Phase 0 complete
