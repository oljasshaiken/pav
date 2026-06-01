package main

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pavillio/pav-edi/internal/lambda/load"
	"github.com/pavillio/pav-edi/internal/pipeline"
	"github.com/pavillio/pav-edi/internal/platform"
	"github.com/pavillio/pav-edi/internal/repository"
)

func main() {
	lambda.Start(handle)
}

func handle(ctx context.Context, req pipeline.LoadClaimRequest) (pipeline.LoadClaimResult, error) {
	pool, err := pgxpool.New(ctx, platform.LoadConfig().DatabaseURL)
	if err != nil {
		return pipeline.LoadClaimResult{}, err
	}
	defer pool.Close()
	return (&load.Handler{Store: repository.New(pool)}).Handle(ctx, req)
}
