package validation_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/validation"
	"github.com/pavillio/pav-edi/pkg/x12"
)

func TestPreValidateClaim_CELDiagnosisRequired(t *testing.T) {
	rules := json.RawMessage(`[{
		"id":"diagnosis_required",
		"cel":"authorization.service_type != \"home_health\" || size(service_line.diagnosis_codes) > 0",
		"message":"diagnosis required"
	}]`)
	claim := domain.ClaimContext{
		Authorization: domain.Authorization{ServiceType: "home_health"},
		ServiceLines:  []domain.ClaimServiceLine{{DiagnosisCodes: nil}},
	}
	err := validation.PreValidateClaim(context.Background(), claim, rules)
	if !validation.IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestPreValidateClaim_requiresDiagnosisForHomeHealth(t *testing.T) {
	rules := json.RawMessage(`[{"field":"diagnosis_code","rule":"required","condition":"service_type = 'home_health'"}]`)
	claim := domain.ClaimContext{
		Authorization: domain.Authorization{ServiceType: "home_health"},
		ServiceLines:  []domain.ClaimServiceLine{{DiagnosisCodes: nil}},
	}
	err := validation.PreValidateClaim(context.Background(), claim, rules)
	if !validation.IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestPreValidateClaim_passesWithDiagnosis(t *testing.T) {
	rules := json.RawMessage(`[{"field":"diagnosis_code","rule":"required","condition":"service_type = 'home_health'"}]`)
	claim := domain.ClaimContext{
		Authorization: domain.Authorization{ServiceType: "home_health"},
		ServiceLines:  []domain.ClaimServiceLine{{DiagnosisCodes: []string{"Z9999"}}},
	}
	if err := validation.PreValidateClaim(context.Background(), claim, rules); err != nil {
		t.Fatal(err)
	}
}

func TestPostValidateDocument_rejectsEmptyEDI(t *testing.T) {
	err := validation.PostValidateDocument(context.Background(), x12.Document{})
	if !validation.IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
