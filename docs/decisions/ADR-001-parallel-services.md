# ADR-001: Parallel Services for Option 1 and Option 2

## Status
Accepted

## Context
RFC-EDI-001 defines three backend options. Phase 0 must house Option 1 and Option 2 for dev comparison.

## Decision
Run two parallel HTTP services (`rules-engine`, `template-engine`) sharing domain, repository, and `internal/api`.

## Consequences
- Easy A/B comparison in dev
- Two binaries to deploy locally; shared API prevents handler drift
