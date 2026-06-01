package domain_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/pavillio/pav-edi/internal/domain"
)

const (
	fixturePayerConfig  = "../../docs/fixtures/payer_config_837p_tx.json"
	fixtureTemplate     = "../../docs/fixtures/template_837p_base.json"
	fixtureOverride     = "../../docs/fixtures/override_837p_tx.json"
	fixtureGoldenNorm   = "../../docs/fixtures/837p_tx_golden.normalized.json"
	fixtureGoldenX12    = "../../docs/fixtures/837p_tx_golden.x12"
	seedClaimID         = "00000000-0000-4000-8000-000000000001"
	referencePayerID    = "TX-MCO-001"
	referenceX12Version = "005010X222A1"
)

type payerConfigMappings struct {
	Patient     map[string]any `json:"patient"`
	Agency      map[string]any `json:"agency"`
	Claim       map[string]any `json:"claim"`
	ServiceLine map[string]any `json:"service_line"`
	EVV         map[string]any `json:"evv"`
}

type validationRule struct {
	Field     string `json:"field"`
	Rule      string `json:"rule"`
	Condition string `json:"condition"`
}

type normalizedSegment struct {
	Loop     string   `json:"loop"`
	Segment  string   `json:"segment"`
	Elements []string `json:"elements"`
}

type normalizedGolden struct {
	ClaimID    string              `json:"claim_id"`
	X12Version string              `json:"x12_version"`
	PayerID    string              `json:"payer_id"`
	Segments   []normalizedSegment `json:"segments"`
}

func TestPhase1_PayerConfigFixture(t *testing.T) {
	data, err := os.ReadFile(fixturePayerConfig)
	if err != nil {
		t.Fatalf("read payer config: %v", err)
	}

	var cfg domain.PayerConfigBody
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal payer config: %v", err)
	}

	if cfg.X12Version != referenceX12Version {
		t.Fatalf("x12_version = %q, want %q", cfg.X12Version, referenceX12Version)
	}

	if len(cfg.Envelope) == 0 {
		t.Fatal("expected envelope block")
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(cfg.Envelope, &envelope); err != nil {
		t.Fatalf("envelope: %v", err)
	}
	for _, key := range []string{"isa", "gs"} {
		if _, ok := envelope[key]; !ok {
			t.Fatalf("envelope missing %q", key)
		}
	}

	var mappings payerConfigMappings
	if err := json.Unmarshal(cfg.Mappings, &mappings); err != nil {
		t.Fatalf("mappings: %v", err)
	}
	for _, section := range []struct {
		name string
		got  map[string]any
	}{
		{"patient", mappings.Patient},
		{"agency", mappings.Agency},
		{"claim", mappings.Claim},
		{"service_line", mappings.ServiceLine},
		{"evv", mappings.EVV},
	} {
		if len(section.got) == 0 {
			t.Fatalf("mappings.%s is required", section.name)
		}
	}

	if _, ok := mappings.Patient["loop_2010BA"]; !ok {
		t.Fatal("mappings.patient.loop_2010BA is required")
	}
	if _, ok := mappings.Agency["loop_2000A"]; !ok {
		t.Fatal("mappings.agency.loop_2000A is required")
	}
	if _, ok := mappings.Claim["loop_2300"]; !ok {
		t.Fatal("mappings.claim.loop_2300 is required")
	}
	if _, ok := mappings.ServiceLine["loop_2400"]; !ok {
		t.Fatal("mappings.service_line.loop_2400 is required")
	}
	if _, ok := mappings.EVV["custom_ref_segment"]; !ok {
		t.Fatal("mappings.evv.custom_ref_segment is required")
	}

	var rules []validationRule
	if err := json.Unmarshal(cfg.ValidationRules, &rules); err != nil {
		t.Fatalf("validation_rules: %v", err)
	}
	foundDiagnosis := false
	for _, rule := range rules {
		if rule.Field == "diagnosis_code" && rule.Rule == "required" &&
			strings.Contains(rule.Condition, "home_health") {
			foundDiagnosis = true
			break
		}
	}
	if !foundDiagnosis {
		t.Fatal("validation_rules must require diagnosis_code for home_health")
	}
}

func TestPhase1_TemplateFixtures(t *testing.T) {
	tmplData, err := os.ReadFile(fixtureTemplate)
	if err != nil {
		t.Fatalf("read template: %v", err)
	}
	var tmpl struct {
		Loops []struct {
			ID       string `json:"id"`
			Segments []struct {
				Tag      string   `json:"tag"`
				Elements []string `json:"elements"`
			} `json:"segments"`
		} `json:"loops"`
	}
	if err := json.Unmarshal(tmplData, &tmpl); err != nil {
		t.Fatalf("unmarshal template: %v", err)
	}
	if len(tmpl.Loops) == 0 {
		t.Fatal("template must define loops")
	}

	requiredLoops := map[string]bool{"2000A": false, "2010BA": false, "2300": false, "2400": false}
	for _, loop := range tmpl.Loops {
		if _, ok := requiredLoops[loop.ID]; ok {
			requiredLoops[loop.ID] = true
			if len(loop.Segments) == 0 {
				t.Fatalf("loop %s must have at least one segment", loop.ID)
			}
		}
	}
	for id, found := range requiredLoops {
		if !found {
			t.Fatalf("template missing loop %s", id)
		}
	}

	overrideData, err := os.ReadFile(fixtureOverride)
	if err != nil {
		t.Fatalf("read override: %v", err)
	}
	var override struct {
		Mappings payerConfigMappings `json:"mappings"`
	}
	if err := json.Unmarshal(overrideData, &override); err != nil {
		t.Fatalf("unmarshal override: %v", err)
	}
	for _, section := range []struct {
		name string
		got  map[string]any
	}{
		{"patient", override.Mappings.Patient},
		{"agency", override.Mappings.Agency},
		{"claim", override.Mappings.Claim},
		{"service_line", override.Mappings.ServiceLine},
		{"evv", override.Mappings.EVV},
	} {
		if len(section.got) == 0 {
			t.Fatalf("override.mappings.%s is required", section.name)
		}
	}

	payerData, err := os.ReadFile(fixturePayerConfig)
	if err != nil {
		t.Fatal(err)
	}
	var payer struct {
		Mappings payerConfigMappings `json:"mappings"`
	}
	if err := json.Unmarshal(payerData, &payer); err != nil {
		t.Fatal(err)
	}
	if string(mustJSON(t, payer.Mappings.Patient)) != string(mustJSON(t, override.Mappings.Patient)) {
		t.Fatal("override patient mappings must match payer config")
	}
	if string(mustJSON(t, payer.Mappings.Agency)) != string(mustJSON(t, override.Mappings.Agency)) {
		t.Fatal("override agency mappings must match payer config")
	}
}

func TestPhase1_GoldenNormalizedFixture(t *testing.T) {
	data, err := os.ReadFile(fixtureGoldenNorm)
	if err != nil {
		t.Fatalf("read normalized golden: %v", err)
	}

	var golden normalizedGolden
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatalf("unmarshal normalized golden: %v", err)
	}

	if golden.ClaimID != seedClaimID {
		t.Fatalf("claim_id = %q, want %q", golden.ClaimID, seedClaimID)
	}
	if golden.PayerID != referencePayerID {
		t.Fatalf("payer_id = %q, want %q", golden.PayerID, referencePayerID)
	}
	if golden.X12Version != referenceX12Version {
		t.Fatalf("x12_version = %q, want %q", golden.X12Version, referenceX12Version)
	}
	if len(golden.Segments) == 0 {
		t.Fatal("segments manifest must not be empty")
	}

	requiredSegments := map[string]bool{
		"ISA": false, "GS": false, "ST": false,
		"HL": false, "NM1": false, "CLM": false, "SV1": false, "REF": false,
		"SE": false, "GE": false, "IEA": false,
	}
	for _, seg := range golden.Segments {
		if seg.Loop == "" {
			t.Fatal("segment entry missing loop")
		}
		if seg.Segment == "" {
			t.Fatal("segment entry missing segment tag")
		}
		if len(seg.Elements) == 0 {
			t.Fatalf("segment %s in loop %s must have elements", seg.Segment, seg.Loop)
		}
		if _, ok := requiredSegments[seg.Segment]; ok {
			requiredSegments[seg.Segment] = true
		}
	}
	for tag, found := range requiredSegments {
		if !found {
			t.Fatalf("normalized manifest missing segment %s", tag)
		}
	}
}

func TestPhase1_GoldenX12Fixture(t *testing.T) {
	data, err := os.ReadFile(fixtureGoldenX12)
	if err != nil {
		t.Fatalf("read golden x12: %v", err)
	}
	raw := strings.TrimSpace(string(data))
	if raw == "" {
		t.Fatal("golden x12 must not be empty")
	}
	if !strings.HasPrefix(raw, "ISA*") {
		t.Fatalf("golden x12 must start with ISA segment, got %q", raw[:min(20, len(raw))])
	}
	if !strings.Contains(raw, "~") {
		t.Fatal("golden x12 must use ~ segment terminator")
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
