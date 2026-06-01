package platform_test

import (
	"context"
	"testing"
	"time"

	"github.com/pavillio/pav-edi/internal/platform"
)

func TestParseGeneratedAt(t *testing.T) {
	tm, ok := platform.ParseGeneratedAt("2026-05-31T12:00:00Z")
	if !ok {
		t.Fatal("expected ok")
	}
	if tm.UTC().Format(time.RFC3339) != "2026-05-31T12:00:00Z" {
		t.Fatalf("time = %v", tm)
	}
}

func TestResolveNow_prefersContext(t *testing.T) {
	fixed := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	ctx := platform.WithGeneratedAt(context.Background(), fixed)
	got := platform.ResolveNow(ctx, func() time.Time {
		return time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	})
	if !got.Equal(fixed) {
		t.Fatalf("got %v want %v", got, fixed)
	}
}
