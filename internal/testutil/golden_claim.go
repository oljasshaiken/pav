package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func InsertGoldenFixtureClaim(t *testing.T, pool *pgxpool.Pool, claimID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	agencyID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	patientID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	authID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	visitID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	lineID := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")

	_, err := pool.Exec(ctx, `
INSERT INTO agencies (id, name, state) VALUES ($1, 'Demo Home Care TX', 'TX')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name`, agencyID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
INSERT INTO patients (id, agency_id, medicaid_id, first_name, last_name, date_of_birth)
VALUES ($1, $2, 'SYN-TX-00001', 'Synthetic', 'Patient', '1975-06-15')
ON CONFLICT (id) DO NOTHING`, patientID, agencyID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
INSERT INTO authorizations (id, patient_id, payer_id, service_type, authorized_hours, authorized_from, authorized_to, status)
VALUES ($1, $2, 'TX-MCO-001', 'home_health', 40, '2026-01-01', '2026-12-31', 'ACTIVE')
ON CONFLICT (id) DO NOTHING`, authID, patientID)
	if err != nil {
		t.Fatal(err)
	}
	loc := []byte(`{"lat":30.2672,"lng":-97.7431}`)
	_, err = pool.Exec(ctx, `
INSERT INTO visits (id, agency_id, patient_id, caregiver_id, authorization_id, evv_status, clock_in_location, clock_in_time, clock_out_time, total_minutes)
VALUES ($1, $2, $3, $4, $5, 'VERIFIED', $6, $7, $8, 60)
ON CONFLICT (id) DO NOTHING`,
		visitID, agencyID, patientID, uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"), authID, loc,
		time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
INSERT INTO claims (id, payer_id, state, claim_number, status)
VALUES ($1, 'TX-MCO-001', 'TX', 'CLM-DEMO-001', 'DRAFT')
ON CONFLICT (id) DO UPDATE SET claim_number = EXCLUDED.claim_number`, claimID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
INSERT INTO claim_service_lines (id, claim_id, visit_id, authorization_id, procedure_code, units, amount, diagnosis_codes)
VALUES ($1, $2, $3, $4, 'T1019', 4, 100.00, ARRAY['Z9999'])
ON CONFLICT (id) DO NOTHING`, lineID, claimID, visitID, authID)
	if err != nil {
		t.Fatal(err)
	}
}
