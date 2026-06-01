package workflow_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pavillio/pav-edi/internal/lambda/persist"
	"github.com/pavillio/pav-edi/internal/pipeline"
	"github.com/pavillio/pav-edi/internal/queue"
	"github.com/pavillio/pav-edi/internal/repository"
	rulesengine "github.com/pavillio/pav-edi/internal/rules"
	"github.com/pavillio/pav-edi/internal/testutil"
	"github.com/pavillio/pav-edi/internal/validation"
	"github.com/pavillio/pav-edi/internal/workflow"
)

func TestOutboundWorkflow_validationFailurePublishesDLQ(t *testing.T) {
	pool := testutil.StartPostgres(t)
	claimID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	testutil.InsertGoldenFixtureClaim(t, pool, claimID)
	testutil.InsertPayerConfig(t, pool)

	_, err := pool.Exec(context.Background(), `
UPDATE claim_service_lines SET diagnosis_codes = NULL WHERE claim_id = $1`, claimID)
	if err != nil {
		t.Fatal(err)
	}

	dlqPub := &queue.MemoryPublisher{}
	wf := newOutboundWorkflowWithDLQ(pool, dlqPub, goldenTime())

	_, err = wf.Run(context.Background(), claimID.String())
	if err == nil {
		t.Fatal("expected validation error")
	}

	msg, ok := dlqPub.Last()
	if !ok {
		t.Fatal("expected DLQ message")
	}
	if msg.Code != "VALIDATION_FAILED" {
		t.Fatalf("code = %q", msg.Code)
	}
	if msg.PayerID != "TX-MCO-001" {
		t.Fatalf("payer_id = %q", msg.PayerID)
	}
	if msg.Phase != string(pipeline.RulesPhasePreTransform) {
		t.Fatalf("phase = %q", msg.Phase)
	}
}

func TestRulesWorkflow_matchesHTTPPipeline(t *testing.T) {
	pool := testutil.StartPostgres(t)
	claimID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	testutil.InsertGoldenFixtureClaim(t, pool, claimID)
	testutil.InsertPayerConfig(t, pool)
	store := repository.New(pool)
	ctx := context.Background()
	now := goldenTime()

	wfDoc, err := newOutboundWorkflow(pool, &persist.MemoryObjectStore{}, now).Run(ctx, claimID.String())
	if err != nil {
		t.Fatal(err)
	}

	gen := pipeline.Generator{
		Store:    store,
		Engine:   &rulesengine.RulesEngine{Store: store, Now: func() time.Time { return now }},
		Validate: validation.ConfigPipeline{},
	}
	httpDoc, err := gen.Generate(ctx, claimID)
	if err != nil {
		t.Fatal(err)
	}

	if wfDoc.EDI != httpDoc.Raw {
		t.Fatal("workflow EDI differs from HTTP pipeline EDI")
	}
}

func newOutboundWorkflowWithDLQ(pool *pgxpool.Pool, dlq *queue.MemoryPublisher, now time.Time) *workflow.Outbound {
	wf := newOutboundWorkflow(pool, &persist.MemoryObjectStore{}, now)
	wf.DLQ = dlq
	return wf
}
