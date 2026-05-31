package template

import (
	"context"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/repository"
	"github.com/pavillio/pav-edi/pkg/x12"
)

type StubRenderer struct {
	Store *repository.Store
}

func (r *StubRenderer) Transform(ctx context.Context, input domain.ClaimContext) (x12.Document, error) {
	o, err := r.Store.GetActiveTemplateOverride(ctx, input.Claim.State, input.Claim.PayerID, "837P")
	if err != nil {
		return x12.Document{}, err
	}
	return x12.NewPlaceholder("template", input.Claim.ID.String(), o.OverrideVersion), nil
}
