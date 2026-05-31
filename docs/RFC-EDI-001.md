# RFC-EDI-001: Configurable Multi-State Medicaid EDI Framework for Pavillio

**Author:** Olzhas Shaikenov  
**Date:** 2026-05-31  
**Status:** Draft  
**Version:** 1.3

## Executive Summary

Pavillio integrates caregiver mobile app EVV data into automated Medicaid billing. This RFC defines a configurable EDI processing framework for X12 837P claims and 270/271 eligibility as we scale nationally.

**Phase 0 recommendation:** Option 1 (Postgres JSONB rules engine) and Option 2 (template + mapper overrides) as parallel services for dev comparison.

## Schema Corrections (Phase 0)

These differ from RFC v1.3 draft text:

| RFC draft | Phase 0 |
|-----------|---------|
| `claims.visit_id` | Removed — visits linked via `claim_service_lines.visit_id` |
| `visits.tasks JSONB[]` | `visits.tasks JSONB` (JSON array) |
| `visits.agency_id` | Added FK to `agencies(id)` |
| Option 2 patch | `template_overrides.mapper JSONB` (same shape as `config.mappings`) |

Full DDL: [`schema/001_canonical.sql`](schema/001_canonical.sql), [`002_payer_configs.sql`](schema/002_payer_configs.sql), [`003_templates.sql`](schema/003_templates.sql).

## Architectural Options

### Option 1: Metadata-Driven Rules Engine (Postgres JSONB)

- `payer_configs` table with flexible `config JSONB`
- Rules engine reads config, transforms visit → X12

### Option 2: Hybrid Template + Mapper Overrides

- `x12_templates` base skeleton + `template_overrides.mapper`
- Template renderer applies mapper, evolves toward rules engine

### Option 3: DynamoDB (deferred)

- Same config shape externalized to DynamoDB; future `internal/rules/dynamodb` adapter

## Phase 0 Scope

- Canonical domain model + both config schemas
- Two parallel HTTP services (rules-engine :8081, template-engine :8082)
- Placeholder EDI generation — not real X12 5010

## Phase 1+ (Out of Scope)

Real X12 generation, 270/271 logic, partitioning, SFTP/AS2, AWS deployment, config UI.

See [`spec/PHASE-0.md`](spec/PHASE-0.md) for executable acceptance criteria.
