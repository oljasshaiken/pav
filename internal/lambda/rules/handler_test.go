package rules_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/lambda/rules"
	"github.com/pavillio/pav-edi/internal/pipeline"
	"github.com/pavillio/pav-edi/internal/validation"
)

func TestHandler_preTransform_matchesPipelineValidation(t *testing.T) {
	cfg := loadPayerConfig(t)
	ctx := goldenClaimContext()
	h := &rules.Handler{}

	pre, err := h.Handle(context.Background(), pipeline.RulesEvaluateRequest{
		ClaimID:       ctx.Claim.ID.String(),
		Phase:         pipeline.RulesPhasePreTransform,
		ClaimContext:  ctx,
		PayerConfig:   cfg,
		ConfigVersion: 1,
	})
	if err != nil || !pre.Valid {
		t.Fatalf("rules pre: valid=%v err=%v error=%+v", pre.Valid, err, pre.Error)
	}

	if err := validation.PreValidateClaim(context.Background(), ctx, cfg.ValidationRules); err != nil {
		t.Fatalf("pipeline pre: %v", err)
	}
	if err := validation.PreValidateEVV(context.Background(), ctx, cfg.EVVRules); err != nil {
		t.Fatalf("pipeline evv: %v", err)
	}
}

func TestHandler_preTransform_rejectsMissingDiagnosis(t *testing.T) {
	cfg := loadPayerConfig(t)
	ctx := goldenClaimContext()
	ctx.ServiceLines[0].DiagnosisCodes = nil
	h := &rules.Handler{}

	pre, err := h.Handle(context.Background(), pipeline.RulesEvaluateRequest{
		ClaimID:      ctx.Claim.ID.String(),
		Phase:        pipeline.RulesPhasePreTransform,
		ClaimContext: ctx,
		PayerConfig:  cfg,
	})
	if err != nil {
		t.Fatal(err)
	}
	if pre.Valid {
		t.Fatal("expected invalid")
	}
	if pre.Error == nil || pre.Error.Code != "VALIDATION_FAILED" {
		t.Fatalf("error = %+v", pre.Error)
	}
}

func loadPayerConfig(t *testing.T) domain.PayerConfigBody {
	t.Helper()
	data, err := os.ReadFile("../../../docs/fixtures/payer_config_837p_tx.json")
	if err != nil {
		t.Fatal(err)
	}
	var cfg domain.PayerConfigBody
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func goldenClaimContext() domain.ClaimContext {
	clockIn := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	return domain.ClaimContext{
		Claim: domain.Claim{
			ID:      uuid.MustParse("00000000-0000-4000-8000-000000000001"),
			PayerID: "TX-MCO-001",
			State:   "TX",
		},
		Authorization: domain.Authorization{ServiceType: "home_health"},
		Patient:       domain.Patient{MedicaidID: "SYN-TX-00001"},
		Agency:        domain.Agency{Name: "Demo Home Care TX"},
		Visits:        []domain.Visit{{EVVStatus: "VERIFIED", ClockInTime: &clockIn}},
		ServiceLines: []domain.ClaimServiceLine{{
			DiagnosisCodes: []string{"Z9999"},
		}},
	}
}
