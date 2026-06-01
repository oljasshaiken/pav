package states_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pavillio/pav-edi/internal/cel"
	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/lambda/load"
	"github.com/pavillio/pav-edi/internal/lambda/persist"
	"github.com/pavillio/pav-edi/internal/lambda/rules"
	"github.com/pavillio/pav-edi/internal/lambda/transformer"
	"github.com/pavillio/pav-edi/internal/repository"
	"github.com/pavillio/pav-edi/internal/states"
	"github.com/pavillio/pav-edi/internal/testutil"
	"github.com/pavillio/pav-edi/internal/workflow"
)

func TestFL_CELHarness(t *testing.T) { runCELHarness(t, states.FL) }
func TestOH_CELHarness(t *testing.T) { runCELHarness(t, states.OH) }
func TestPA_CELHarness(t *testing.T) { runCELHarness(t, states.PA) }
func TestNY_CELHarness(t *testing.T) { runCELHarness(t, states.NY) }

func TestFL_OutboundGolden(t *testing.T) { runOutboundGolden(t, states.FL) }
func TestOH_OutboundGolden(t *testing.T) { runOutboundGolden(t, states.OH) }
func TestPA_OutboundGolden(t *testing.T) { runOutboundGolden(t, states.PA) }
func TestNY_OutboundGolden(t *testing.T) { runOutboundGolden(t, states.NY) }

func runCELHarness(t *testing.T, fx states.Fixture) {
	t.Helper()
	cfg := loadPayerConfig(t, fx.ConfigPath)

	validation, err := cfg.CELValidationRules()
	if err != nil {
		t.Fatal(err)
	}
	if len(validation) < 2 {
		t.Fatalf("expected at least 2 validation rules, got %d", len(validation))
	}
	foundExtra := false
	for _, rule := range validation {
		if rule.ID == fx.ExtraValidationID && rule.CEL != "" {
			foundExtra = true
		}
	}
	if !foundExtra {
		t.Fatalf("missing state validation rule %q", fx.ExtraValidationID)
	}

	evv, err := cfg.CELEvvRules()
	if err != nil {
		t.Fatal(err)
	}
	allRules := append(validation, evv...)
	rs, err := cel.NewRuleSet(allRules)
	if err != nil {
		t.Fatal(err)
	}

	ctx := claimContextFor(fx)
	if err := rs.Evaluate(cel.ClaimBindings(ctx)); err != nil {
		t.Fatalf("expected no violations: %v", err)
	}

	if fx.ExtraValidationID == "ny_agency_state" {
		bad := ctx
		bad.Agency.State = "TX"
		if err := rs.Evaluate(cel.ClaimBindings(bad)); err == nil {
			t.Fatal("expected NY agency state violation")
		}
	}
	if fx.ExtraValidationID == "fl_units_positive" {
		bad := ctx
		bad.ServiceLines[0].Units = 0
		if err := rs.Evaluate(cel.ClaimBindings(bad)); err == nil {
			t.Fatal("expected FL units violation")
		}
	}
}

func runOutboundGolden(t *testing.T, fx states.Fixture) {
	t.Helper()
	pool := testutil.StartPostgres(t)
	seedStateClaim(t, pool, fx)

	mem := &persist.MemoryObjectStore{}
	wf := newOutboundWorkflow(pool, mem, goldenTime())
	result, err := wf.Run(context.Background(), fx.ClaimID.String())
	if err != nil {
		t.Fatal(err)
	}

	goldenBytes, err := os.ReadFile(filepath.Join("..", "..", fx.GoldenPath))
	if err != nil {
		t.Fatal(err)
	}
	if collapseEDI(result.EDI) != collapseEDI(string(goldenBytes)) {
		t.Fatal("workflow edi does not match golden")
	}
}

func seedStateClaim(t *testing.T, pool *pgxpool.Pool, fx states.Fixture) {
	t.Helper()
	testutil.InsertStateFixtureClaim(t, pool, fx.ClaimID, testutil.StateClaimParams{
		State: fx.State, PayerID: fx.PayerID, ClaimNumber: fx.ClaimNumber,
		MedicaidID: fx.MedicaidID, AgencyName: fx.AgencyName,
	})
	testutil.InsertStatePayerConfig(t, pool, fx.State, fx.PayerID, fx.ConfigPath)
}

func loadPayerConfig(t *testing.T, relPath string) domain.PayerConfigBody {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", relPath))
	if err != nil {
		t.Fatal(err)
	}
	var cfg domain.PayerConfigBody
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func claimContextFor(fx states.Fixture) domain.ClaimContext {
	claimNumber := fx.ClaimNumber
	amount := 100.00
	clockIn := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	return domain.ClaimContext{
		Claim: domain.Claim{
			ID: fx.ClaimID, PayerID: fx.PayerID, State: fx.State, ClaimNumber: &claimNumber,
		},
		Authorization: domain.Authorization{ServiceType: "home_health"},
		Patient:       domain.Patient{FirstName: "Synthetic", LastName: "Patient", MedicaidID: fx.MedicaidID},
		Agency:        domain.Agency{Name: fx.AgencyName, State: fx.State, NPI: "1234567890"},
		Visits:        []domain.Visit{{ClockInTime: &clockIn, EVVStatus: "VERIFIED"}},
		ServiceLines: []domain.ClaimServiceLine{{
			ProcedureCode: "T1019", Units: 4, Amount: &amount, DiagnosisCodes: []string{"Z9999"},
		}},
	}
}

func newOutboundWorkflow(pool *pgxpool.Pool, mem *persist.MemoryObjectStore, now time.Time) *workflow.Outbound {
	store := repository.New(pool)
	return &workflow.Outbound{
		Load:        &load.Handler{Store: store},
		Rules:       &rules.Handler{},
		Transform:   &transformer.Handler{Now: func() time.Time { return now }},
		Persist:     &persist.Handler{Store: store, Object: mem},
		Now:         func() time.Time { return now },
		S3Bucket:    "pav-edi-outbound",
		S3KeyPrefix: "",
	}
}

func goldenTime() time.Time {
	return time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
}

func collapseEDI(s string) string {
	return strings.ReplaceAll(s, "\n", "")
}
