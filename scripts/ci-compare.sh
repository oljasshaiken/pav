#!/usr/bin/env bash
# CI entrypoint: postgres + seed + rules/template HTTP servers + three-engine compare.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

chmod +x scripts/compare.sh scripts/seed-configs.sh

cleanup() {
  [[ -n "${RULES_PID:-}" ]] && kill "$RULES_PID" 2>/dev/null || true
  [[ -n "${TEMPLATE_PID:-}" ]] && kill "$TEMPLATE_PID" 2>/dev/null || true
}
trap cleanup EXIT

echo "Starting Postgres..."
docker compose up -d postgres

echo "Waiting for Postgres..."
for _ in $(seq 1 60); do
  if docker compose exec -T postgres pg_isready -U pav -d pav >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
docker compose exec -T postgres pg_isready -U pav -d pav

echo "Migrating and seeding..."
make migrate-up seed

echo "Starting rules and template engines..."
go run ./cmd/rules-engine &
RULES_PID=$!
go run ./cmd/template-engine &
TEMPLATE_PID=$!

wait_for_health() {
  local port=$1
  for _ in $(seq 1 60); do
    if curl -sf "http://localhost:${port}/health" >/dev/null; then
      return 0
    fi
    sleep 1
  done
  echo "engine on :${port} did not become healthy" >&2
  return 1
}

wait_for_health 8081
wait_for_health 8082

echo "Running three-engine compare..."
make compare CLAIM_ID="${CLAIM_ID:-00000000-0000-4000-8000-000000000001}"
