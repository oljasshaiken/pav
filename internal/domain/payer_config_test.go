package domain_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/pavillio/pav-edi/internal/domain"
)

func TestPayerConfig_UnmarshalFixture(t *testing.T) {
	data, err := os.ReadFile("../../docs/fixtures/payer_config_837p_tx.json")
	if err != nil {
		t.Fatal(err)
	}
	var cfg domain.PayerConfigBody
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cfg.X12Version != "005010X222A1" {
		t.Fatalf("x12_version = %q", cfg.X12Version)
	}
	if len(cfg.Mappings) == 0 {
		t.Fatal("expected mappings")
	}
}
