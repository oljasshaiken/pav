package validation

import (
	"context"

	"github.com/pavillio/pav-edi/pkg/x12"
)

type Pipeline interface {
	Validate(ctx context.Context, doc x12.Document) error
}

type NoopPipeline struct{}

func (NoopPipeline) Validate(context.Context, x12.Document) error { return nil }
