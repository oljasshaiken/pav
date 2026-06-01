#!/usr/bin/env bash
# Build and deploy SAM stack to LocalStack (requires: sam CLI, docker, localstack running).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENDPOINT="${LOCALSTACK_ENDPOINT:-http://localhost:4566}"
STACK="${STACK_NAME:-pav-edi-local}"
DEPLOY_BUCKET="${SAM_DEPLOY_BUCKET:-pav-edi-sam-deploy}"

export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1
export AWS_ENDPOINT_URL="$ENDPOINT"

cd "$ROOT/infra"

echo "Building SAM artifacts..."
sam build --template template.yaml

echo "Ensuring SAM deploy bucket exists on LocalStack..."
aws s3 mb "s3://${DEPLOY_BUCKET}" --region us-east-1 2>/dev/null || true

echo "Deploying to LocalStack at $ENDPOINT ..."
sam deploy \
  --stack-name "$STACK" \
  --s3-bucket "$DEPLOY_BUCKET" \
  --no-confirm-changeset \
  --no-fail-on-empty-changeset \
  --capabilities CAPABILITY_IAM \
  --region us-east-1 \
  --parameter-overrides Environment=local

echo "Stack $STACK deployed to LocalStack"
