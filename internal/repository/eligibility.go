package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/pavillio/pav-edi/internal/domain"
)

// SaveEligibilityResponse inserts a parsed 271 eligibility response.
func (s *Store) SaveEligibilityResponse(ctx context.Context, row domain.EligibilityResponse) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `
INSERT INTO eligibility_responses (patient_id, payer_id, inquiry_ref, coverage_status, service_type, response_271)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id`,
		row.PatientID, row.PayerID, row.InquiryRef, row.CoverageStatus, nullIfEmpty(row.ServiceType), row.Response271,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("save eligibility response: %w", err)
	}
	return id, nil
}

// FindPatientIDByMedicaidID resolves a patient UUID from medicaid_id.
func (s *Store) FindPatientIDByMedicaidID(ctx context.Context, medicaidID string) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT id FROM patients WHERE medicaid_id = $1 LIMIT 1`, medicaidID).Scan(&id)
	if err == pgx.ErrNoRows {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("find patient by medicaid_id: %w", err)
	}
	return id, nil
}

// PatientIDForClaim returns the patient linked to a claim via its first service line visit.
func (s *Store) PatientIDForClaim(ctx context.Context, claimID uuid.UUID) (uuid.UUID, error) {
	var patientID uuid.UUID
	err := s.pool.QueryRow(ctx, `
SELECT v.patient_id
FROM claim_service_lines csl
JOIN visits v ON v.id = csl.visit_id
WHERE csl.claim_id = $1
LIMIT 1`, claimID).Scan(&patientID)
	if err == pgx.ErrNoRows {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("patient for claim: %w", err)
	}
	return patientID, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
