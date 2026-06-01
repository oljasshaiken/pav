package cel_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/pavillio/pav-edi/internal/cel"
	"github.com/pavillio/pav-edi/internal/cel/evvrules"
	"github.com/pavillio/pav-edi/internal/domain"
)

func TestEvaluateRules_gpsAtClockIn(t *testing.T) {
	rules := []domain.CELRule{{
		ID: "gps_at_clock_in", CEL: `visit.clock_in_has_gps`, Message: "GPS required",
	}}

	noGPS := cel.ClaimBindings(domain.ClaimContext{
		Visits: []domain.Visit{{EVVStatus: domain.EVVStatusVerified}},
	})
	if err := cel.EvaluateAll(rules, noGPS); err == nil {
		t.Fatal("expected GPS validation error")
	}

	withGPS := cel.ClaimBindings(domain.ClaimContext{
		Visits: []domain.Visit{{
			EVVStatus:       domain.EVVStatusVerified,
			ClockInLocation: json.RawMessage(`{"lat":30.27,"lng":-97.74}`),
		}},
	})
	if err := cel.EvaluateAll(rules, withGPS); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEvaluateRules_tasksAllCompleted(t *testing.T) {
	rules := []domain.CELRule{{
		ID: "tasks_completed", CEL: `visit.task_count == 0 || visit.tasks_all_completed`, Message: "tasks incomplete",
	}}

	incomplete := cel.ClaimBindings(domain.ClaimContext{
		Visits: []domain.Visit{{
			Tasks: json.RawMessage(`[{"task_id":"bathing","completed":false},{"task_id":"meal","completed":true}]`),
		}},
	})
	if err := cel.EvaluateAll(rules, incomplete); err == nil {
		t.Fatal("expected tasks validation error")
	}

	complete := cel.ClaimBindings(domain.ClaimContext{
		Visits: []domain.Visit{{
			Tasks: json.RawMessage(`[{"task_id":"bathing","completed":true}]`),
		}},
	})
	if err := cel.EvaluateAll(rules, complete); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEvaluateRules_signaturesObjectForm(t *testing.T) {
	rules := []domain.CELRule{{
		ID: "patient_signature", CEL: `visit.has_patient_signature`, Message: "patient sig required",
	}}

	missing := cel.ClaimBindings(domain.ClaimContext{
		Visits: []domain.Visit{{Signatures: json.RawMessage(`{"caregiver":true}`)}},
	})
	if err := cel.EvaluateAll(rules, missing); err == nil {
		t.Fatal("expected signature validation error")
	}

	present := cel.ClaimBindings(domain.ClaimContext{
		Visits: []domain.Visit{{Signatures: json.RawMessage(`{"patient":true,"caregiver":true}`)}},
	})
	if err := cel.EvaluateAll(rules, present); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEvaluateRules_signaturesArrayForm(t *testing.T) {
	rules := []domain.CELRule{{
		ID: "caregiver_signature", CEL: `visit.has_caregiver_signature`, Message: "caregiver sig required",
	}}

	present := cel.ClaimBindings(domain.ClaimContext{
		Visits: []domain.Visit{{Signatures: json.RawMessage(`[
			{"role":"patient","signed":true},
			{"role":"caregiver","signed":true}
		]`)}},
	})
	if err := cel.EvaluateAll(rules, present); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEvaluateRules_offlineSyncRequired(t *testing.T) {
	rules := []domain.CELRule{{
		ID: "offline_sync_required",
		CEL: `visit.evv_status != "OFFLINE_PENDING" || visit.offline_synced`,
		Message: "offline visit must sync",
	}}

	pending := cel.ClaimBindings(domain.ClaimContext{
		Visits: []domain.Visit{{EVVStatus: "OFFLINE_PENDING"}},
	})
	if err := cel.EvaluateAll(rules, pending); err == nil {
		t.Fatal("expected offline sync validation error")
	}

	synced := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	ok := cel.ClaimBindings(domain.ClaimContext{
		Visits: []domain.Visit{{EVVStatus: "OFFLINE_PENDING", OfflineSyncAt: &synced}},
	})
	if err := cel.EvaluateAll(rules, ok); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEVVRulesStandardLibraryCompiles(t *testing.T) {
	if _, err := cel.NewRuleSet(evvrules.Standard()); err != nil {
		t.Fatalf("standard evv rules must compile: %v", err)
	}
}

func TestEVVRulesStandardLibraryEvaluatesOnGoldenVisit(t *testing.T) {
	clockIn := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	syncAt := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	bindings := cel.ClaimBindings(domain.ClaimContext{
		Visits: []domain.Visit{{
			EVVStatus:        domain.EVVStatusVerified,
			ClockInTime:      &clockIn,
			ClockInLocation:  json.RawMessage(`{"lat":30.0,"lng":-97.0}`),
			ClockOutLocation: json.RawMessage(`{"lat":30.0,"lng":-97.0}`),
			Tasks:            json.RawMessage(`[{"task_id":"personal_care","completed":true}]`),
			Signatures:       json.RawMessage(`{"patient":true,"caregiver":true}`),
			OfflineSyncAt:      &syncAt,
		}},
	})

	// Use GPS + verified + tasks + signatures rules (skip offline pending rule).
	subset := []domain.CELRule{
		evvrules.Standard()[0],
		evvrules.Standard()[1],
		evvrules.Standard()[2],
		evvrules.Standard()[3],
		evvrules.Standard()[4],
		evvrules.Standard()[5],
	}
	if err := cel.EvaluateAll(subset, bindings); err != nil {
		t.Fatalf("golden visit should pass standard evv rules: %v", err)
	}
}
