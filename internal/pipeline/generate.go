package pipeline

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/repository"
	"github.com/pavillio/pav-edi/internal/validation"
	"github.com/pavillio/pav-edi/pkg/x12"
)

// Transformer generates X12 from claim context.
type Transformer interface {
	Transform(ctx context.Context, input domain.ClaimContext) (x12.Document, error)
}

// Generator runs LoadClaim → PreValidate → PreValidateEVV → Transform → PostValidate.
type Generator struct {
	Store    ClaimStore
	Engine   Transformer
	Validate validation.Pipeline
}

// Generate produces validated X12 for a claim.
func (g *Generator) Generate(ctx context.Context, claimID uuid.UUID) (x12.Document, error) {
	return g.GenerateWithTrace(ctx, claimID, noopRecorder{})
}

func (g *Generator) load(ctx context.Context, claimID uuid.UUID) (domain.ClaimContext, domain.PayerConfig, error) {
	claimCtx, err := g.Store.LoadClaimContext(ctx, claimID)
	if errors.Is(err, repository.ErrNotFound) || errors.Is(err, repository.ErrNoServiceLines) {
		return domain.ClaimContext{}, domain.PayerConfig{}, ErrClaimNotFound
	}
	if err != nil {
		return domain.ClaimContext{}, domain.PayerConfig{}, err
	}

	pc, err := g.Store.GetActivePayerConfig(ctx, claimCtx.Claim.State, claimCtx.Claim.PayerID, "837P")
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return domain.ClaimContext{}, domain.PayerConfig{}, err
	}
	return claimCtx, pc, nil
}

func (g *Generator) preValidate(ctx context.Context, claimCtx domain.ClaimContext, rules, evvRules json.RawMessage) error {
	if err := g.Validate.PreValidate(ctx, claimCtx, rules); err != nil {
		return err
	}
	return validation.PreValidateEVV(ctx, claimCtx, evvRules)
}

func (g *Generator) transform(ctx context.Context, claimCtx domain.ClaimContext) (x12.Document, error) {
	doc, err := g.Engine.Transform(ctx, claimCtx)
	if errors.Is(err, repository.ErrNotFound) {
		return x12.Document{}, ErrConfigNotFound
	}
	return doc, err
}

func (g *Generator) postValidate(ctx context.Context, doc x12.Document, businessRules json.RawMessage) error {
	if err := g.Validate.PostValidate(ctx, doc); err != nil {
		return err
	}
	return validation.PostValidateBusinessRules(ctx, doc, businessRules)
}
