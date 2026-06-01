package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Agency struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	NPI       string    `json:"npi,omitempty"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
}

type Patient struct {
	ID          uuid.UUID       `json:"id"`
	AgencyID    uuid.UUID       `json:"agency_id"`
	MedicaidID  string          `json:"medicaid_id"`
	FirstName   string          `json:"first_name"`
	LastName    string          `json:"last_name"`
	DateOfBirth time.Time       `json:"date_of_birth"`
	Address     json.RawMessage `json:"address,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

type Authorization struct {
	ID              uuid.UUID `json:"id"`
	PatientID       uuid.UUID `json:"patient_id"`
	PayerID         string    `json:"payer_id"`
	ServiceType     string    `json:"service_type"`
	AuthorizedHours float64   `json:"authorized_hours"`
	AuthorizedFrom  time.Time `json:"authorized_from"`
	AuthorizedTo    time.Time `json:"authorized_to"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

type Visit struct {
	ID               uuid.UUID       `json:"id"`
	AgencyID         uuid.UUID       `json:"agency_id"`
	PatientID        uuid.UUID       `json:"patient_id"`
	CaregiverID      uuid.UUID       `json:"caregiver_id"`
	AuthorizationID  *uuid.UUID      `json:"authorization_id,omitempty"`
	ScheduledStart   *time.Time      `json:"scheduled_start,omitempty"`
	ScheduledEnd     *time.Time      `json:"scheduled_end,omitempty"`
	ClockInTime      *time.Time      `json:"clock_in_time,omitempty"`
	ClockOutTime     *time.Time      `json:"clock_out_time,omitempty"`
	ClockInLocation  json.RawMessage `json:"clock_in_location,omitempty"`
	ClockOutLocation json.RawMessage `json:"clock_out_location,omitempty"`
	TotalMinutes     *int32          `json:"total_minutes,omitempty"`
	EVVStatus        string          `json:"evv_status"`
	Tasks            json.RawMessage `json:"tasks,omitempty"`
	Notes            []string        `json:"notes,omitempty"`
	Signatures       json.RawMessage `json:"signatures,omitempty"`
	Attachments      []string        `json:"attachments,omitempty"`
	OfflineSyncAt    *time.Time      `json:"offline_sync_at,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type Claim struct {
	ID                uuid.UUID  `json:"id"`
	PayerID           string     `json:"payer_id"`
	State             string     `json:"state"`
	ClaimNumber       *string    `json:"claim_number,omitempty"`
	Status            string     `json:"status"`
	SubmissionAttempt int32      `json:"submission_attempt"`
	LastSubmittedAt   *time.Time `json:"last_submitted_at,omitempty"`
	X12_837           *string    `json:"x12_837,omitempty"`
	Response277       *string    `json:"response_277,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

type ClaimServiceLine struct {
	ID              uuid.UUID       `json:"id"`
	ClaimID         uuid.UUID       `json:"claim_id"`
	VisitID         uuid.UUID       `json:"visit_id"`
	AuthorizationID *uuid.UUID      `json:"authorization_id,omitempty"`
	ProcedureCode   string          `json:"procedure_code"`
	Modifier        []string        `json:"modifier,omitempty"`
	Units           float64         `json:"units"`
	Amount          *float64        `json:"amount,omitempty"`
	DiagnosisCodes  []string        `json:"diagnosis_codes,omitempty"`
	EVVSegmentData  json.RawMessage `json:"evv_segment_data,omitempty"`
}

type ClaimContext struct {
	Claim         Claim
	ServiceLines  []ClaimServiceLine
	Visits        []Visit
	Patient       Patient
	Authorization Authorization
	Agency        Agency
}

const (
	ClaimStatusDraft  = "DRAFT"
	EVVStatusVerified = "VERIFIED"
)
