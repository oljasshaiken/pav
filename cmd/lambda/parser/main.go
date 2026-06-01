package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pavillio/pav-edi/internal/lambda/parser"
	"github.com/pavillio/pav-edi/internal/pipeline"
	"github.com/pavillio/pav-edi/internal/platform"
	"github.com/pavillio/pav-edi/internal/platform/observability"
	"github.com/pavillio/pav-edi/internal/repository"
)

func main() {
	observability.InitLambda()
	lambda.Start(handle)
}

func handle(ctx context.Context, req pipeline.Parse277Request) (pipeline.Parse277Result, error) {
	if req.S3Bucket == "" {
		req.S3Bucket = os.Getenv("INBOUND_BUCKET")
	}
	pool, err := pgxpool.New(ctx, platform.LoadConfig().DatabaseURL)
	if err != nil {
		return pipeline.Parse277Result{}, err
	}
	defer pool.Close()

	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return pipeline.Parse277Result{}, err
	}
	if endpoint := os.Getenv("AWS_ENDPOINT_URL"); endpoint != "" {
		cfg.BaseEndpoint = aws.String(endpoint)
	}
	s3Client := s3.NewFromConfig(cfg)

	return (&parser.Handler{
		Store:  repository.New(pool),
		Object: parser.NewS3ObjectReader(s3Client),
	}).Handle(ctx, req)
}
