package config_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/pavillio/pav-edi/internal/config"
	"github.com/pavillio/pav-edi/internal/domain"
)

func TestRedisCache_roundTrip(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache := config.NewRedisCache(client)

	cfg := domain.PayerConfig{
		State:         "TX",
		PayerID:       "TX-MCO-001",
		ConfigVersion: 1,
		Config: domain.PayerConfigBody{
			X12Version: "005010X222A1",
			Mappings:   json.RawMessage(`{"patient":{}}`),
		},
	}
	key := config.LookupKey("TX", "TX-MCO-001", "837P")
	cache.Set(key, cfg, 0)

	got, ok := cache.Get(key)
	if !ok {
		t.Fatal("cache miss")
	}
	if got.Config.X12Version != cfg.Config.X12Version {
		t.Fatalf("x12_version = %q", got.Config.X12Version)
	}
}

func TestCachedLoader_withRedisCache(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	loader := &stubLoader{
		cfg: domain.PayerConfig{
			State: "TX", PayerID: "TX-MCO-001",
			Config: domain.PayerConfigBody{Mappings: json.RawMessage(`{}`)},
		},
	}
	cached := config.NewCachedLoader(loader, config.NewRedisCache(client))

	if _, err := cached.Load(context.Background(), "TX", "TX-MCO-001", "837P"); err != nil {
		t.Fatal(err)
	}
	if _, err := cached.Load(context.Background(), "TX", "TX-MCO-001", "837P"); err != nil {
		t.Fatal(err)
	}
	if loader.calls != 1 {
		t.Fatalf("loader calls = %d, want 1", loader.calls)
	}
}
