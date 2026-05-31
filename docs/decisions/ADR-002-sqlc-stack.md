# ADR-002: pgx + Hand-Written Repository Queries

## Status
Accepted

## Context
Phase 0 needs type-safe Postgres access with minimal ceremony.

## Decision
Use `pgx/v5` with hand-written queries in `internal/repository`. golang-migrate for schema. sqlc deferred — queries are small and stable in Phase 0.

## Consequences
- No sqlc codegen step in Makefile for Phase 0
- Can introduce sqlc in Phase 1 if query surface grows
