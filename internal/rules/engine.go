package rules

import (
	"context"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/repository"
	"github.com/pavillio/pav-edi/pkg/x12"
)

type Engine interface {
	Transform(ctx context.Context, input domain.ClaimContext) (x12.Document, error)
}

type StubEngine struct {
	Store *repository.Store
}

func (e *StubEngine) Transform(ctx context.Context, input domain.ClaimContext) (x12.Document, error) {
	cfg, err := e.Store.GetActivePayerConfig(ctx, input.Claim.State, input.Claim.PayerID, "837P")
	if err != nil {
		return x12.Document{}, err
	}
	return x12.NewPlaceholder("rules", input.Claim.ID.String(), cfg.ConfigVersion), nil
}
