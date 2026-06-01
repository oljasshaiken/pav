#!/usr/bin/env bash
# Invoke transformer handler locally (no Lambda runtime). For SAM/LocalStack use make sam-deploy-localstack.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CLAIM_ID="${CLAIM_ID:-00000000-0000-4000-8000-000000000001}"

go test "$ROOT/internal/lambda/transformer/..." -run TestHandler_matchesGoldenStrict -count=1 -v

echo "Transformer handler golden test passed for claim $CLAIM_ID"
