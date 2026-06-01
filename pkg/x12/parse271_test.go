package x12_test

import (
	"testing"

	"github.com/pavillio/pav-edi/pkg/x12"
)

func TestParseEligibility271_golden(t *testing.T) {
	raw := readFixture(t, "271_tx_golden.x12")
	got, err := x12.ParseEligibility271(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.InquiryRef != "CLM-DEMO-001" {
		t.Fatalf("inquiry_ref = %q", got.InquiryRef)
	}
	if got.MedicaidID != "SYN-TX-00001" {
		t.Fatalf("medicaid_id = %q", got.MedicaidID)
	}
	if got.PayerID != "TX-MCO-001" {
		t.Fatalf("payer_id = %q", got.PayerID)
	}
	if got.CoverageStatus != "ACTIVE" {
		t.Fatalf("coverage = %q", got.CoverageStatus)
	}
	if got.ServiceType != "42" {
		t.Fatalf("service_type = %q", got.ServiceType)
	}
}

func TestParseEligibility271_rejects277(t *testing.T) {
	raw := readFixture(t, "277_tx_golden.x12")
	_, err := x12.ParseEligibility271(raw)
	if err == nil {
		t.Fatal("expected error for 277 payload")
	}
}
