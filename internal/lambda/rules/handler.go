package rules

import (
	"context"
	"errors"
	"fmt"

	"github.com/pavillio/pav-edi/internal/cel"
	"github.com/pavillio/pav-edi/internal/pipeline"
	"github.com/pavillio/pav-edi/internal/validation"
)

// Handler evaluates CEL rules for workflow pre/post transform phases.
type Handler struct{}

func (h *Handler) Handle(ctx context.Context, req pipeline.RulesEvaluateRequest) (pipeline.RulesEvaluateResult, error) {
	if req.ClaimID == "" {
		return pipeline.RulesEvaluateResult{}, fmt.Errorf("claim_id required")
	}

	var err error
	switch req.Phase {
	case pipeline.RulesPhasePreTransform:
		err = h.preTransform(ctx, req)
	case pipeline.RulesPhasePostTransform:
		err = h.postTransform(ctx, req)
	default:
		return pipeline.RulesEvaluateResult{}, fmt.Errorf("unknown phase %q", req.Phase)
	}

	if err != nil {
		if validation.IsValidationError(err) {
			return pipeline.RulesEvaluateResult{
				ClaimID: req.ClaimID,
				Valid:   false,
				Error:   &pipeline.WorkflowError{Code: "VALIDATION_FAILED", Message: err.Error()},
			}, nil
		}
		var ve *cel.ValidationError
		if errors.As(err, &ve) {
			return pipeline.RulesEvaluateResult{
				ClaimID: req.ClaimID,
				Valid:   false,
				Error:   &pipeline.WorkflowError{Code: "VALIDATION_FAILED", Message: ve.Message, RuleID: ve.RuleID},
			}, nil
		}
		return pipeline.RulesEvaluateResult{}, err
	}

	return pipeline.RulesEvaluateResult{ClaimID: req.ClaimID, Valid: true}, nil
}

func (h *Handler) preTransform(ctx context.Context, req pipeline.RulesEvaluateRequest) error {
	if err := validation.PreValidateClaim(ctx, req.ClaimContext, req.PayerConfig.ValidationRules); err != nil {
		return err
	}
	return validation.PreValidateEVV(ctx, req.ClaimContext, req.PayerConfig.EVVRules)
}

func (h *Handler) postTransform(ctx context.Context, req pipeline.RulesEvaluateRequest) error {
	if req.Document == nil {
		return fmt.Errorf("document required for post_transform")
	}
	if err := validation.PostValidateDocument(ctx, *req.Document); err != nil {
		return err
	}
	return validation.PostValidateBusinessRules(ctx, *req.Document, req.PayerConfig.BusinessRules)
}
