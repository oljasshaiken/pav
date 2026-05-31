package repository_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/repository"
	"github.com/pavillio/pav-edi/internal/testutil"
)

func TestPayerConfig_GetActive(t *testing.T) {
	pool := testutil.StartPostgres(t)
	store := repository.New(pool)
	ctx := context.Background()

	testutil.InsertPayerConfig(t, pool)

	pc, err := store.GetActivePayerConfig(ctx, "TX", "TX-MCO-001", "837P")
	if err != nil {
		t.Fatal(err)
	}
	if pc.ConfigVersion != 1 {
		t.Fatalf("version = %d", pc.ConfigVersion)
	}
}

func TestTemplate_GetActiveOverride(t *testing.T) {
	pool := testutil.StartPostgres(t)
	store := repository.New(pool)
	ctx := context.Background()

	testutil.InsertTemplateOverride(t, pool)

	o, err := store.GetActiveTemplateOverride(ctx, "TX", "TX-MCO-001", "837P")
	if err != nil {
		t.Fatal(err)
	}
	if o.Template.Name != "837P-base" {
		t.Fatalf("template name = %q", o.Template.Name)
	}
}

func TestClaimContext_Load(t *testing.T) {
	pool := testutil.StartPostgres(t)
	store := repository.New(pool)
	ctx := context.Background()

	claimID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	testutil.InsertFixtureClaim(t, pool, claimID)

	cc, err := store.LoadClaimContext(ctx, claimID)
	if err != nil {
		t.Fatal(err)
	}
	if cc.Claim.ID != claimID {
		t.Fatalf("claim id mismatch")
	}
	if len(cc.ServiceLines) != 1 {
		t.Fatalf("service lines = %d", len(cc.ServiceLines))
	}
	if cc.Patient.FirstName == "" {
		t.Fatal("expected patient")
	}
}

func TestPayerConfig_NotFound(t *testing.T) {
	pool := testutil.StartPostgres(t)
	store := repository.New(pool)

	_, err := store.GetActivePayerConfig(context.Background(), "TX", "missing", "837P")
	if err != repository.ErrNotFound {
		t.Fatalf("err = %v", err)
	}
}

func TestClaimContext_MissingClaim(t *testing.T) {
	pool := testutil.StartPostgres(t)
	store := repository.New(pool)

	_, err := store.LoadClaimContext(context.Background(), uuid.New())
	if err != repository.ErrNotFound {
		t.Fatalf("err = %v", err)
	}
}

// Ensure fixture JSON still loads (guards testutil paths).
func TestFixturesExist(t *testing.T) {
	for _, f := range []string{
		"docs/fixtures/payer_config_837p_tx.json",
		"docs/fixtures/template_837p_base.json",
		"docs/fixtures/override_837p_tx.json",
	} {
		if _, err := os.Stat(filepath.Join("..", "..", f)); err != nil {
			t.Fatal(err)
		}
	}
}
