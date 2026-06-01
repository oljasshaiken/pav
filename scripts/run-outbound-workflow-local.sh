#!/usr/bin/env bash
# Run OutboundClaimWorkflow locally against docker Postgres (make db-up migrate-up seed first).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
export DATABASE_URL="${DATABASE_URL:-postgres://pav:pav@localhost:5432/pav?sslmode=disable}"
export CLAIM_ID="${CLAIM_ID:-00000000-0000-4000-8000-000000000001}"
export OUTBOUND_BUCKET="${OUTBOUND_BUCKET:-pav-edi-outbound}"

cd "$ROOT"
go run ./cmd/workflow-local
