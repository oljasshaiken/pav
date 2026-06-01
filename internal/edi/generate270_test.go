package edi_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/edi"
)

const fixture270Config = "../../docs/fixtures/payer_config_270_tx.json"

func TestGenerate270_matchesGolden(t *testing.T) {
	data, err := os.ReadFile(fixture270Config)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg domain.PayerConfigBody
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

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
	got, err := edi.Generate270(cfg.Envelope, cfg.Mappings, ctx, now)
	if err != nil {
		t.Fatalf("Generate270: %v", err)
	}

	goldenBytes, err := os.ReadFile("../../docs/fixtures/270_tx_golden.x12")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	want := collapseEDI(string(goldenBytes))
	if collapseEDI(got) != want {
		t.Fatalf("270 output mismatch")
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
