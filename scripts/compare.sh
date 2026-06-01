#!/usr/bin/env bash
# Compare Option 1 (rules :8081), Option 2 (template :8082), and Option 3 (workflow-local).
# Requires: postgres seeded, rules + template engines running.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CLAIM_ID="${1:?usage: compare.sh CLAIM_ID}"
RULES_URL="${RULES_URL:-http://localhost:8081}"
TEMPLATE_URL="${TEMPLATE_URL:-http://localhost:8082}"
GENERATED_AT="${GENERATED_AT:-2026-05-31T12:00:00Z}"
DATABASE_URL="${DATABASE_URL:-postgres://pav:pav@localhost:5432/pav?sslmode=disable}"
export GENERATED_AT DATABASE_URL CLAIM_ID COMPARE_DRY_RUN=1

GA_QS=$(python3 -c "import urllib.parse; print(urllib.parse.quote('''${GENERATED_AT}''', safe=''))")
EDI_QUERY="?generated_at=${GA_QS}"

json_edi() {
  python3 -c "import sys,json; print(json.load(sys.stdin)['edi'])"
}

rules=$(curl -sf "${RULES_URL}/claims/${CLAIM_ID}/edi${EDI_QUERY}")
template=$(curl -sf "${TEMPLATE_URL}/claims/${CLAIM_ID}/edi${EDI_QUERY}")

rules_edi=$(echo "$rules" | json_edi)
template_edi=$(echo "$template" | json_edi)

if ! workflow_json=$(cd "$ROOT" && go run ./cmd/workflow-local); then
  echo "option 3 (workflow-local) failed — see errors above" >&2
  exit 1
fi
workflow_edi=$(echo "$workflow_json" | json_edi)

echo "rules:    ${rules_edi:0:80}..."
echo "template: ${template_edi:0:80}..."
echo "workflow: ${workflow_edi:0:80}..."

if [[ -z "$rules_edi" || -z "$template_edi" || -z "$workflow_edi" ]]; then
  echo "empty edi output" >&2
  exit 1
fi

if [[ "$rules_edi" != "$template_edi" ]]; then
  echo "option 1 (rules) and option 2 (template) differ" >&2
  exit 1
fi

if [[ "$rules_edi" != "$workflow_edi" ]]; then
  echo "option 1 (rules) and option 3 (workflow) differ" >&2
  exit 1
fi

echo "all three outputs match (ok)"
