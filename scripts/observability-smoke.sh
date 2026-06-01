#!/usr/bin/env bash
# Smoke test: validate CloudWatch dashboard template and observability package tests.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "Running observability unit tests..."
go test ./internal/platform/observability/... ./internal/lambda/dlq/... -count=1

echo "Validating CloudWatch dashboard template..."
python3 <<'PY'
import json
import sys

path = "infra/observability/dashboard.json"
with open(path) as f:
    dashboard = json.load(f)

widgets = dashboard.get("widgets", [])
if len(widgets) < 3:
    sys.exit("dashboard must define at least 3 widgets")

titles = [w.get("properties", {}).get("title", "") for w in widgets]
required = ["Claims DLQ Depth", "Lambda Errors", "Step Functions Failed"]
for title in required:
    if title not in titles:
        sys.exit(f"missing widget title: {title}")

log_widgets = [w for w in widgets if w.get("type") == "log"]
if not log_widgets:
    sys.exit("missing Logs Insights widget for dlq_alert")
query = log_widgets[0].get("properties", {}).get("query", "")
if "dlq_alert" not in query:
    sys.exit("Logs Insights query must filter event = dlq_alert")

print("observability dashboard smoke ok")
PY
