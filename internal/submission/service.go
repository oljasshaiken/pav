package submission

import (
	"context"

	"github.com/pavillio/pav-edi/pkg/x12"
)

type Service interface {
	Submit(ctx context.Context, doc x12.Document) error
}

type NoopService struct{}

func (NoopService) Submit(context.Context, x12.Document) error { return nil }
