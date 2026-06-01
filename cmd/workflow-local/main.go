package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pavillio/pav-edi/internal/lambda/load"
	"github.com/pavillio/pav-edi/internal/lambda/persist"
	"github.com/pavillio/pav-edi/internal/lambda/rules"
	"github.com/pavillio/pav-edi/internal/lambda/transformer"
	"github.com/pavillio/pav-edi/internal/platform"
	"github.com/pavillio/pav-edi/internal/repository"
	"github.com/pavillio/pav-edi/internal/workflow"
)

func compareDryRun() bool {
	return os.Getenv("COMPARE_DRY_RUN") == "1" || os.Getenv("COMPARE_DRY_RUN") == "true"
}

func main() {
	claimID := os.Getenv("CLAIM_ID")
	if claimID == "" {
		claimID = "00000000-0000-4000-8000-000000000001"
	}
	ctx := context.Background()
	cfg := platform.LoadConfig()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "postgres: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	mem := &persist.MemoryObjectStore{}
	store := repository.New(pool)
	now := platform.NowFromEnv()

	wf := &workflow.Outbound{
		Load:        &load.Handler{Store: store},
		Rules:       &rules.Handler{},
		Transform:   &transformer.Handler{Now: func() time.Time { return now }},
		Persist:     &persist.Handler{Store: store, Object: mem},
		Now:         func() time.Time { return now },
		SkipPersist: compareDryRun(),
		S3Bucket:    envOr("OUTBOUND_BUCKET", "pav-edi-outbound"),
		S3KeyPrefix: "",
	}

	result, err := wf.Run(ctx, claimID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "workflow: %v\n", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(result)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
