package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pavillio/pav-edi/internal/lambda/dispatch"
	"github.com/pavillio/pav-edi/internal/lambda/load"
	"github.com/pavillio/pav-edi/internal/lambda/persist"
	"github.com/pavillio/pav-edi/internal/lambda/rules"
	"github.com/pavillio/pav-edi/internal/lambda/transformer"
	"github.com/pavillio/pav-edi/internal/platform"
	"github.com/pavillio/pav-edi/internal/repository"
	"github.com/pavillio/pav-edi/internal/workflow"
)

func main() {
	lambda.Start(handle)
}

func handle(ctx context.Context, raw []byte) error {
	pool, err := pgxpool.New(ctx, platform.LoadConfig().DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	store := repository.New(pool)
	wf := &workflow.Outbound{
		Load:      &load.Handler{Store: store},
		Rules:     &rules.Handler{},
		Transform: &transformer.Handler{},
		Persist: &persist.Handler{
			Store:  store,
			Object: persist.NoopObjectStore{},
		},
		S3Bucket: os.Getenv("OUTBOUND_BUCKET"),
	}

	return (&dispatch.Handler{
		Workflow: dispatch.WorkflowAdapter{
			RunFn: func(ctx context.Context, claimID string) error {
				_, err := wf.Run(ctx, claimID)
				return err
			},
		},
	}).HandleEvent(ctx, raw)
}
