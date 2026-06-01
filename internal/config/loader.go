package config

import (
	"context"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/repository"
)

// StoreLoader loads payer configs from Postgres.
type StoreLoader struct {
	Store *repository.Store
}

func (l *StoreLoader) Load(ctx context.Context, state, payerID, transactionType string) (domain.PayerConfig, error) {
	return l.Store.GetActivePayerConfig(ctx, state, payerID, transactionType)
}
