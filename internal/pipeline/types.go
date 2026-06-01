package pipeline

import (
	"context"

	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/domain"
)

// ClaimStore loads claim context and payer configuration for EDI generation.
type ClaimStore interface {
	LoadClaimContext(ctx context.Context, claimID uuid.UUID) (domain.ClaimContext, error)
	GetActivePayerConfig(ctx context.Context, state, payerID, transactionType string) (domain.PayerConfig, error)
}
