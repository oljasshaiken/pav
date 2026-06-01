package transformer

import (
	"context"
	"fmt"
	"time"

	"github.com/pavillio/pav-edi/internal/pipeline"
	"github.com/pavillio/pav-edi/internal/rules"
)

// Handler implements the transformer Lambda business logic.
type Handler struct {
	Now func() time.Time
}

func (h *Handler) NowTime() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now().UTC()
}

// Handle generates 837P X12 from a TransformRequest.
func (h *Handler) Handle(_ context.Context, req pipeline.TransformRequest) (pipeline.TransformResult, error) {
	if req.ClaimID == "" {
		return pipeline.TransformResult{}, fmt.Errorf("claim_id required")
	}
	now := h.NowTime()
	if !req.GeneratedAt.IsZero() {
		now = req.GeneratedAt
	}
	doc, err := rules.Transform837P(req.PayerConfig, req.ClaimContext, req.ConfigVersion, now)
	if err != nil {
		return pipeline.TransformResult{}, err
	}
	return pipeline.TransformResult{
		ClaimID:       req.ClaimID,
		ConfigVersion: req.ConfigVersion,
		Document:      doc,
	}, nil
}
