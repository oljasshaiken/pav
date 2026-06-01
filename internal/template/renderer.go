package template

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/edi"
	"github.com/pavillio/pav-edi/internal/repository"
	"github.com/pavillio/pav-edi/pkg/x12"
)

type Renderer struct {
	Store *repository.Store
	Now   func() time.Time
}

func (r *Renderer) Transform(ctx context.Context, input domain.ClaimContext) (x12.Document, error) {
	o, err := r.Store.GetActiveTemplateOverride(ctx, input.Claim.State, input.Claim.PayerID, "837P")
	if err != nil {
		return x12.Document{}, err
	}
	pc, err := r.Store.GetActivePayerConfig(ctx, input.Claim.State, input.Claim.PayerID, "837P")
	if err != nil {
		return x12.Document{}, err
	}

	mappings, err := extractMappings(o.Mapper)
	if err != nil {
		return x12.Document{}, err
	}

	now := time.Now().UTC()
	if r.Now != nil {
		now = r.Now()
	}
	raw, err := edi.Generate837P(pc.Config.Envelope, mappings, input, now)
	if err != nil {
		return x12.Document{}, fmt.Errorf("generate 837P: %w", err)
	}
	return x12.Document{
		Raw:           raw,
		Engine:        "template",
		ClaimID:       input.Claim.ID.String(),
		ConfigVersion: o.OverrideVersion,
		GeneratedAt:   now,
	}, nil
}

// StubRenderer is an alias for Renderer.
type StubRenderer = Renderer

func extractMappings(mapper json.RawMessage) (json.RawMessage, error) {
	var wrapper struct {
		Mappings json.RawMessage `json:"mappings"`
	}
	if err := json.Unmarshal(mapper, &wrapper); err != nil {
		return nil, fmt.Errorf("parse mapper: %w", err)
	}
	if len(wrapper.Mappings) == 0 {
		return mapper, nil
	}
	return wrapper.Mappings, nil
}
