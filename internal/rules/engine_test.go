package rules

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/domain"
)

func TestEngine_Transform_producesRealX12(t *testing.T) {
	e := &RulesEngine{Now: func() time.Time { return goldenTime() }}
	cfg := loadPayerConfigBody(t)
	claimCtx := mappingTestClaimContext()

	doc, err := e.transformWithConfig(context.Background(), cfg, claimCtx, 1)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if doc.Raw == "" {
		t.Fatal("expected non-empty edi")
	}
	if doc.Raw[:3] != "ISA" {
		t.Fatalf("edi should start with ISA, got %q", doc.Raw[:min(20, len(doc.Raw))])
	}
	if doc.Engine != "rules" {
		t.Fatalf("engine = %q", doc.Engine)
	}
}

func TestEngine_Transform_matchesGoldenStrict(t *testing.T) {
	e := &RulesEngine{Now: func() time.Time { return goldenTime() }}
	cfg := loadPayerConfigBody(t)
	claimCtx := mappingTestClaimContext()

	doc, err := e.transformWithConfig(context.Background(), cfg, claimCtx, 1)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}

	goldenBytes, err := os.ReadFile("../../docs/fixtures/837p_tx_golden.x12")
	if err != nil {
		t.Fatal(err)
	}
	want := collapseEDI(string(goldenBytes))
	got := collapseEDI(doc.Raw)
	if got != want {
		t.Fatalf("golden mismatch\n%s", diffPrefix(got, want))
	}
}

func loadPayerConfigBody(t *testing.T) domain.PayerConfigBody {
	t.Helper()
	data, err := os.ReadFile("../../docs/fixtures/payer_config_837p_tx.json")
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
		Visits: []domain.Visit{{ClockInTime: &clockIn}},
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

func diffPrefix(got, want string) string {
	for i := 0; i < len(got) && i < len(want); i++ {
		if got[i] != want[i] {
			start := i - 20
			if start < 0 {
				start = 0
			}
			end := i + 40
			if end > len(got) {
				end = len(got)
			}
			wantEnd := end
			if wantEnd > len(want) {
				wantEnd = len(want)
			}
			return "got[" + got[start:end] + "] vs want[" + want[start:wantEnd] + "]"
		}
	}
	if len(got) != len(want) {
		return "length mismatch"
	}
	return "identical"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
