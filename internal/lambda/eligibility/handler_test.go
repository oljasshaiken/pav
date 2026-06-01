package eligibility_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/edi"
	"github.com/pavillio/pav-edi/internal/lambda/eligibility"
	"github.com/pavillio/pav-edi/internal/pipeline"
	"github.com/pavillio/pav-edi/internal/repository"
	"github.com/pavillio/pav-edi/internal/testutil"
	"github.com/pavillio/pav-edi/pkg/x12"
)

func TestHandler_271GoldenPersists(t *testing.T) {
	pool := testutil.StartPostgres(t)
	claimID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	patientID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	testutil.InsertGoldenFixtureClaim(t, pool, claimID)

	raw := readFixture(t, "271_tx_golden.x12")
	h := &eligibility.Handler{
		Store: repository.New(pool),
		Object: &eligibility.MemoryObjectReader{
			Objects: map[string][]byte{
				"pav-edi-inbound/eligibility/271_tx.x12": raw,
			},
		},
	}

	res, err := h.Handle(context.Background(), pipeline.Parse271Request{
		S3Bucket: "pav-edi-inbound",
		S3Key:    "eligibility/271_tx.x12",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.PatientID != patientID.String() {
		t.Fatalf("patient_id = %q", res.PatientID)
	}
	if res.InquiryRef != "CLM-DEMO-001" {
		t.Fatalf("inquiry_ref = %q", res.InquiryRef)
	}
	if res.CoverageStatus != domain.CoverageActive {
		t.Fatalf("coverage = %q", res.CoverageStatus)
	}

	var stored struct {
		PatientID      uuid.UUID
		PayerID        string
		InquiryRef     string
		CoverageStatus string
		ServiceType    *string
		Response271    string
	}
	err = pool.QueryRow(context.Background(), `
SELECT patient_id, payer_id, inquiry_ref, coverage_status, service_type, response_271
FROM eligibility_responses
WHERE patient_id = $1
ORDER BY created_at DESC
LIMIT 1`, patientID).Scan(
		&stored.PatientID, &stored.PayerID, &stored.InquiryRef,
		&stored.CoverageStatus, &stored.ServiceType, &stored.Response271,
	)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Response271 != string(raw) {
		t.Fatal("response_271 not persisted")
	}
	if stored.ServiceType == nil || *stored.ServiceType != "42" {
		t.Fatalf("service_type = %v", stored.ServiceType)
	}
}

func TestEligibility270To271RoundTrip(t *testing.T) {
	pool := testutil.StartPostgres(t)
	claimID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	patientID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	testutil.InsertGoldenFixtureClaim(t, pool, claimID)

	cfg := load270Config(t)
	claimNumber := "CLM-DEMO-001"
	ctx := domain.ClaimContext{
		Claim: domain.Claim{
			ClaimNumber: &claimNumber,
			PayerID:     "TX-MCO-001",
		},
		Patient: domain.Patient{
			FirstName:   "Synthetic",
			LastName:    "Patient",
			MedicaidID:  "SYN-TX-00001",
			DateOfBirth: time.Date(1975, 6, 15, 0, 0, 0, 0, time.UTC),
		},
		Agency: domain.Agency{Name: "Demo Home Care TX"},
	}

	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	edi270, err := edi.Generate270(cfg.Envelope, cfg.Mappings, ctx, now)
	if err != nil {
		t.Fatalf("Generate270: %v", err)
	}
	segs, err := x12.SplitSegments(edi270)
	if err != nil {
		t.Fatal(err)
	}
	inquiryRef := findBHTReference(segs)
	if inquiryRef != "CLM-DEMO-001" {
		t.Fatalf("270 inquiry ref = %q", inquiryRef)
	}

	raw271 := readFixture(t, "271_tx_golden.x12")
	parsed271, err := x12.ParseEligibility271(string(raw271))
	if err != nil {
		t.Fatal(err)
	}
	if parsed271.InquiryRef != inquiryRef {
		t.Fatalf("271 inquiry ref %q != 270 ref %q", parsed271.InquiryRef, inquiryRef)
	}

	h := &eligibility.Handler{
		Store: repository.New(pool),
		Object: &eligibility.MemoryObjectReader{
			Objects: map[string][]byte{
				"pav-edi-inbound/eligibility/271_tx.x12": raw271,
			},
		},
	}
	res, err := h.Handle(context.Background(), pipeline.Parse271Request{
		S3Bucket: "pav-edi-inbound",
		S3Key:    "eligibility/271_tx.x12",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.PatientID != patientID.String() {
		t.Fatalf("patient_id = %q", res.PatientID)
	}
	if res.CoverageStatus != domain.CoverageActive {
		t.Fatalf("coverage = %q", res.CoverageStatus)
	}
}

func findBHTReference(segs []x12.Segment) string {
	for _, seg := range segs {
		if seg.Tag == "BHT" && len(seg.Elements) >= 3 && seg.Elements[0] == "0022" {
			return seg.Elements[2]
		}
	}
	return ""
}

func load270Config(t *testing.T) domain.PayerConfigBody {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "fixtures", "payer_config_270_tx.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg domain.PayerConfigBody
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "..", "docs", "fixtures", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
