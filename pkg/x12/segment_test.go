package x12_test

import (
	"testing"

	"github.com/pavillio/pav-edi/pkg/x12"
)

func TestSegmentSerialize_defaultSeparators(t *testing.T) {
	seg := x12.Segment{
		Tag:      "NM1",
		Elements: []string{"85", "2", "Demo Home Care TX", "", "", "", "XX", "1234567890"},
	}
	got := seg.Serialize(x12.Separators{})
	want := "NM1*85*2*Demo Home Care TX****XX*1234567890~"
	if got != want {
		t.Fatalf("Serialize() = %q, want %q", got, want)
	}
}

func TestSegmentSerialize_customSeparators(t *testing.T) {
	sep := x12.Separators{Element: "|", Segment: "\n"}
	seg := x12.Segment{Tag: "ST", Elements: []string{"837", "0001", "005010X222A1"}}
	got := seg.Serialize(sep)
	want := "ST|837|0001|005010X222A1\n"
	if got != want {
		t.Fatalf("Serialize() = %q, want %q", got, want)
	}
}

func TestSegmentSerialize_emptyElementsPreserved(t *testing.T) {
	seg := x12.Segment{Tag: "HL", Elements: []string{"1", "", "20", "1"}}
	got := seg.Serialize(x12.Separators{})
	want := "HL*1**20*1~"
	if got != want {
		t.Fatalf("Serialize() = %q, want %q", got, want)
	}
}
