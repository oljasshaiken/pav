CREATE TABLE eligibility_responses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id UUID NOT NULL REFERENCES patients(id),
    payer_id TEXT NOT NULL,
    inquiry_ref TEXT NOT NULL,
    coverage_status TEXT NOT NULL,
    service_type TEXT,
    response_271 TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_eligibility_responses_patient_payer
    ON eligibility_responses (patient_id, payer_id, created_at DESC);
