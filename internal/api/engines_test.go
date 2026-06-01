package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/repository"
	"github.com/pavillio/pav-edi/internal/rules"
	"github.com/pavillio/pav-edi/internal/template"
	"github.com/pavillio/pav-edi/internal/testutil"
)

func TestEnginesProduceMatchingEDI(t *testing.T) {
	pool := testutil.StartPostgres(t)
	store := repository.New(pool)
	ctx := context.Background()
	now := goldenNow()

	claimID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	testutil.InsertGoldenFixtureClaim(t, pool, claimID)
	testutil.InsertPayerConfig(t, pool)
	testutil.InsertTemplateOverride(t, pool)

	cc, err := store.LoadClaimContext(ctx, claimID)
	if err != nil {
		t.Fatal(err)
	}

	rulesDoc, err := (&rules.RulesEngine{Store: store, Now: func() time.Time { return now }}).Transform(ctx, cc)
	if err != nil {
		t.Fatal(err)
	}
	templateDoc, err := (&template.Renderer{Store: store, Now: func() time.Time { return now }}).Transform(ctx, cc)
	if err != nil {
		t.Fatal(err)
	}

	if rulesDoc.Raw != templateDoc.Raw {
		t.Fatalf("engines produced different edi")
	}
}
