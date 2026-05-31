#!/usr/bin/env bash
set -euo pipefail

CLAIM_ID="${1:?usage: compare.sh CLAIM_ID}"
RULES_URL="${RULES_URL:-http://localhost:8081}"
TEMPLATE_URL="${TEMPLATE_URL:-http://localhost:8082}"

rules=$(curl -sf "${RULES_URL}/claims/${CLAIM_ID}/edi")
template=$(curl -sf "${TEMPLATE_URL}/claims/${CLAIM_ID}/edi")

rules_edi=$(echo "$rules" | python3 -c "import sys,json; print(json.load(sys.stdin)['edi'])")
template_edi=$(echo "$template" | python3 -c "import sys,json; print(json.load(sys.stdin)['edi'])")

echo "rules:    $rules_edi"
echo "template: $template_edi"

if [[ -z "$rules_edi" || -z "$template_edi" ]]; then
  echo "empty edi output" >&2
  exit 1
fi

if [[ "$rules_edi" == "$template_edi" ]]; then
  echo "outputs are identical" >&2
  exit 1
fi

echo "outputs differ (ok)"
