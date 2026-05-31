CREATE TABLE payer_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    state TEXT NOT NULL,
    payer_id TEXT NOT NULL,
    transaction_type TEXT NOT NULL,
    config_version INTEGER NOT NULL,
    destination JSONB,
    active BOOLEAN NOT NULL DEFAULT true,
    config JSONB NOT NULL,
    updated_by TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (state, payer_id, transaction_type, config_version)
);

CREATE INDEX idx_payer_configs_lookup
    ON payer_configs (state, payer_id, transaction_type, config_version DESC)
    WHERE active;

CREATE INDEX idx_payer_configs_config_gin ON payer_configs USING GIN (config);
