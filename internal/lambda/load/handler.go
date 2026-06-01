package load

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/pipeline"
	"github.com/pavillio/pav-edi/internal/repository"
)

// Handler loads claim context and active payer config for workflow steps.
type Handler struct {
	Store *repository.Store
}

func (h *Handler) Handle(ctx context.Context, req pipeline.LoadClaimRequest) (pipeline.LoadClaimResult, error) {
	if req.ClaimID == "" {
		return pipeline.LoadClaimResult{}, fmt.Errorf("claim_id required")
	}
	claimID, err := uuid.Parse(req.ClaimID)
	if err != nil {
		return pipeline.LoadClaimResult{}, fmt.Errorf("invalid claim_id: %w", err)
	}
	if h.Store == nil {
		return pipeline.LoadClaimResult{}, fmt.Errorf("store required")
	}

	claimCtx, err := h.Store.LoadClaimContext(ctx, claimID)
	if errors.Is(err, repository.ErrNotFound) || errors.Is(err, repository.ErrNoServiceLines) {
		return pipeline.LoadClaimResult{}, pipeline.ErrClaimNotFound
	}
	if err != nil {
		return pipeline.LoadClaimResult{}, err
	}

	pc, err := h.Store.GetActivePayerConfig(ctx, claimCtx.Claim.State, claimCtx.Claim.PayerID, "837P")
	if errors.Is(err, repository.ErrNotFound) {
		return pipeline.LoadClaimResult{}, pipeline.ErrConfigNotFound
	}
	if err != nil {
		return pipeline.LoadClaimResult{}, err
	}

	return pipeline.LoadClaimResult{
		ClaimID:       req.ClaimID,
		ClaimContext:  claimCtx,
		PayerConfig:   pc.Config,
		ConfigVersion: pc.ConfigVersion,
	}, nil
}
