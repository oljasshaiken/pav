package rules

import (
	"context"
	"fmt"
	"time"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/edi"
	"github.com/pavillio/pav-edi/internal/repository"
	"github.com/pavillio/pav-edi/pkg/x12"
)

// RulesEngine implements Option 1 rules-based 837P generation.
type RulesEngine struct {
	Store *repository.Store
	Now   func() time.Time
}

func (e *RulesEngine) Transform(ctx context.Context, input domain.ClaimContext) (x12.Document, error) {
	cfg, err := e.Store.GetActivePayerConfig(ctx, input.Claim.State, input.Claim.PayerID, "837P")
	if err != nil {
		return x12.Document{}, err
	}
	return e.transformWithConfig(cfg.Config, input, cfg.ConfigVersion)
}

func (e *RulesEngine) transformWithConfig(cfg domain.PayerConfigBody, input domain.ClaimContext, version int32) (x12.Document, error) {
	now := time.Now().UTC()
	if e.Now != nil {
		now = e.Now()
	}
	raw, err := edi.Generate837P(cfg.Envelope, cfg.Mappings, input, now)
	if err != nil {
		return x12.Document{}, fmt.Errorf("generate 837P: %w", err)
	}
	return x12.Document{
		Raw:           raw,
		Engine:        "rules",
		ClaimID:       input.Claim.ID.String(),
		ConfigVersion: version,
		GeneratedAt:   now,
	}, nil
}

// StubEngine is an alias for RulesEngine.
type StubEngine = RulesEngine
