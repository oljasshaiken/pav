#!/usr/bin/env bash
# Enqueue a claim to LocalStack SQS FIFO (grouped by payer_id).
set -euo pipefail

ENDPOINT="${LOCALSTACK_ENDPOINT:-http://localhost:4566}"
REGION="${AWS_DEFAULT_REGION:-us-east-1}"
CLAIM_ID="${CLAIM_ID:-00000000-0000-4000-8000-000000000001}"
PAYER_ID="${PAYER_ID:-TX-MCO-001}"

export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test

QUEUE_URL=$(aws sqs get-queue-url \
  --endpoint-url "$ENDPOINT" \
  --region "$REGION" \
  --queue-name pav-edi-claims.fifo \
  --query QueueUrl --output text)

BODY=$(printf '{"claim_id":"%s","payer_id":"%s"}' "$CLAIM_ID" "$PAYER_ID")

aws sqs send-message \
  --endpoint-url "$ENDPOINT" \
  --region "$REGION" \
  --queue-url "$QUEUE_URL" \
  --message-body "$BODY" \
  --message-group-id "$PAYER_ID" \
  --message-deduplication-id "${CLAIM_ID}-$(date +%s)"

echo "Enqueued claim $CLAIM_ID (group $PAYER_ID)"
