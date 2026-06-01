package testutil

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StateClaimParams identifies synthetic claim seed data for a target state.
type StateClaimParams struct {
	State       string
	PayerID     string
	ClaimNumber string
	MedicaidID  string
	AgencyName  string
}

// InsertStateFixtureClaim seeds a golden-style claim for FL/OH/PA/NY harness tests.
func InsertStateFixtureClaim(t *testing.T, pool *pgxpool.Pool, claimID uuid.UUID, p StateClaimParams) {
	t.Helper()
	ctx := context.Background()
	agencyID := uuid.New()
	patientID := uuid.New()
	authID := uuid.New()
	visitID := uuid.New()
	lineID := uuid.New()

	_, err := pool.Exec(ctx, `
INSERT INTO agencies (id, name, state) VALUES ($1, $2, $3)`, agencyID, p.AgencyName, p.State)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
INSERT INTO patients (id, agency_id, medicaid_id, first_name, last_name, date_of_birth)
VALUES ($1, $2, $3, 'Synthetic', 'Patient', '1975-06-15')`, patientID, agencyID, p.MedicaidID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
INSERT INTO authorizations (id, patient_id, payer_id, service_type, authorized_hours, authorized_from, authorized_to, status)
VALUES ($1, $2, $3, 'home_health', 40, '2026-01-01', '2026-12-31', 'ACTIVE')`, authID, patientID, p.PayerID)
	if err != nil {
		t.Fatal(err)
	}
	loc := []byte(`{"lat":30.2672,"lng":-97.7431}`)
	_, err = pool.Exec(ctx, `
INSERT INTO visits (id, agency_id, patient_id, caregiver_id, authorization_id, evv_status, clock_in_location, clock_in_time, clock_out_time, total_minutes)
VALUES ($1, $2, $3, $4, $5, 'VERIFIED', $6, $7, $8, 60)`,
		visitID, agencyID, patientID, uuid.New(), authID, loc,
		time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
INSERT INTO claims (id, payer_id, state, claim_number, status)
VALUES ($1, $2, $3, $4, 'DRAFT')`, claimID, p.PayerID, p.State, p.ClaimNumber)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
INSERT INTO claim_service_lines (id, claim_id, visit_id, authorization_id, procedure_code, units, amount, diagnosis_codes)
VALUES ($1, $2, $3, $4, 'T1019', 4, 100.00, ARRAY['Z9999'])`, lineID, claimID, visitID, authID)
	if err != nil {
		t.Fatal(err)
	}
}

// InsertStatePayerConfig loads a payer config fixture JSON for the given state/payer.
func InsertStatePayerConfig(t *testing.T, pool *pgxpool.Pool, state, payerID, configRelPath string) {
	t.Helper()
	cfgBytes, err := os.ReadFile(filepath.Join("..", "..", configRelPath))
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(context.Background(), `
INSERT INTO payer_configs (state, payer_id, transaction_type, config_version, config, active)
VALUES ($1, $2, '837P', 1, $3, true)`, state, payerID, cfgBytes)
	if err != nil {
		t.Fatal(err)
	}
}
