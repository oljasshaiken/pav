package pipeline_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/pipeline"
)

func TestGenerateRequest_roundTripJSON(t *testing.T) {
	id := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	req := pipeline.GenerateRequest{ClaimID: id.String()}
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var decoded pipeline.GenerateRequest
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ClaimID != req.ClaimID {
		t.Fatalf("claim_id = %q", decoded.ClaimID)
	}
}

func TestTransformRequest_roundTripJSON(t *testing.T) {
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	req := pipeline.TransformRequest{
		ClaimID:       "00000000-0000-4000-8000-000000000001",
		ConfigVersion: 1,
		ClaimContext: domain.ClaimContext{
			Claim: domain.Claim{PayerID: "TX-MCO-001", State: "TX"},
		},
		PayerConfig: domain.PayerConfigBody{X12Version: "005010X222A1"},
		GeneratedAt: now,
	}
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var decoded pipeline.TransformRequest
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.PayerConfig.X12Version != "005010X222A1" {
		t.Fatalf("x12_version = %q", decoded.PayerConfig.X12Version)
	}
}

func TestWorkflowError_marshal(t *testing.T) {
	err := pipeline.WorkflowError{Code: "VALIDATION_FAILED", Message: "diagnosis required", RuleID: "diagnosis_required"}
	raw, errJSON := json.Marshal(err)
	if errJSON != nil {
		t.Fatal(errJSON)
	}
	if !json.Valid(raw) {
		t.Fatal("invalid json")
	}
}
