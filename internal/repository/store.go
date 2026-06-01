package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pavillio/pav-edi/internal/domain"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrNoServiceLines = errors.New("claim has no service lines")
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Ping checks postgres connectivity.
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *Store) GetActivePayerConfig(ctx context.Context, state, payerID, txType string) (domain.PayerConfig, error) {
	const q = `
SELECT id, state, payer_id, transaction_type, config_version, destination, active, config, updated_by
FROM payer_configs
WHERE state = $1 AND payer_id = $2 AND transaction_type = $3 AND active = true
ORDER BY config_version DESC
LIMIT 1`

	var pc domain.PayerConfig
	var dest []byte
	var cfgBytes []byte
	var updatedBy *string

	err := s.pool.QueryRow(ctx, q, state, payerID, txType).Scan(
		&pc.ID, &pc.State, &pc.PayerID, &pc.TransactionType, &pc.ConfigVersion,
		&dest, &pc.Active, &cfgBytes, &updatedBy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.PayerConfig{}, ErrNotFound
	}
	if err != nil {
		return domain.PayerConfig{}, fmt.Errorf("query payer config: %w", err)
	}
	pc.Destination = dest
	pc.UpdatedBy = updatedBy
	if err := json.Unmarshal(cfgBytes, &pc.Config); err != nil {
		return domain.PayerConfig{}, fmt.Errorf("unmarshal payer config: %w", err)
	}
	return pc, nil
}

func (s *Store) GetActiveTemplateOverride(ctx context.Context, state, payerID, txType string) (domain.TemplateOverride, error) {
	const q = `
SELECT o.id, o.template_id, o.state, o.payer_id, o.override_version, o.mapper, o.destination,
       t.id, t.name, t.transaction_type, t.x12_version, t.template
FROM template_overrides o
JOIN x12_templates t ON t.id = o.template_id
WHERE o.state = $1 AND o.payer_id = $2 AND t.transaction_type = $3
  AND t.name = '837P-base' AND o.active = true
ORDER BY o.override_version DESC
LIMIT 1`

	var o domain.TemplateOverride
	var mapper, dest, tmpl []byte

	err := s.pool.QueryRow(ctx, q, state, payerID, txType).Scan(
		&o.ID, &o.TemplateID, &o.State, &o.PayerID, &o.OverrideVersion, &mapper, &dest,
		&o.Template.ID, &o.Template.Name, &o.Template.TransactionType, &o.Template.X12Version, &tmpl,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.TemplateOverride{}, ErrNotFound
	}
	if err != nil {
		return domain.TemplateOverride{}, fmt.Errorf("query template override: %w", err)
	}
	o.Mapper = mapper
	o.Destination = dest
	o.Template.Template = tmpl
	return o, nil
}

func (s *Store) LoadClaimContext(ctx context.Context, claimID uuid.UUID) (domain.ClaimContext, error) {
	var claim domain.Claim
	err := s.pool.QueryRow(ctx, `
SELECT id, payer_id, state, claim_number, status, submission_attempt, last_submitted_at, x12_837, response_277, created_at
FROM claims WHERE id = $1`, claimID).Scan(
		&claim.ID, &claim.PayerID, &claim.State, &claim.ClaimNumber, &claim.Status,
		&claim.SubmissionAttempt, &claim.LastSubmittedAt, &claim.X12_837, &claim.Response277, &claim.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ClaimContext{}, ErrNotFound
	}
	if err != nil {
		return domain.ClaimContext{}, fmt.Errorf("load claim: %w", err)
	}

	lines, err := s.listServiceLines(ctx, claimID)
	if err != nil {
		return domain.ClaimContext{}, err
	}
	if len(lines) == 0 {
		return domain.ClaimContext{}, ErrNoServiceLines
	}

	visits, err := s.loadVisits(ctx, lines)
	if err != nil {
		return domain.ClaimContext{}, err
	}

	// Phase 0: single-patient claims — patient/agency from first visit (see PHASE-0 spec).
	patient, err := s.loadPatient(ctx, visits[0].PatientID)
	if err != nil {
		return domain.ClaimContext{}, err
	}

	auth, err := s.resolveAuthorization(ctx, lines[0], claim.PayerID, patient.ID)
	if err != nil {
		return domain.ClaimContext{}, err
	}

	agency, err := s.loadAgency(ctx, visits[0].AgencyID)
	if err != nil {
		return domain.ClaimContext{}, err
	}

	return domain.ClaimContext{
		Claim:         claim,
		ServiceLines:  lines,
		Visits:        visits,
		Patient:       patient,
		Authorization: auth,
		Agency:        agency,
	}, nil
}

func (s *Store) listServiceLines(ctx context.Context, claimID uuid.UUID) ([]domain.ClaimServiceLine, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, claim_id, visit_id, authorization_id, procedure_code, modifier, units, amount, diagnosis_codes, evv_segment_data
FROM claim_service_lines WHERE claim_id = $1`, claimID)
	if err != nil {
		return nil, fmt.Errorf("list service lines: %w", err)
	}
	defer rows.Close()

	var lines []domain.ClaimServiceLine
	for rows.Next() {
		var line domain.ClaimServiceLine
		var evv []byte
		if err := rows.Scan(
			&line.ID, &line.ClaimID, &line.VisitID, &line.AuthorizationID,
			&line.ProcedureCode, &line.Modifier, &line.Units, &line.Amount,
			&line.DiagnosisCodes, &evv,
		); err != nil {
			return nil, fmt.Errorf("scan service line: %w", err)
		}
		line.EVVSegmentData = evv
		lines = append(lines, line)
	}
	return lines, rows.Err()
}

func (s *Store) loadVisits(ctx context.Context, lines []domain.ClaimServiceLine) ([]domain.Visit, error) {
	seen := map[uuid.UUID]struct{}{}
	var visits []domain.Visit
	for _, line := range lines {
		if _, ok := seen[line.VisitID]; ok {
			continue
		}
		seen[line.VisitID] = struct{}{}
		v, err := s.loadVisit(ctx, line.VisitID)
		if err != nil {
			return nil, err
		}
		visits = append(visits, v)
	}
	return visits, nil
}

func (s *Store) loadVisit(ctx context.Context, id uuid.UUID) (domain.Visit, error) {
	var v domain.Visit
	var tasks, sigs []byte
	err := s.pool.QueryRow(ctx, `
SELECT id, agency_id, patient_id, caregiver_id, authorization_id,
       scheduled_start, scheduled_end, clock_in_time, clock_out_time,
       clock_in_location, clock_out_location, total_minutes, evv_status,
       tasks, notes, signatures, attachments, offline_sync_at, created_at, updated_at
FROM visits WHERE id = $1`, id).Scan(
		&v.ID, &v.AgencyID, &v.PatientID, &v.CaregiverID, &v.AuthorizationID,
		&v.ScheduledStart, &v.ScheduledEnd, &v.ClockInTime, &v.ClockOutTime,
		&v.ClockInLocation, &v.ClockOutLocation, &v.TotalMinutes, &v.EVVStatus,
		&tasks, &v.Notes, &sigs, &v.Attachments, &v.OfflineSyncAt, &v.CreatedAt, &v.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Visit{}, fmt.Errorf("visit %s: %w", id, ErrNotFound)
	}
	if err != nil {
		return domain.Visit{}, fmt.Errorf("load visit: %w", err)
	}
	v.Tasks = tasks
	v.Signatures = sigs
	return v, nil
}

func (s *Store) loadPatient(ctx context.Context, id uuid.UUID) (domain.Patient, error) {
	var p domain.Patient
	var addr []byte
	err := s.pool.QueryRow(ctx, `
SELECT id, agency_id, medicaid_id, first_name, last_name, date_of_birth, address, created_at
FROM patients WHERE id = $1`, id).Scan(
		&p.ID, &p.AgencyID, &p.MedicaidID, &p.FirstName, &p.LastName, &p.DateOfBirth, &addr, &p.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Patient{}, ErrNotFound
	}
	if err != nil {
		return domain.Patient{}, fmt.Errorf("load patient: %w", err)
	}
	p.Address = addr
	return p, nil
}

func (s *Store) loadAgency(ctx context.Context, id uuid.UUID) (domain.Agency, error) {
	var a domain.Agency
	err := s.pool.QueryRow(ctx, `SELECT id, name, state, created_at FROM agencies WHERE id = $1`, id).
		Scan(&a.ID, &a.Name, &a.State, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Agency{}, ErrNotFound
	}
	if err != nil {
		return domain.Agency{}, fmt.Errorf("load agency: %w", err)
	}
	return a, nil
}

func (s *Store) resolveAuthorization(ctx context.Context, line domain.ClaimServiceLine, payerID string, patientID uuid.UUID) (domain.Authorization, error) {
	if line.AuthorizationID != nil {
		return s.loadAuthorization(ctx, *line.AuthorizationID)
	}
	var a domain.Authorization
	err := s.pool.QueryRow(ctx, `
SELECT id, patient_id, payer_id, service_type, authorized_hours, authorized_from, authorized_to, status, created_at
FROM authorizations
WHERE patient_id = $1 AND payer_id = $2 AND status = 'ACTIVE'
ORDER BY authorized_to DESC
LIMIT 1`, patientID, payerID).Scan(
		&a.ID, &a.PatientID, &a.PayerID, &a.ServiceType, &a.AuthorizedHours,
		&a.AuthorizedFrom, &a.AuthorizedTo, &a.Status, &a.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Authorization{}, ErrNotFound
	}
	if err != nil {
		return domain.Authorization{}, fmt.Errorf("resolve authorization: %w", err)
	}
	return a, nil
}

func (s *Store) loadAuthorization(ctx context.Context, id uuid.UUID) (domain.Authorization, error) {
	var a domain.Authorization
	err := s.pool.QueryRow(ctx, `
SELECT id, patient_id, payer_id, service_type, authorized_hours, authorized_from, authorized_to, status, created_at
FROM authorizations WHERE id = $1`, id).Scan(
		&a.ID, &a.PatientID, &a.PayerID, &a.ServiceType, &a.AuthorizedHours,
		&a.AuthorizedFrom, &a.AuthorizedTo, &a.Status, &a.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Authorization{}, ErrNotFound
	}
	if err != nil {
		return domain.Authorization{}, fmt.Errorf("load authorization: %w", err)
	}
	return a, nil
}

// SaveGeneratedEDI persists dry-run submit output for a claim.
func (s *Store) SaveGeneratedEDI(ctx context.Context, claimID uuid.UUID, edi string) (int32, error) {
	var attempt int32
	err := s.pool.QueryRow(ctx, `
UPDATE claims
SET x12_837 = $2, submission_attempt = submission_attempt + 1, last_submitted_at = now()
WHERE id = $1
RETURNING submission_attempt`, claimID, edi).Scan(&attempt)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("save generated edi: %w", err)
	}
	return attempt, nil
}

// SaveResponse277 stores an inbound 277/999 acknowledgment on the claim.
func (s *Store) SaveResponse277(ctx context.Context, claimID uuid.UUID, response string) error {
	tag, err := s.pool.Exec(ctx, `
UPDATE claims SET response_277 = $2 WHERE id = $1`, claimID, response)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("save response 277: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// FindClaimIDByNumber resolves a claim UUID from its claim_number.
func (s *Store) FindClaimIDByNumber(ctx context.Context, claimNumber string) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT id FROM claims WHERE claim_number = $1 LIMIT 1`, claimNumber).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("find claim by number: %w", err)
	}
	return id, nil
}
