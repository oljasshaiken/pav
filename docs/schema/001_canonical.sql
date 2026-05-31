-- Canonical domain model (RFC-EDI-001 §5, corrected for Phase 0)
-- Partitioning deferred to Phase 1

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE agencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    state TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE patients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agency_id UUID NOT NULL REFERENCES agencies(id),
    medicaid_id TEXT NOT NULL,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    date_of_birth DATE NOT NULL,
    address JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE authorizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id UUID NOT NULL REFERENCES patients(id),
    payer_id TEXT NOT NULL,
    service_type TEXT NOT NULL,
    authorized_hours NUMERIC NOT NULL,
    authorized_from DATE NOT NULL,
    authorized_to DATE NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE visits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agency_id UUID NOT NULL REFERENCES agencies(id),
    patient_id UUID NOT NULL REFERENCES patients(id),
    caregiver_id UUID NOT NULL,
    authorization_id UUID REFERENCES authorizations(id),
    scheduled_start TIMESTAMPTZ,
    scheduled_end TIMESTAMPTZ,
    clock_in_time TIMESTAMPTZ,
    clock_out_time TIMESTAMPTZ,
    clock_in_location JSONB,
    clock_out_location JSONB,
    total_minutes INTEGER,
    evv_status TEXT NOT NULL,
    tasks JSONB,
    notes TEXT[],
    signatures JSONB,
    attachments TEXT[],
    offline_sync_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE claims (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payer_id TEXT NOT NULL,
    state TEXT NOT NULL,
    claim_number TEXT,
    status TEXT NOT NULL,
    submission_attempt INTEGER NOT NULL DEFAULT 0,
    last_submitted_at TIMESTAMPTZ,
    x12_837 TEXT,
    response_277 TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE claim_service_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    claim_id UUID NOT NULL REFERENCES claims(id),
    visit_id UUID NOT NULL REFERENCES visits(id),
    authorization_id UUID REFERENCES authorizations(id),
    procedure_code TEXT NOT NULL,
    modifier TEXT[],
    units NUMERIC NOT NULL,
    amount NUMERIC,
    diagnosis_codes TEXT[],
    evv_segment_data JSONB
);

CREATE INDEX idx_claims_state_payer_status ON claims (state, payer_id, status);
CREATE INDEX idx_visits_clock_in_location ON visits USING GIN (clock_in_location);
CREATE INDEX idx_claim_service_lines_evv ON claim_service_lines USING GIN (evv_segment_data);
