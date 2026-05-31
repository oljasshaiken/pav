package domain_test

import (
	"encoding/json"
	"os"
	"testing"
)

func TestTemplateFixture_Unmarshal(t *testing.T) {
	data, err := os.ReadFile("../../docs/fixtures/template_837p_base.json")
	if err != nil {
		t.Fatal(err)
	}
	var tmpl map[string]any
	if err := json.Unmarshal(data, &tmpl); err != nil {
		t.Fatalf("template: %v", err)
	}
	if _, ok := tmpl["loops"]; !ok {
		t.Fatal("expected loops in template")
	}

	override, err := os.ReadFile("../../docs/fixtures/override_837p_tx.json")
	if err != nil {
		t.Fatal(err)
	}
	var mapper map[string]any
	if err := json.Unmarshal(override, &mapper); err != nil {
		t.Fatalf("override: %v", err)
	}
	if _, ok := mapper["mappings"]; !ok {
		t.Fatal("expected mappings in override")
	}
}
