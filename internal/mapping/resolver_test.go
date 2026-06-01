package mapping_test

import (
	"testing"
	"time"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/mapping"
)

func TestResolvePath_patientLastName(t *testing.T) {
	ctx := syntheticClaimContext(t)
	got, err := mapping.ResolvePath(ctx, "patient.last_name")
	if err != nil {
		t.Fatalf("ResolvePath: %v", err)
	}
	if got != "Patient" {
		t.Fatalf("patient.last_name = %q, want %q", got, "Patient")
	}
}

func TestResolvePath_agencyName(t *testing.T) {
	ctx := syntheticClaimContext(t)
	got, err := mapping.ResolvePath(ctx, "agency.name")
	if err != nil {
		t.Fatalf("ResolvePath: %v", err)
	}
	if got != "Demo Home Care TX" {
		t.Fatalf("agency.name = %q, want %q", got, "Demo Home Care TX")
	}
}

func TestResolvePath_claimClaimNumber(t *testing.T) {
	ctx := syntheticClaimContext(t)
	got, err := mapping.ResolvePath(ctx, "claim.claim_number")
	if err != nil {
		t.Fatalf("ResolvePath: %v", err)
	}
	if got != "CLM-DEMO-001" {
		t.Fatalf("claim.claim_number = %q, want %q", got, "CLM-DEMO-001")
	}
}

func TestResolvePath_visitClockInTimeCompact(t *testing.T) {
	ctx := syntheticClaimContext(t)
	got, err := mapping.ResolvePath(ctx, "visit.clock_in_time")
	if err != nil {
		t.Fatalf("ResolvePath: %v", err)
	}
	if got != "20260501T090000" {
		t.Fatalf("visit.clock_in_time = %q, want %q", got, "20260501T090000")
	}
}

func TestResolvePath_serviceLineFields(t *testing.T) {
	ctx := syntheticClaimContext(t)

	proc, err := mapping.ResolvePath(ctx, "service_line.procedure_code")
	if err != nil {
		t.Fatalf("procedure_code: %v", err)
	}
	if proc != "T1019" {
		t.Fatalf("service_line.procedure_code = %q, want %q", proc, "T1019")
	}

	units, err := mapping.ResolvePath(ctx, "service_line.units")
	if err != nil {
		t.Fatalf("units: %v", err)
	}
	if units != "4" {
		t.Fatalf("service_line.units = %q, want %q", units, "4")
	}

	amount, err := mapping.ResolvePath(ctx, "service_line.amount")
	if err != nil {
		t.Fatalf("amount: %v", err)
	}
	if amount != "100.00" {
		t.Fatalf("service_line.amount = %q, want %q", amount, "100.00")
	}
}

func TestResolvePath_claimTotalAmountFromFirstServiceLine(t *testing.T) {
	ctx := syntheticClaimContext(t)
	got, err := mapping.ResolvePath(ctx, "claim.total_amount")
	if err != nil {
		t.Fatalf("ResolvePath: %v", err)
	}
	if got != "100.00" {
		t.Fatalf("claim.total_amount = %q, want %q", got, "100.00")
	}
}

func syntheticClaimContext(t *testing.T) domain.ClaimContext {
	t.Helper()

	claimNumber := "CLM-DEMO-001"
	amount := 100.00
	clockIn := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)

	return domain.ClaimContext{
		Claim: domain.Claim{
			ClaimNumber: &claimNumber,
		},
		Patient: domain.Patient{
			FirstName:  "Synthetic",
			LastName:   "Patient",
			MedicaidID: "SYN-TX-00001",
		},
		Agency: domain.Agency{
			Name: "Demo Home Care TX",
			NPI:  "1234567890",
		},
		Visits: []domain.Visit{
			{ClockInTime: &clockIn},
		},
		ServiceLines: []domain.ClaimServiceLine{
			{
				ProcedureCode:  "T1019",
				Units:          4,
				Amount:         &amount,
				DiagnosisCodes: []string{"Z9999"},
			},
		},
	}
}
