package testutil

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func StartPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("pav"),
		postgres.WithUsername("pav"),
		postgres.WithPassword("pav"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2)),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	RunMigrations(t, connStr)
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func RunMigrations(t *testing.T, connStr string) {
	t.Helper()
	root := filepath.Join("..", "..")
	files := []string{
		"migrations/000001_canonical_domain.up.sql",
		"migrations/000002_option1_payer_configs.up.sql",
		"migrations/000003_option2_templates.up.sql",
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	for _, f := range files {
		sql, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("exec %s: %v", f, err)
		}
	}
}

func InsertFixtureClaim(t *testing.T, pool *pgxpool.Pool, claimID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	agencyID := uuid.New()
	patientID := uuid.New()
	authID := uuid.New()
	visitID := uuid.New()
	lineID := uuid.New()

	_, err := pool.Exec(ctx, `INSERT INTO agencies (id, name, state) VALUES ($1, 'Test Agency', 'TX')`, agencyID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
INSERT INTO patients (id, agency_id, medicaid_id, first_name, last_name, date_of_birth)
VALUES ($1, $2, 'SYN-123', 'Jane', 'Doe', '1980-01-01')`, patientID, agencyID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
INSERT INTO authorizations (id, patient_id, payer_id, service_type, authorized_hours, authorized_from, authorized_to, status)
VALUES ($1, $2, 'TX-MCO-001', 'home_health', 40, '2026-01-01', '2026-12-31', 'ACTIVE')`, authID, patientID)
	if err != nil {
		t.Fatal(err)
	}
	loc := []byte(`{"lat":30.0,"lng":-97.0}`)
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
VALUES ($1, 'TX-MCO-001', 'TX', 'CLM-001', 'DRAFT')`, claimID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
INSERT INTO claim_service_lines (id, claim_id, visit_id, authorization_id, procedure_code, units)
VALUES ($1, $2, $3, $4, 'T1019', 4)`, lineID, claimID, visitID, authID)
	if err != nil {
		t.Fatal(err)
	}
}

func InsertPayerConfig(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	cfgBytes, err := os.ReadFile(filepath.Join("..", "..", "docs", "fixtures", "payer_config_837p_tx.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(context.Background(), `
INSERT INTO payer_configs (state, payer_id, transaction_type, config_version, config, active)
VALUES ('TX', 'TX-MCO-001', '837P', 1, $1, true)`, cfgBytes)
	if err != nil {
		t.Fatal(err)
	}
}

func InsertTemplateOverride(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	tmpl, err := os.ReadFile(filepath.Join("..", "..", "docs", "fixtures", "template_837p_base.json"))
	if err != nil {
		t.Fatal(err)
	}
	mapper, err := os.ReadFile(filepath.Join("..", "..", "docs", "fixtures", "override_837p_tx.json"))
	if err != nil {
		t.Fatal(err)
	}
	var templateID uuid.UUID
	err = pool.QueryRow(ctx, `
INSERT INTO x12_templates (name, transaction_type, x12_version, template)
VALUES ('837P-base', '837P', '005010X222A1', $1) RETURNING id`, tmpl).Scan(&templateID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
INSERT INTO template_overrides (template_id, state, payer_id, override_version, mapper, active)
VALUES ($1, 'TX', 'TX-MCO-001', 1, $2, true)`, templateID, mapper)
	if err != nil {
		t.Fatal(err)
	}
}
