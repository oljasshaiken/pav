package x12_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/pavillio/pav-edi/pkg/x12"
)

const (
	fixtureGoldenX12  = "../../docs/fixtures/837p_tx_golden.x12"
	fixtureGoldenNorm = "../../docs/fixtures/837p_tx_golden.normalized.json"
)

func TestNormalize_goldenX12_matchesFixture(t *testing.T) {
	raw, err := os.ReadFile(fixtureGoldenX12)
	if err != nil {
		t.Fatalf("read golden x12: %v", err)
	}
	normBytes, err := os.ReadFile(fixtureGoldenNorm)
	if err != nil {
		t.Fatalf("read normalized golden: %v", err)
	}

	var want x12.Manifest
	if err := json.Unmarshal(normBytes, &want); err != nil {
		t.Fatalf("unmarshal normalized golden: %v", err)
	}

	got, err := x12.Normalize(string(raw))
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	gotJSON, err := json.Marshal(got.Segments)
	if err != nil {
		t.Fatal(err)
	}
	wantJSON, err := json.Marshal(want.Segments)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("segments mismatch:\ngot  %s\nwant %s", gotJSON, wantJSON)
	}
}

func TestStripDynamicFields_replacesEnvelopeControls(t *testing.T) {
	raw, err := os.ReadFile(fixtureGoldenX12)
	if err != nil {
		t.Fatal(err)
	}
	m, err := x12.Normalize(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	x12.StripDynamicFields(&m)

	isa := m.Segments[0]
	if isa.Segment != "ISA" {
		t.Fatalf("first segment = %s", isa.Segment)
	}
	if isa.Elements[8] != x12.DynamicPlaceholder || isa.Elements[9] != x12.DynamicPlaceholder {
		t.Fatalf("ISA date/time not stripped: %v %v", isa.Elements[8], isa.Elements[9])
	}
	if isa.Elements[12] != x12.DynamicPlaceholder {
		t.Fatalf("ISA control not stripped: %v", isa.Elements[12])
	}
}
