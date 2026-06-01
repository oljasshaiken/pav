package mapping_test

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/mapping"
	"github.com/pavillio/pav-edi/pkg/x12"
)

const fixturePayerConfig = "../../docs/fixtures/payer_config_837p_tx.json"

func TestBuildBodySegments_matchesGoldenBody(t *testing.T) {
	mappings := loadMappings(t)
	ctx := syntheticClaimContext(t)
	clock := goldenClockOptions()

	got, err := mapping.BuildBodySegments(mappings, ctx, clock)
	if err != nil {
		t.Fatalf("BuildBodySegments: %v", err)
	}

	want := goldenBodySegments()
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

func TestBuildBodySegments_integrationWithX12Builder(t *testing.T) {
	mappings := loadMappings(t)
	ctx := syntheticClaimContext(t)
	clock := goldenClockOptions()

	body, err := mapping.BuildBodySegments(mappings, ctx, clock)
	if err != nil {
		t.Fatalf("BuildBodySegments: %v", err)
	}

	cfg := loadEnvelopeConfig(t)
	b := x12.NewBuilder(cfg, clock, x12.Separators{})
	b.AppendBody(body...)

	goldenBytes, err := os.ReadFile("../../docs/fixtures/837p_tx_golden.x12")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	want := collapseEDI(string(goldenBytes))
	got := collapseEDI(b.Build())
	if got != want {
		t.Fatalf("full interchange mismatch\nfirst diff near output:\n%s", diffPrefix(got, want))
	}
}

func loadMappings(t *testing.T) json.RawMessage {
	t.Helper()
	data, err := os.ReadFile(fixturePayerConfig)
	if err != nil {
		t.Fatalf("read payer config: %v", err)
	}
	var cfg domain.PayerConfigBody
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal payer config: %v", err)
	}
	return cfg.Mappings
}

func loadEnvelopeConfig(t *testing.T) x12.EnvelopeConfig {
	t.Helper()
	data, err := os.ReadFile(fixturePayerConfig)
	if err != nil {
		t.Fatalf("read payer config: %v", err)
	}
	var cfg struct {
		Envelope x12.EnvelopeConfig `json:"envelope"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	return cfg.Envelope
}

func goldenClockOptions() x12.FixedClockOptions {
	return x12.FixedClockOptions{
		ISADate:        "260531",
		ISATime:        "1200",
		GSDate:         "20260531",
		GSTime:         "1200",
		ISAControl:     "000000001",
		GSControl:      "1",
		STControl:      "0001",
		SESegmentCount: 13,
	}
}

func goldenBodySegments() []x12.Segment {
	return []x12.Segment{
		{Tag: "BHT", Elements: []string{"0019", "00", "CLM-DEMO-001", "20260531", "1200", "CH"}},
		{Tag: "HL", Elements: []string{"1", "", "20", "1"}},
		{Tag: "NM1", Elements: []string{"85", "2", "Demo Home Care TX", "", "", "", "", "XX", "1234567890"}},
		{Tag: "HL", Elements: []string{"2", "1", "22", "0"}},
		{Tag: "NM1", Elements: []string{"IL", "1", "Patient", "Synthetic", "", "", "", "MI", "SYN-TX-00001"}},
		{Tag: "CLM", Elements: []string{"CLM-DEMO-001", "100.00", "", "", "11:B:1", "Y", "A", "Y", "Y"}},
		{Tag: "HI", Elements: []string{"ABK:Z9999"}},
		{Tag: "LX", Elements: []string{"1"}},
		{Tag: "SV1", Elements: []string{"HC:T1019", "100.00", "UN", "4", "", "", "1"}},
		{Tag: "REF", Elements: []string{"EVV", "20260501T090000"}},
	}
}

func collapseEDI(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\n' {
			out = append(out, s[i])
		}
	}
	for len(out) > 0 && (out[len(out)-1] == ' ' || out[len(out)-1] == '\t') {
		out = out[:len(out)-1]
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
