package api_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/repository"
	"github.com/pavillio/pav-edi/internal/rules"
	"github.com/pavillio/pav-edi/internal/template"
	"github.com/pavillio/pav-edi/internal/testutil"
)

func TestEnginesProduceDistinctPlaceholderEDI(t *testing.T) {
	pool := testutil.StartPostgres(t)
	store := repository.New(pool)
	ctx := context.Background()

	claimID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	testutil.InsertFixtureClaim(t, pool, claimID)
	testutil.InsertPayerConfig(t, pool)
	testutil.InsertTemplateOverride(t, pool)

	cc, err := store.LoadClaimContext(ctx, claimID)
	if err != nil {
		t.Fatal(err)
	}

	rulesDoc, err := (&rules.StubEngine{Store: store}).Transform(ctx, cc)
	if err != nil {
		t.Fatal(err)
	}
	templateDoc, err := (&template.StubRenderer{Store: store}).Transform(ctx, cc)
	if err != nil {
		t.Fatal(err)
	}

	if rulesDoc.Raw == templateDoc.Raw {
		t.Fatalf("engines produced identical edi: %q", rulesDoc.Raw)
	}
	if !strings.HasPrefix(rulesDoc.Raw, "rules:") {
		t.Fatalf("rules edi = %q", rulesDoc.Raw)
	}
	if !strings.HasPrefix(templateDoc.Raw, "template:") {
		t.Fatalf("template edi = %q", templateDoc.Raw)
	}
}
