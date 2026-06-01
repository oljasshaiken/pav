package config

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pavillio/pav-edi/internal/domain"
)

// Loader fetches active payer configuration from a backing store.
type Loader interface {
	Load(ctx context.Context, state, payerID, transactionType string) (domain.PayerConfig, error)
}

// Cache stores payer configs by lookup key.
type Cache interface {
	Get(key string) (domain.PayerConfig, bool)
	Set(key string, cfg domain.PayerConfig, ttl time.Duration)
	Invalidate(key string)
}

// CacheKey builds a stable cache key for payer config lookups.
func CacheKey(state, payerID, transactionType string, version int32) string {
	return fmt.Sprintf("%s:%s:%s:%d", state, payerID, transactionType, version)
}

// LookupKey is the key used before version is known (active config resolution).
func LookupKey(state, payerID, transactionType string) string {
	return fmt.Sprintf("%s:%s:%s:active", state, payerID, transactionType)
}

// CachedLoader wraps a Loader with a Cache.
type CachedLoader struct {
	inner Loader
	cache Cache
	ttl   time.Duration
}

func NewCachedLoader(inner Loader, cache Cache) *CachedLoader {
	return &CachedLoader{inner: inner, cache: cache, ttl: 15 * time.Minute}
}

func (c *CachedLoader) Invalidate(state, payerID, transactionType string) {
	c.cache.Invalidate(LookupKey(state, payerID, transactionType))
}

func (c *CachedLoader) Load(ctx context.Context, state, payerID, transactionType string) (domain.PayerConfig, error) {
	lookup := LookupKey(state, payerID, transactionType)
	if cfg, ok := c.cache.Get(lookup); ok {
		return cfg, nil
	}
	cfg, err := c.inner.Load(ctx, state, payerID, transactionType)
	if err != nil {
		return domain.PayerConfig{}, err
	}
	c.cache.Set(lookup, cfg, c.ttl)
	versionKey := CacheKey(state, payerID, transactionType, cfg.ConfigVersion)
	c.cache.Set(versionKey, cfg, c.ttl)
	return cfg, nil
}

// MemoryCache is an in-process config cache for local dev and tests.
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]cacheEntry
}

type cacheEntry struct {
	cfg       domain.PayerConfig
	expiresAt time.Time
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{items: make(map[string]cacheEntry)}
}

func (m *MemoryCache) Get(key string) (domain.PayerConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.items[key]
	if !ok {
		return domain.PayerConfig{}, false
	}
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		return domain.PayerConfig{}, false
	}
	return entry.cfg, true
}

func (m *MemoryCache) Set(key string, cfg domain.PayerConfig, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry := cacheEntry{cfg: cfg}
	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}
	m.items[key] = entry
}

func (m *MemoryCache) Invalidate(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, key)
}
