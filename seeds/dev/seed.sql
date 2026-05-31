-- Synthetic PHI only — Phase 0 dev seed
-- Claim UUID for make compare: 00000000-0000-4000-8000-000000000001

INSERT INTO agencies (id, name, state)
VALUES ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Demo Home Care TX', 'TX')
ON CONFLICT (id) DO NOTHING;

INSERT INTO patients (id, agency_id, medicaid_id, first_name, last_name, date_of_birth, address)
VALUES (
  'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
  'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
  'SYN-TX-00001',
  'Synthetic',
  'Patient',
  '1975-06-15',
  '{"street":"123 Demo St","city":"Austin","state":"TX","zip":"78701"}'::jsonb
) ON CONFLICT (id) DO NOTHING;

INSERT INTO authorizations (id, patient_id, payer_id, service_type, authorized_hours, authorized_from, authorized_to, status)
VALUES (
  'cccccccc-cccc-cccc-cccc-cccccccccccc',
  'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
  'TX-MCO-001',
  'home_health',
  40,
  '2026-01-01',
  '2026-12-31',
  'ACTIVE'
) ON CONFLICT (id) DO NOTHING;

INSERT INTO visits (id, agency_id, patient_id, caregiver_id, authorization_id, evv_status,
  clock_in_time, clock_out_time, total_minutes, clock_in_location, tasks)
VALUES (
  'dddddddd-dddd-dddd-dddd-dddddddddddd',
  'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
  'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
  'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee',
  'cccccccc-cccc-cccc-cccc-cccccccccccc',
  'VERIFIED',
  '2026-05-01T09:00:00Z',
  '2026-05-01T10:00:00Z',
  60,
  '{"lat":30.2672,"lng":-97.7431,"method":"GPS"}'::jsonb,
  '[{"task":"bathing","completed_at":"2026-05-01T09:15:00Z"}]'::jsonb
) ON CONFLICT (id) DO NOTHING;

INSERT INTO claims (id, payer_id, state, claim_number, status)
VALUES (
  '00000000-0000-4000-8000-000000000001',
  'TX-MCO-001',
  'TX',
  'CLM-DEMO-001',
  'DRAFT'
) ON CONFLICT (id) DO NOTHING;

INSERT INTO claim_service_lines (id, claim_id, visit_id, authorization_id, procedure_code, units, amount)
VALUES (
  'ffffffff-ffff-ffff-ffff-ffffffffffff',
  '00000000-0000-4000-8000-000000000001',
  'dddddddd-dddd-dddd-dddd-dddddddddddd',
  'cccccccc-cccc-cccc-cccc-cccccccccccc',
  'T1019',
  4,
  100.00
) ON CONFLICT (id) DO NOTHING;

INSERT INTO payer_configs (state, payer_id, transaction_type, config_version, config, active)
SELECT 'TX', 'TX-MCO-001', '837P', 1,
  '{"x12_version":"005010X222A1","envelope":{"isa":{"sender_id":"PAVILLIO"}},"mappings":{"patient":{"loop_2010BA":{"NM103":"patient.last_name"}}},"validation_rules":[],"business_rules":{}}'::jsonb,
  true
WHERE NOT EXISTS (
  SELECT 1 FROM payer_configs
  WHERE state = 'TX' AND payer_id = 'TX-MCO-001' AND transaction_type = '837P' AND config_version = 1
);

INSERT INTO x12_templates (id, name, transaction_type, x12_version, template)
VALUES (
  '11111111-1111-1111-1111-111111111111',
  '837P-base',
  '837P',
  '005010X222A1',
  '{"loops":[{"id":"2300","segments":[{"tag":"CLM"}]}]}'::jsonb
) ON CONFLICT (id) DO NOTHING;

INSERT INTO template_overrides (template_id, state, payer_id, override_version, mapper, active)
SELECT '11111111-1111-1111-1111-111111111111', 'TX', 'TX-MCO-001', 1,
  '{"mappings":{"patient":{"loop_2010BA":{"NM103":"patient.last_name"}}}}'::jsonb,
  true
WHERE NOT EXISTS (
  SELECT 1 FROM template_overrides
  WHERE template_id = '11111111-1111-1111-1111-111111111111'
    AND state = 'TX' AND payer_id = 'TX-MCO-001' AND override_version = 1
);
