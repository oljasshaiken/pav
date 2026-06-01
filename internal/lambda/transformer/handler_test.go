package transformer_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/lambda/transformer"
	"github.com/pavillio/pav-edi/internal/pipeline"
)

func TestHandler_matchesGoldenStrict(t *testing.T) {
	cfg := loadPayerConfigBody(t)
	req := pipeline.TransformRequest{
		ClaimID:       "00000000-0000-4000-8000-000000000001",
		ConfigVersion: 1,
		ClaimContext:  mappingTestClaimContext(),
		PayerConfig:   cfg,
		GeneratedAt:   goldenTime(),
	}
	h := &transformer.Handler{Now: goldenTime}
	res, err := h.Handle(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	goldenBytes, err := os.ReadFile("../../../docs/fixtures/837p_tx_golden.x12")
	if err != nil {
		t.Fatal(err)
	}
	if collapseEDI(res.Document.Raw) != collapseEDI(string(goldenBytes)) {
		t.Fatal("golden mismatch")
	}
}

func loadPayerConfigBody(t *testing.T) domain.PayerConfigBody {
	t.Helper()
	data, err := os.ReadFile("../../../docs/fixtures/payer_config_837p_tx.json")
	if err != nil {
		t.Fatal(err)
	}
	var cfg domain.PayerConfigBody
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func goldenTime() time.Time {
	return time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
}

func mappingTestClaimContext() domain.ClaimContext {
	claimNumber := "CLM-DEMO-001"
	amount := 100.00
	clockIn := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	return domain.ClaimContext{
		Claim: domain.Claim{
			ID:          uuid.MustParse("00000000-0000-4000-8000-000000000001"),
			ClaimNumber: &claimNumber,
		},
		Patient: domain.Patient{
			FirstName:  "Synthetic",
			LastName:   "Patient",
			MedicaidID: "SYN-TX-00001",
		},
		Agency: domain.Agency{Name: "Demo Home Care TX", NPI: "1234567890"},
		Visits: []domain.Visit{{ClockInTime: &clockIn, EVVStatus: "VERIFIED"}},
		ServiceLines: []domain.ClaimServiceLine{{
			ProcedureCode:  "T1019",
			Units:          4,
			Amount:         &amount,
			DiagnosisCodes: []string{"Z9999"},
		}},
	}
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
