package evvrules

import "github.com/pavillio/pav-edi/internal/domain"

// Standard returns reusable EVV CEL rules for payer config evv_rules.
// Payers pick a subset; expressions assume visit.* bindings from internal/cel.
func Standard() []domain.CELRule {
	return []domain.CELRule{
		{
			ID:      "evv_verified",
			CEL:     `visit.evv_status == "VERIFIED"`,
			Message: "EVV visit must be verified before billing",
		},
		{
			ID:      "gps_at_clock_in",
			CEL:     `visit.clock_in_has_gps`,
			Message: "Clock-in GPS location required",
		},
		{
			ID:      "gps_at_clock_out",
			CEL:     `visit.clock_out_has_gps`,
			Message: "Clock-out GPS location required",
		},
		{
			ID:      "tasks_completed",
			CEL:     `visit.task_count == 0 || visit.tasks_all_completed`,
			Message: "All documented visit tasks must be completed",
		},
		{
			ID:      "patient_signature",
			CEL:     `visit.has_patient_signature`,
			Message: "Patient signature required",
		},
		{
			ID:      "caregiver_signature",
			CEL:     `visit.has_caregiver_signature`,
			Message: "Caregiver signature required",
		},
		{
			ID:      "offline_sync_required",
			CEL:     `visit.evv_status != "OFFLINE_PENDING" || visit.offline_synced`,
			Message: "Offline-captured visit must be synced before billing",
		},
	}
}
