package submission

import (
	"context"

	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/repository"
	"github.com/pavillio/pav-edi/pkg/x12"
)

type Service interface {
	SubmitDryRun(ctx context.Context, claimID uuid.UUID, doc x12.Document) (int32, error)
}

type DryRunService struct {
	Store *repository.Store
}

func (s *DryRunService) SubmitDryRun(ctx context.Context, claimID uuid.UUID, doc x12.Document) (int32, error) {
	return s.Store.SaveGeneratedEDI(ctx, claimID, doc.Raw)
}

type NoopService struct{}

func (NoopService) SubmitDryRun(context.Context, uuid.UUID, x12.Document) (int32, error) {
	return 0, nil
}
