# ADR-004: Defer Table Partitioning

## Status
Accepted

## Context
RFC recommends partitioning `visits` and `claims` by month and state at scale.

## Decision
Defer partitioning to Phase 1. Phase 0 uses plain indexes only.

## Consequences
- Simpler migrations and local dev
- Revisit before high-volume state rollout
