package pipeline

import (
	"context"
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
	claimCtx, err := g.Store.LoadClaimContext(ctx, claimID)
	if errors.Is(err, repository.ErrNotFound) || errors.Is(err, repository.ErrNoServiceLines) {
		return x12.Document{}, ErrClaimNotFound
	}
	if err != nil {
		return x12.Document{}, err
	}

	pc, err := g.Store.GetActivePayerConfig(ctx, claimCtx.Claim.State, claimCtx.Claim.PayerID, "837P")
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return x12.Document{}, err
	}

	rules := pc.Config.ValidationRules
	evvRules := pc.Config.EVVRules

	if err := g.Validate.PreValidate(ctx, claimCtx, rules); err != nil {
		return x12.Document{}, err
	}
	if err := validation.PreValidateEVV(ctx, claimCtx, evvRules); err != nil {
		return x12.Document{}, err
	}

	doc, err := g.Engine.Transform(ctx, claimCtx)
	if errors.Is(err, repository.ErrNotFound) {
		return x12.Document{}, ErrConfigNotFound
	}
	if err != nil {
		return x12.Document{}, err
	}

	if err := g.Validate.PostValidate(ctx, doc); err != nil {
		return x12.Document{}, err
	}
	if err := validation.PostValidateBusinessRules(ctx, doc, pc.Config.BusinessRules); err != nil {
		return x12.Document{}, err
	}
	return doc, nil
}
