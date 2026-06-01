package validation

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/pkg/x12"
)

type Pipeline interface {
	PreValidate(ctx context.Context, claim domain.ClaimContext, rulesJSON json.RawMessage) error
	PostValidate(ctx context.Context, doc x12.Document) error
}

type ConfigPipeline struct{}

func (ConfigPipeline) PreValidate(ctx context.Context, claim domain.ClaimContext, rulesJSON json.RawMessage) error {
	return PreValidateClaim(ctx, claim, rulesJSON)
}

func (ConfigPipeline) PostValidate(ctx context.Context, doc x12.Document) error {
	return PostValidateDocument(ctx, doc)
}

type NoopPipeline struct{}

func (NoopPipeline) PreValidate(context.Context, domain.ClaimContext, json.RawMessage) error {
	return nil
}

func (NoopPipeline) PostValidate(context.Context, x12.Document) error {
	return nil
}

func IsValidationError(err error) bool {
	return errors.Is(err, ErrValidationFailed)
}
