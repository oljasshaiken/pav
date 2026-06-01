package platform

import (
	"context"
	"os"
	"time"
)

type generatedAtKey struct{}

// WithGeneratedAt pins envelope timestamps for deterministic EDI generation.
func WithGeneratedAt(ctx context.Context, t time.Time) context.Context {
	return context.WithValue(ctx, generatedAtKey{}, t.UTC())
}

// GeneratedAtFromContext returns a fixed generation time when set.
func GeneratedAtFromContext(ctx context.Context) (time.Time, bool) {
	t, ok := ctx.Value(generatedAtKey{}).(time.Time)
	return t, ok
}

// ResolveNow prefers a context-generated-at time, then fallback (typically time.Now).
func ResolveNow(ctx context.Context, fallback func() time.Time) time.Time {
	if t, ok := GeneratedAtFromContext(ctx); ok {
		return t
	}
	if fallback != nil {
		return fallback()
	}
	return time.Now().UTC()
}

// NowFromEnv returns GENERATED_AT (RFC3339) when set, otherwise time.Now().UTC().
func NowFromEnv() time.Time {
	if t, ok := ParseGeneratedAt(os.Getenv("GENERATED_AT")); ok {
		return t
	}
	return time.Now().UTC()
}

// ParseGeneratedAt parses an RFC3339 timestamp for compare / workflow-local.
func ParseGeneratedAt(raw string) (time.Time, bool) {
	if raw == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, false
	}
	return t.UTC(), true
}
