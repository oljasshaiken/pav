package pipeline_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/pipeline"
	"github.com/pavillio/pav-edi/internal/repository"
	"github.com/pavillio/pav-edi/internal/validation"
	"github.com/pavillio/pav-edi/pkg/x12"
)

type fakeStore struct {
	claim     domain.ClaimContext
	claimErr  error
	config    domain.PayerConfig
	configErr error
}

func (f *fakeStore) LoadClaimContext(context.Context, uuid.UUID) (domain.ClaimContext, error) {
	return f.claim, f.claimErr
}

func (f *fakeStore) GetActivePayerConfig(context.Context, string, string, string) (domain.PayerConfig, error) {
	return f.config, f.configErr
}

type fakeEngine struct {
	doc x12.Document
	err error
}

func (f *fakeEngine) Transform(context.Context, domain.ClaimContext) (x12.Document, error) {
	return f.doc, f.err
}

func TestGenerate_success(t *testing.T) {
	claimID := uuid.New()
	rules := json.RawMessage(`[{"id":"r1","cel":"true","message":"fail"}]`)
	gen := pipeline.Generator{
		Store: &fakeStore{
			claim: domain.ClaimContext{Claim: domain.Claim{ID: claimID}},
			config: domain.PayerConfig{
				Config: domain.PayerConfigBody{ValidationRules: rules},
			},
		},
		Engine: &fakeEngine{
			doc: x12.Document{Raw: "ISA*TEST~", ClaimID: claimID.String(), GeneratedAt: time.Now().UTC()},
		},
		Validate: validation.NoopPipeline{},
	}

	doc, err := gen.Generate(context.Background(), claimID)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Raw != "ISA*TEST~" {
		t.Fatalf("edi = %q", doc.Raw)
	}
}

func TestGenerate_claimNotFound(t *testing.T) {
	gen := pipeline.Generator{
		Store: &fakeStore{claimErr: repository.ErrNotFound},
	}
	_, err := gen.Generate(context.Background(), uuid.New())
	if !errors.Is(err, pipeline.ErrClaimNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestGenerate_configNotFound(t *testing.T) {
	gen := pipeline.Generator{
		Store: &fakeStore{
			claim: domain.ClaimContext{},
		},
		Engine:   &fakeEngine{err: repository.ErrNotFound},
		Validate: validation.NoopPipeline{},
	}
	_, err := gen.Generate(context.Background(), uuid.New())
	if !errors.Is(err, pipeline.ErrConfigNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestGenerate_validationFailed(t *testing.T) {
	rules := json.RawMessage(`[{
		"id":"diagnosis_required",
		"cel":"size(service_line.diagnosis_codes) > 0",
		"message":"diagnosis required"
	}]`)
	gen := pipeline.Generator{
		Store: &fakeStore{
			claim: domain.ClaimContext{
				Authorization: domain.Authorization{ServiceType: "home_health"},
				ServiceLines:  []domain.ClaimServiceLine{{DiagnosisCodes: nil}},
			},
			config: domain.PayerConfig{
				Config: domain.PayerConfigBody{ValidationRules: rules},
			},
		},
		Engine:   &fakeEngine{},
		Validate: validation.ConfigPipeline{},
	}
	_, err := gen.Generate(context.Background(), uuid.New())
	if !validation.IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
