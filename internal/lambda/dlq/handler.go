package dlq

import (
	"context"

	"github.com/pavillio/pav-edi/internal/pipeline"
	"github.com/pavillio/pav-edi/internal/platform/observability"
	"github.com/pavillio/pav-edi/internal/queue"
)

// Handler publishes workflow failures to DLQ.
type Handler struct {
	Publisher queue.Publisher
}

func (h *Handler) Handle(ctx context.Context, req pipeline.DLQPublishRequest) error {
	if h.Publisher == nil {
		return nil
	}
	if req.Error == nil {
		req.Error = &pipeline.WorkflowError{Code: "WORKFLOW_ERROR", Message: "unknown failure"}
	}
	msg := queue.DLQMessage{
		ClaimID: req.ClaimID,
		PayerID: req.PayerID,
		State:   req.State,
		Phase:   string(req.Phase),
		Code:    req.Error.Code,
		Message: req.Error.Message,
		RuleID:  req.Error.RuleID,
	}
	observability.LogDLQAlert(ctx, msg)
	return h.Publisher.Publish(ctx, msg)
}
