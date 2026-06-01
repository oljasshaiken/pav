package workflow_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/lambda/parser"
	"github.com/pavillio/pav-edi/internal/pipeline"
	"github.com/pavillio/pav-edi/internal/repository"
	"github.com/pavillio/pav-edi/internal/testutil"
	"github.com/pavillio/pav-edi/internal/workflow"
)

func TestInboundWorkflow_277Golden(t *testing.T) {
	pool := testutil.StartPostgres(t)
	claimID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	testutil.InsertGoldenFixtureClaim(t, pool, claimID)

	raw := readAckFixture(t, "277_tx_golden.x12")
	wf := &workflow.Inbound{
		Parser: &parser.Handler{
			Store: repository.New(pool),
			Object: &parser.MemoryObjectReader{
				Objects: map[string][]byte{
					"pav-edi-inbound/acks/277_tx.x12": raw,
				},
			},
		},
	}

	res, err := wf.Run(context.Background(), pipeline.Parse277Request{
		S3Bucket: "pav-edi-inbound",
		S3Key:    "acks/277_tx.x12",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ClaimID != claimID.String() {
		t.Fatalf("claim_id = %q", res.ClaimID)
	}

	var stored *string
	err = pool.QueryRow(context.Background(), `SELECT response_277 FROM claims WHERE id = $1`, claimID).Scan(&stored)
	if err != nil {
		t.Fatal(err)
	}
	if stored == nil || *stored != string(raw) {
		t.Fatal("response_277 not persisted")
	}
}

func TestInboundWorkflow_999WithExplicitClaimID(t *testing.T) {
	pool := testutil.StartPostgres(t)
	claimID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	testutil.InsertGoldenFixtureClaim(t, pool, claimID)

	raw := readAckFixture(t, "999_tx_golden.x12")
	wf := &workflow.Inbound{
		Parser: &parser.Handler{
			Store: repository.New(pool),
			Object: &parser.MemoryObjectReader{
				Objects: map[string][]byte{
					"pav-edi-inbound/acks/999_tx.x12": raw,
				},
			},
		},
	}

	res, err := wf.Run(context.Background(), pipeline.Parse277Request{
		S3Bucket: "pav-edi-inbound",
		S3Key:    "acks/999_tx.x12",
		ClaimID:  claimID.String(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ClaimID != claimID.String() {
		t.Fatalf("claim_id = %q", res.ClaimID)
	}
}

func readAckFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "docs", "fixtures", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
