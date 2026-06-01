package cel_test

import (
	"testing"

	"github.com/pavillio/pav-edi/internal/cel"
	"github.com/pavillio/pav-edi/internal/domain"
)

func TestEvaluateRules_diagnosisRequiredFailsWhenMissing(t *testing.T) {
	rules := []domain.CELRule{{
		ID:      "diagnosis_required",
		CEL:     `authorization.service_type != "home_health" || size(service_line.diagnosis_codes) > 0`,
		Message: "diagnosis_code required for home health claims",
	}}
	bindings := cel.ClaimBindings(domain.ClaimContext{
		Authorization: domain.Authorization{ServiceType: "home_health"},
		ServiceLines:  []domain.ClaimServiceLine{{DiagnosisCodes: nil}},
	})

	err := cel.EvaluateAll(rules, bindings)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestEvaluateRules_diagnosisRequiredPassesWithCodes(t *testing.T) {
	rules := []domain.CELRule{{
		ID:      "diagnosis_required",
		CEL:     `authorization.service_type != "home_health" || size(service_line.diagnosis_codes) > 0`,
		Message: "diagnosis_code required",
	}}
	bindings := cel.ClaimBindings(domain.ClaimContext{
		Authorization: domain.Authorization{ServiceType: "home_health"},
		ServiceLines:  []domain.ClaimServiceLine{{DiagnosisCodes: []string{"Z9999"}}},
	})

	if err := cel.EvaluateAll(rules, bindings); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEvaluateRules_evvVerifiedFails(t *testing.T) {
	rules := []domain.CELRule{{
		ID:      "evv_verified",
		CEL:     `visit.evv_status == "VERIFIED"`,
		Message: "EVV must be verified",
	}}
	bindings := cel.ClaimBindings(domain.ClaimContext{
		Visits: []domain.Visit{{EVVStatus: "PENDING"}},
	})

	if err := cel.EvaluateAll(rules, bindings); err == nil {
		t.Fatal("expected validation error")
	}
}
