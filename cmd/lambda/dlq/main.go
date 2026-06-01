package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/pavillio/pav-edi/internal/lambda/dlq"
	"github.com/pavillio/pav-edi/internal/pipeline"
	"github.com/pavillio/pav-edi/internal/platform/observability"
	"github.com/pavillio/pav-edi/internal/queue"
)

func main() {
	observability.InitLambda()
	lambda.Start(handle)
}

func handle(ctx context.Context, req pipeline.DLQPublishRequest) error {
	if req.Error == nil {
		req.Error = &pipeline.WorkflowError{Code: "WORKFLOW_ERROR", Message: "unknown failure"}
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}
	if endpoint := os.Getenv("AWS_ENDPOINT_URL"); endpoint != "" {
		cfg.BaseEndpoint = aws.String(endpoint)
	}
	client := sqs.NewFromConfig(cfg)
	queueURL := os.Getenv("DLQ_URL")
	pub := queue.JSONPublisher(func(body []byte) error {
		_, err := client.SendMessage(ctx, &sqs.SendMessageInput{
			QueueUrl:               aws.String(queueURL),
			MessageBody:            aws.String(string(body)),
			MessageGroupId:         aws.String(req.PayerID),
			MessageDeduplicationId: aws.String(req.ClaimID + ":" + req.Error.Code),
		})
		return err
	})
	return (&dlq.Handler{Publisher: pub}).Handle(ctx, req)
}
