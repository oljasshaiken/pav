CREATE TABLE x12_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    transaction_type TEXT NOT NULL,
    x12_version TEXT NOT NULL,
    template JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE template_overrides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID NOT NULL REFERENCES x12_templates(id),
    state TEXT NOT NULL,
    payer_id TEXT NOT NULL,
    override_version INTEGER NOT NULL,
    mapper JSONB NOT NULL,
    destination JSONB,
    active BOOLEAN NOT NULL DEFAULT true,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (template_id, state, payer_id, override_version)
);

CREATE INDEX idx_template_overrides_lookup
    ON template_overrides (state, payer_id, override_version DESC)
    WHERE active;
