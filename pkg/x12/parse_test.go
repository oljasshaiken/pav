package x12_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pavillio/pav-edi/pkg/x12"
)

func TestParseAcknowledgment_277Golden(t *testing.T) {
	raw := readFixture(t, "277_tx_golden.x12")
	ack, err := x12.ParseAcknowledgment(raw)
	if err != nil {
		t.Fatal(err)
	}
	if ack.TransactionSet != "277" {
		t.Fatalf("tx = %q", ack.TransactionSet)
	}
	if ack.ClaimRef != "CLM-DEMO-001" {
		t.Fatalf("claim ref = %q", ack.ClaimRef)
	}
}

func TestParseAcknowledgment_999RequiresClaimID(t *testing.T) {
	raw := readFixture(t, "999_tx_golden.x12")
	ack, err := x12.ParseAcknowledgment(raw)
	if err != nil {
		t.Fatal(err)
	}
	if ack.TransactionSet != "999" {
		t.Fatalf("tx = %q", ack.TransactionSet)
	}
	if ack.ClaimRef != "" {
		t.Fatalf("expected empty claim ref for 999, got %q", ack.ClaimRef)
	}
}

func TestParseAcknowledgment_rejects837(t *testing.T) {
	_, err := x12.ParseAcknowledgment("ST*837*0001~SE*1*0001~")
	if err == nil {
		t.Fatal("expected error")
	}
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "..", "docs", "fixtures", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
