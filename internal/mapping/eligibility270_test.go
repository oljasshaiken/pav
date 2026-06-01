package mapping_test

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/mapping"
	"github.com/pavillio/pav-edi/pkg/x12"
)

const fixture270Config = "../../docs/fixtures/payer_config_270_tx.json"

func TestBuild270BodySegments_matchesGoldenBody(t *testing.T) {
	mappings := load270Mappings(t)
	ctx := synthetic270ClaimContext(t)
	clock := goldenClockOptions270()

	got, err := mapping.Build270BodySegments(mappings, ctx, clock)
	if err != nil {
		t.Fatalf("Build270BodySegments: %v", err)
	}

	want := golden270BodySegments()
	if len(got) != len(want) {
		t.Fatalf("segment count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Tag != want[i].Tag {
			t.Fatalf("segment[%d] tag = %q, want %q", i, got[i].Tag, want[i].Tag)
		}
		if !reflect.DeepEqual(got[i].Elements, want[i].Elements) {
			t.Fatalf("segment[%d] %s elements = %#v, want %#v", i, got[i].Tag, got[i].Elements, want[i].Elements)
		}
	}
}

func TestBuild270BodySegments_integrationWithX12Builder(t *testing.T) {
	mappings := load270Mappings(t)
	ctx := synthetic270ClaimContext(t)
	clock := goldenClockOptions270()

	body, err := mapping.Build270BodySegments(mappings, ctx, clock)
	if err != nil {
		t.Fatalf("Build270BodySegments: %v", err)
	}

	cfg := load270EnvelopeConfig(t)
	b := x12.NewBuilder(cfg, clock, x12.Separators{}).WithX12Version("005010X279A1")
	b.AppendBody(body...)

	goldenBytes, err := os.ReadFile("../../docs/fixtures/270_tx_golden.x12")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	want := collapseEDI(string(goldenBytes))
	got := collapseEDI(b.Build())
	if got != want {
		t.Fatalf("full 270 interchange mismatch\nfirst diff near output:\n%s", diffPrefix(got, want))
	}
}

func load270Mappings(t *testing.T) json.RawMessage {
	t.Helper()
	data, err := os.ReadFile(fixture270Config)
	if err != nil {
		t.Fatalf("read 270 payer config: %v", err)
	}
	var cfg domain.PayerConfigBody
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal 270 payer config: %v", err)
	}
	return cfg.Mappings
}

func load270EnvelopeConfig(t *testing.T) x12.EnvelopeConfig {
	t.Helper()
	data, err := os.ReadFile(fixture270Config)
	if err != nil {
		t.Fatalf("read 270 payer config: %v", err)
	}
	var cfg struct {
		Envelope x12.EnvelopeConfig `json:"envelope"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	return cfg.Envelope
}

func goldenClockOptions270() x12.FixedClockOptions {
	return x12.FixedClockOptions{
		ISADate:        "260531",
		ISATime:        "1200",
		GSDate:         "20260531",
		GSTime:         "1200",
		ISAControl:     "000000001",
		GSControl:      "1",
		STControl:      "0001",
		SESegmentCount: 11,
	}
}

func golden270BodySegments() []x12.Segment {
	return []x12.Segment{
		{Tag: "BHT", Elements: []string{"0022", "13", "CLM-DEMO-001", "20260531", "1200"}},
		{Tag: "HL", Elements: []string{"1", "", "20", "1"}},
		{Tag: "NM1", Elements: []string{"PR", "2", "TX Medicaid MCO", "", "", "", "", "PI", "TX-MCO-001"}},
		{Tag: "HL", Elements: []string{"2", "1", "21", "1"}},
		{Tag: "NM1", Elements: []string{"1P", "2", "Demo Home Care TX", "", "", "", "", "XX", "1234567890"}},
		{Tag: "HL", Elements: []string{"3", "2", "22", "0"}},
		{Tag: "NM1", Elements: []string{"IL", "1", "Patient", "Synthetic", "", "", "", "MI", "SYN-TX-00001"}},
		{Tag: "DMG", Elements: []string{"D8", "19750615"}},
		{Tag: "DTP", Elements: []string{"291", "D8", "20260531"}},
		{Tag: "EQ", Elements: []string{"42"}},
	}
}

func synthetic270ClaimContext(t *testing.T) domain.ClaimContext {
	t.Helper()
	ctx := syntheticClaimContext(t)
	ctx.Claim.PayerID = "TX-MCO-001"
	ctx.Patient.DateOfBirth = time.Date(1975, 6, 15, 0, 0, 0, 0, time.UTC)
	return ctx
}
