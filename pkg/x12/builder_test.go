package x12_test

import (
	"os"
	"strings"
	"testing"

	"github.com/pavillio/pav-edi/pkg/x12"
)

func TestBuilder_goldenEnvelopeAndNM1(t *testing.T) {
	cfg := loadEnvelopeConfig(t)
	opts := goldenClockOptions()
	b := x12.NewBuilder(cfg, opts, x12.Separators{})

	b.AppendBody(
		x12.Segment{Tag: "NM1", Elements: []string{"85", "2", "Demo Home Care TX", "", "", "", "", "XX", "1234567890"}},
	)

	got := collapseEDI(b.BuildEnvelopePlusNM1())
	want := collapseEDI(`ISA*00*          *00*          *ZZ*PAVILLIO       *ZZ*TX_MCO         *260531*1200*^*00501*000000001*0*T*:~
GS*HC*PAVILLIO*TX_MCO*20260531*1200*1*X*005010X222A1~
ST*837*0001*005010X222A1~
NM1*85*2*Demo Home Care TX*****XX*1234567890~`)

	if got != want {
		t.Fatalf("BuildEnvelopePlusNM1():\n%s\nwant:\n%s", got, want)
	}
}

func TestBuilder_fullGoldenInterchange(t *testing.T) {
	goldenBytes, err := os.ReadFile("../../docs/fixtures/837p_tx_golden.x12")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	want := collapseEDI(string(goldenBytes))

	cfg := loadEnvelopeConfig(t)
	opts := goldenClockOptions()
	b := x12.NewBuilder(cfg, opts, x12.Separators{})
	b.AppendBody(goldenBodySegments()...)
	got := collapseEDI(b.Build())
	if got != want {
		t.Fatalf("builder output length %d, golden length %d\nfirst diff near output:\n%s",
			len(got), len(want), diffPrefix(got, want))
	}
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
		SESegmentCount: 13, // matches docs/fixtures/837p_tx_golden.x12
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
			return "got[" + got[start:end] + "] vs want[" + want[start:min(end, len(want))] + "]"
		}
	}
	if len(got) != len(want) {
		return "length mismatch"
	}
	return "identical"
}

func collapseEDI(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), "\n", "")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
