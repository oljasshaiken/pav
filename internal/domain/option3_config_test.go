package domain_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/pavillio/pav-edi/internal/domain"
)

func TestOption3_TXFixtureCELRules(t *testing.T) {
	data, err := os.ReadFile(fixturePayerConfig)
	if err != nil {
		t.Fatal(err)
	}
	var cfg domain.PayerConfigBody
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	validation, err := cfg.CELValidationRules()
	if err != nil {
		t.Fatalf("CEL validation rules: %v", err)
	}
	if len(validation) == 0 {
		t.Fatal("expected at least one CEL validation rule")
	}
	foundDiagnosis := false
	for _, rule := range validation {
		if rule.ID == "diagnosis_required" && rule.CEL != "" && rule.Message != "" {
			foundDiagnosis = true
		}
	}
	if !foundDiagnosis {
		t.Fatal("expected diagnosis_required CEL validation rule")
	}

	evv, err := cfg.CELEvvRules()
	if err != nil {
		t.Fatalf("EVV rules: %v", err)
	}
	if len(evv) == 0 {
		t.Fatal("expected at least one evv_rule")
	}
	foundVerified := false
	for _, rule := range evv {
		if rule.ID == "evv_verified" && rule.CEL != "" {
			foundVerified = true
		}
	}
	if !foundVerified {
		t.Fatal("expected evv_verified rule")
	}
}

func TestOption3_LegacyValidationRulesUnmarshal(t *testing.T) {
	raw := json.RawMessage(`[
		{"field":"diagnosis_code","rule":"required","condition":"service_type = 'home_health'"}
	]`)
	rules, err := domain.ParseLegacyValidationRules(raw)
	if err != nil {
		t.Fatalf("parse legacy: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("rules = %d, want 1", len(rules))
	}
	if rules[0].Field != "diagnosis_code" || rules[0].Rule != "required" {
		t.Fatalf("unexpected rule: %+v", rules[0])
	}
}

func TestOption3_PayerConfigSchemaFile(t *testing.T) {
	data, err := os.ReadFile("../../docs/schema/payer_config_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("schema must be valid JSON: %v", err)
	}
	if schema["$schema"] == nil {
		t.Fatal("schema missing $schema")
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema missing properties")
	}
	for _, key := range []string{"x12_version", "envelope", "mappings", "evv_rules", "validation_rules", "business_rules"} {
		if _, ok := props[key]; !ok {
			t.Fatalf("schema properties missing %q", key)
		}
	}
}
