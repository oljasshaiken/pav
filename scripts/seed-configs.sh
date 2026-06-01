#!/usr/bin/env bash
# Load Phase 1 TX reference payer config and template override from docs/fixtures.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CONTAINER="${PAV_POSTGRES_CONTAINER:-pav-postgres-1}"

insert_payer_config() {
  docker exec -i "$CONTAINER" psql -U pav -d pav -v ON_ERROR_STOP=1 <<EOSQL
DELETE FROM payer_configs WHERE state = 'TX' AND payer_id = 'TX-MCO-001';
INSERT INTO payer_configs (state, payer_id, transaction_type, config_version, config, active)
VALUES ('TX', 'TX-MCO-001', '837P', 1, \$PAYCFG\$
$(cat "$ROOT/docs/fixtures/payer_config_837p_tx.json")
\$PAYCFG\$::jsonb, true);
EOSQL
}

insert_template() {
  docker exec -i "$CONTAINER" psql -U pav -d pav -v ON_ERROR_STOP=1 <<EOSQL
INSERT INTO x12_templates (id, name, transaction_type, x12_version, template)
VALUES (
  '11111111-1111-1111-1111-111111111111',
  '837P-base',
  '837P',
  '005010X222A1',
  \$TMPLCFG\$
$(cat "$ROOT/docs/fixtures/template_837p_base.json")
\$TMPLCFG\$::jsonb
)
ON CONFLICT (id) DO UPDATE SET template = EXCLUDED.template;
EOSQL
}

insert_override() {
  docker exec -i "$CONTAINER" psql -U pav -d pav -v ON_ERROR_STOP=1 <<EOSQL
DELETE FROM template_overrides WHERE state = 'TX' AND payer_id = 'TX-MCO-001';
INSERT INTO template_overrides (template_id, state, payer_id, override_version, mapper, active)
VALUES (
  '11111111-1111-1111-1111-111111111111',
  'TX',
  'TX-MCO-001',
  1,
  \$MAPCFG\$
$(cat "$ROOT/docs/fixtures/override_837p_tx.json")
\$MAPCFG\$::jsonb,
  true
);
EOSQL
}

insert_payer_config
insert_template
insert_override
