# ADR-003: Mapper Config for Template Overrides

## Status
Accepted

## Context
Option 2 could use JSON Patch or mapper configs.

## Decision
Store overrides in `template_overrides.mapper` JSONB using the same shape as `payer_configs.config.mappings`.

## Consequences
- Both engines converge on a shared mapping vocabulary
- Template base skeleton remains in `x12_templates.template`
