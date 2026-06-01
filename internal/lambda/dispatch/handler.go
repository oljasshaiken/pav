package dispatch

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pavillio/pav-edi/internal/queue"
)

// WorkflowRunner executes outbound claim generation.
type WorkflowRunner interface {
	Run(ctx context.Context, claimID string) error
}

// Handler consumes SQS FIFO messages and runs the outbound workflow.
type Handler struct {
	Workflow WorkflowRunner
}

type sqsRecord struct {
	Body string `json:"body"`
}

type sqsEvent struct {
	Records []sqsRecord `json:"Records"`
}

// HandleEvent processes an SQS batch (Lambda event shape).
func (h *Handler) HandleEvent(ctx context.Context, raw json.RawMessage) error {
	var event sqsEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		return fmt.Errorf("parse sqs event: %w", err)
	}
	// Single claim queue message (direct invoke / tests).
	if len(event.Records) == 0 {
		var msg queue.ClaimQueueMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			return fmt.Errorf("parse sqs event: %w", err)
		}
		return h.runOne(ctx, msg)
	}
	for _, rec := range event.Records {
		var msg queue.ClaimQueueMessage
		if err := json.Unmarshal([]byte(rec.Body), &msg); err != nil {
			return err
		}
		if err := h.runOne(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) runOne(ctx context.Context, msg queue.ClaimQueueMessage) error {
	if msg.ClaimID == "" {
		return fmt.Errorf("claim_id required")
	}
	if h.Workflow == nil {
		return fmt.Errorf("workflow required")
	}
	return h.Workflow.Run(ctx, msg.ClaimID)
}

// WorkflowAdapter adapts a function to WorkflowRunner.
type WorkflowAdapter struct {
	RunFn func(ctx context.Context, claimID string) error
}

func (a WorkflowAdapter) Run(ctx context.Context, claimID string) error {
	return a.RunFn(ctx, claimID)
}
