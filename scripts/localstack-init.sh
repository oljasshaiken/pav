#!/usr/bin/env bash
# LocalStack ready hook: create S3 buckets and SQS FIFO queue for EDI pipeline dev.
set -euo pipefail

awslocal s3 mb s3://pav-edi-inbound 2>/dev/null || true
awslocal s3 mb s3://pav-edi-outbound 2>/dev/null || true
awslocal s3 mb s3://pav-edi-sam-deploy 2>/dev/null || true

awslocal sqs create-queue \
  --queue-name pav-edi-claims-dlq.fifo \
  --attributes FifoQueue=true,ContentBasedDeduplication=true \
  2>/dev/null || true

DLQ_ARN=$(awslocal sqs get-queue-attributes \
  --queue-url "$(awslocal sqs get-queue-url --queue-name pav-edi-claims-dlq.fifo --query QueueUrl --output text)" \
  --attribute-names QueueArn --query 'Attributes.QueueArn' --output text)

awslocal sqs create-queue \
  --queue-name pav-edi-claims.fifo \
  --attributes "FifoQueue=true,ContentBasedDeduplication=true,RedrivePolicy={\"deadLetterTargetArn\":\"${DLQ_ARN}\",\"maxReceiveCount\":\"3\"}" \
  2>/dev/null || true

echo "LocalStack EDI resources ready"
