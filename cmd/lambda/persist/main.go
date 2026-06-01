package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pavillio/pav-edi/internal/lambda/persist"
	"github.com/pavillio/pav-edi/internal/pipeline"
	"github.com/pavillio/pav-edi/internal/platform"
	"github.com/pavillio/pav-edi/internal/repository"
)

func main() {
	lambda.Start(handle)
}

func handle(ctx context.Context, req pipeline.PersistRequest) (pipeline.PersistResult, error) {
	if req.S3Bucket == "" {
		req.S3Bucket = os.Getenv("OUTBOUND_BUCKET")
	}
	pool, err := pgxpool.New(ctx, platform.LoadConfig().DatabaseURL)
	if err != nil {
		return pipeline.PersistResult{}, err
	}
	defer pool.Close()
	return (&persist.Handler{
		Store:  repository.New(pool),
		Object: persist.NoopObjectStore{},
	}).Handle(ctx, req)
}
