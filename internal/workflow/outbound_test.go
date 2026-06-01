package workflow_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pavillio/pav-edi/internal/lambda/load"
	"github.com/pavillio/pav-edi/internal/lambda/persist"
	"github.com/pavillio/pav-edi/internal/lambda/rules"
	"github.com/pavillio/pav-edi/internal/lambda/transformer"
	"github.com/pavillio/pav-edi/internal/repository"
	"github.com/pavillio/pav-edi/internal/testutil"
	"github.com/pavillio/pav-edi/internal/workflow"
)

func TestOutboundWorkflow_TXGoldenE2E(t *testing.T) {
	pool := testutil.StartPostgres(t)
	claimID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	testutil.InsertGoldenFixtureClaim(t, pool, claimID)
	testutil.InsertPayerConfig(t, pool)

	mem := &persist.MemoryObjectStore{}
	wf := newOutboundWorkflow(pool, mem, goldenTime())

	result, err := wf.Run(context.Background(), claimID.String())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(result.EDI, "ISA") {
		t.Fatalf("edi = %q", result.EDI[:min(20, len(result.EDI))])
	}

	var x12Stored string
	err = pool.QueryRow(context.Background(), `SELECT x12_837 FROM claims WHERE id = $1`, claimID).Scan(&x12Stored)
	if err != nil {
		t.Fatal(err)
	}
	if x12Stored == "" {
		t.Fatal("x12_837 not persisted")
	}

	key := "pav-edi-outbound/" + claimID.String() + ".837"
	if _, ok := mem.Objects[key]; !ok {
		t.Fatalf("s3 object missing %q", key)
	}

	goldenBytes, err := os.ReadFile("../../docs/fixtures/837p_tx_golden.x12")
	if err != nil {
		t.Fatal(err)
	}
	if collapseEDI(result.EDI) != collapseEDI(string(goldenBytes)) {
		t.Fatal("workflow edi does not match golden")
	}
}

func TestOutboundWorkflow_SkipPersistNoDBWrite(t *testing.T) {
	pool := testutil.StartPostgres(t)
	claimID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	testutil.InsertGoldenFixtureClaim(t, pool, claimID)
	testutil.InsertPayerConfig(t, pool)

	var attemptBefore int32
	err := pool.QueryRow(context.Background(), `SELECT submission_attempt FROM claims WHERE id = $1`, claimID).Scan(&attemptBefore)
	if err != nil {
		t.Fatal(err)
	}

	wf := newOutboundWorkflow(pool, &persist.MemoryObjectStore{}, goldenTime())
	wf.SkipPersist = true
	result, err := wf.Run(context.Background(), claimID.String())
	if err != nil {
		t.Fatal(err)
	}
	if result.EDI == "" {
		t.Fatal("expected edi")
	}

	var attemptAfter int32
	var x12Stored *string
	err = pool.QueryRow(context.Background(), `SELECT submission_attempt, x12_837 FROM claims WHERE id = $1`, claimID).
		Scan(&attemptAfter, &x12Stored)
	if err != nil {
		t.Fatal(err)
	}
	if attemptAfter != attemptBefore {
		t.Fatalf("submission_attempt changed: %d -> %d", attemptBefore, attemptAfter)
	}
	if x12Stored != nil && *x12Stored != "" {
		t.Fatal("x12_837 should not be written in compare dry-run")
	}
}

func newOutboundWorkflow(pool *pgxpool.Pool, mem *persist.MemoryObjectStore, now time.Time) *workflow.Outbound {
	store := repository.New(pool)
	return &workflow.Outbound{
		Load:        &load.Handler{Store: store},
		Rules:       &rules.Handler{},
		Transform:   &transformer.Handler{Now: func() time.Time { return now }},
		Persist:     &persist.Handler{Store: store, Object: mem},
		Now:         func() time.Time { return now },
		S3Bucket:    "pav-edi-outbound",
		S3KeyPrefix: "",
	}
}

func goldenTime() time.Time {
	return time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
}

func collapseEDI(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\n' {
			out = append(out, s[i])
		}
	}
	return string(out)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
