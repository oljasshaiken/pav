package x12_test

import (
	"testing"

	"github.com/pavillio/pav-edi/pkg/x12"
)

func TestNewPlaceholder(t *testing.T) {
	doc := x12.NewPlaceholder("rules", "00000000-0000-4000-8000-000000000001", 1)
	if doc.Raw != "rules:00000000-0000-4000-8000-000000000001:1" {
		t.Fatalf("raw = %q", doc.Raw)
	}
	if doc.Engine != "rules" {
		t.Fatalf("engine = %q", doc.Engine)
	}
}
