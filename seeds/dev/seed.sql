-- Synthetic PHI only — Phase 1 dev seed (claim data)
-- Full payer/template configs: run `make seed-configs` after this file.
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

INSERT INTO claim_service_lines (id, claim_id, visit_id, authorization_id, procedure_code, units, amount, diagnosis_codes)
VALUES (
  'ffffffff-ffff-ffff-ffff-ffffffffffff',
  '00000000-0000-4000-8000-000000000001',
  'dddddddd-dddd-dddd-dddd-dddddddddddd',
  'cccccccc-cccc-cccc-cccc-cccccccccccc',
  'T1019',
  4,
  100.00,
  ARRAY['Z9999']
) ON CONFLICT (id) DO UPDATE SET diagnosis_codes = EXCLUDED.diagnosis_codes;
