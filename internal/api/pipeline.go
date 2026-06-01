package api

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/repository"
	"github.com/pavillio/pav-edi/pkg/x12"
)

// generateEDI runs LoadClaim → PreValidate → Transform → PostValidate.
func (s *Server) generateEDI(ctx context.Context, claimID uuid.UUID) (x12.Document, error) {
	claimCtx, err := s.Store.LoadClaimContext(ctx, claimID)
	if errors.Is(err, repository.ErrNotFound) || errors.Is(err, repository.ErrNoServiceLines) {
		return x12.Document{}, errClaimNotFound
	}
	if err != nil {
		return x12.Document{}, err
	}

	rules, err := s.loadValidationRules(ctx, claimCtx.Claim)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return x12.Document{}, err
	}
	if err := s.Validate.PreValidate(ctx, claimCtx, rules); err != nil {
		return x12.Document{}, err
	}

	doc, err := s.Engine.Transform(ctx, claimCtx)
	if errors.Is(err, repository.ErrNotFound) {
		return x12.Document{}, errConfigNotFound
	}
	if err != nil {
		return x12.Document{}, err
	}

	if err := s.Validate.PostValidate(ctx, doc); err != nil {
		return x12.Document{}, err
	}
	return doc, nil
}

func (s *Server) loadValidationRules(ctx context.Context, claim domain.Claim) (json.RawMessage, error) {
	pc, err := s.Store.GetActivePayerConfig(ctx, claim.State, claim.PayerID, "837P")
	if err != nil {
		return nil, err
	}
	return pc.Config.ValidationRules, nil
}

var (
	errClaimNotFound  = errors.New("claim not found")
	errConfigNotFound = errors.New("config not found")
)
