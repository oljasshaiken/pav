package config_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pavillio/pav-edi/internal/config"
	"github.com/pavillio/pav-edi/internal/domain"
)

type stubLoader struct {
	cfg   domain.PayerConfig
	err   error
	calls int
}

func (s *stubLoader) Load(_ context.Context, _, _, _ string) (domain.PayerConfig, error) {
	s.calls++
	return s.cfg, s.err
}

func TestCachedLoader_returnsCachedConfig(t *testing.T) {
	loader := &stubLoader{
		cfg: domain.PayerConfig{
			State: "TX", PayerID: "TX-MCO-001",
			Config: domain.PayerConfigBody{
				X12Version: "005010X222A1",
				Mappings:   json.RawMessage(`{}`),
			},
		},
	}
	cached := config.NewCachedLoader(loader, config.NewMemoryCache())

	cfg1, err := cached.Load(context.Background(), "TX", "TX-MCO-001", "837P")
	if err != nil {
		t.Fatal(err)
	}
	cfg2, err := cached.Load(context.Background(), "TX", "TX-MCO-001", "837P")
	if err != nil {
		t.Fatal(err)
	}
	if loader.calls != 1 {
		t.Fatalf("loader calls = %d, want 1", loader.calls)
	}
	if cfg1.Config.X12Version != cfg2.Config.X12Version {
		t.Fatal("cached config mismatch")
	}
}

func TestCachedLoader_invalidateRefreshesConfig(t *testing.T) {
	loader := &stubLoader{
		cfg: domain.PayerConfig{ConfigVersion: 1, Config: domain.PayerConfigBody{Mappings: json.RawMessage(`{}`)}},
	}
	cached := config.NewCachedLoader(loader, config.NewMemoryCache())
	if _, err := cached.Load(context.Background(), "TX", "TX-MCO-001", "837P"); err != nil {
		t.Fatal(err)
	}
	cached.Invalidate("TX", "TX-MCO-001", "837P")
	loader.cfg.ConfigVersion = 2
	if _, err := cached.Load(context.Background(), "TX", "TX-MCO-001", "837P"); err != nil {
		t.Fatal(err)
	}
	if loader.calls != 2 {
		t.Fatalf("loader calls = %d, want 2 after invalidate", loader.calls)
	}
}
