#!/usr/bin/env bash
# Start OutboundClaimWorkflow on LocalStack Step Functions (requires sam-deploy-localstack).
set -euo pipefail

ENDPOINT="${LOCALSTACK_ENDPOINT:-http://localhost:4566}"
REGION="${AWS_DEFAULT_REGION:-us-east-1}"
CLAIM_ID="${CLAIM_ID:-00000000-0000-4000-8000-000000000001}"

export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test

STATE_MACHINE_ARN=$(aws stepfunctions list-state-machines \
  --endpoint-url "$ENDPOINT" \
  --region "$REGION" \
  --query "stateMachines[?name=='pav-edi-outbound-claim'].stateMachineArn | [0]" \
  --output text)

if [[ -z "$STATE_MACHINE_ARN" || "$STATE_MACHINE_ARN" == "None" ]]; then
  echo "State machine pav-edi-outbound-claim not found. Run: make sam-deploy-localstack" >&2
  exit 1
fi

EXEC_ARN=$(aws stepfunctions start-execution \
  --endpoint-url "$ENDPOINT" \
  --region "$REGION" \
  --state-machine-arn "$STATE_MACHINE_ARN" \
  --input "{\"claim_id\":\"$CLAIM_ID\"}" \
  --query executionArn \
  --output text)

echo "Started execution: $EXEC_ARN"
aws stepfunctions describe-execution \
  --endpoint-url "$ENDPOINT" \
  --region "$REGION" \
  --execution-arn "$EXEC_ARN" \
  --query '{status: status, output: output}' \
  --output json
