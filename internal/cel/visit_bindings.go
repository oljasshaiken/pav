package cel

import (
	"encoding/json"

	"github.com/pavillio/pav-edi/internal/domain"
)

type geoPoint struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type visitTask struct {
	TaskID    string `json:"task_id"`
	Completed bool   `json:"completed"`
}

func visitBindings(v domain.Visit) map[string]any {
	inLat, inLng, inGPS := geoFromRaw(v.ClockInLocation)
	outLat, outLng, outGPS := geoFromRaw(v.ClockOutLocation)
	tasks, taskCount, tasksDone := tasksFromRaw(v.Tasks)
	patientSig, caregiverSig := signaturesFromRaw(v.Signatures)

	var totalMin int64
	if v.TotalMinutes != nil {
		totalMin = int64(*v.TotalMinutes)
	}

	return map[string]any{
		"evv_status":              v.EVVStatus,
		"total_minutes":           totalMin,
		"has_clock_in":            v.ClockInTime != nil,
		"has_clock_out":           v.ClockOutTime != nil,
		"clock_in_has_gps":        inGPS,
		"clock_out_has_gps":       outGPS,
		"clock_in_lat":            inLat,
		"clock_in_lng":            inLng,
		"clock_out_lat":           outLat,
		"clock_out_lng":           outLng,
		"task_count":              taskCount,
		"tasks_completed_count":   tasksDone,
		"tasks_all_completed":     taskCount > 0 && tasksDone == taskCount,
		"tasks":                   tasks,
		"has_patient_signature":   patientSig,
		"has_caregiver_signature": caregiverSig,
		"offline_synced":          v.OfflineSyncAt != nil,
	}
}

func geoFromRaw(raw json.RawMessage) (lat, lng float64, ok bool) {
	if len(raw) == 0 {
		return 0, 0, false
	}
	var p geoPoint
	if err := json.Unmarshal(raw, &p); err != nil {
		return 0, 0, false
	}
	return p.Lat, p.Lng, true
}

func tasksFromRaw(raw json.RawMessage) ([]map[string]any, int64, int64) {
	if len(raw) == 0 {
		return nil, 0, 0
	}
	var parsed []visitTask
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, 0, 0
	}
	out := make([]map[string]any, 0, len(parsed))
	var done int64
	for _, t := range parsed {
		out = append(out, map[string]any{
			"task_id":   t.TaskID,
			"completed": t.Completed,
		})
		if t.Completed {
			done++
		}
	}
	return out, int64(len(parsed)), done
}

func signaturesFromRaw(raw json.RawMessage) (patient, caregiver bool) {
	if len(raw) == 0 {
		return false, false
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err == nil {
		patient = signatureTruthy(obj["patient"])
		caregiver = signatureTruthy(obj["caregiver"])
		if patient || caregiver {
			return patient, caregiver
		}
	}

	var arr []map[string]any
	if err := json.Unmarshal(raw, &arr); err != nil {
		return false, false
	}
	for _, item := range arr {
		role, _ := item["role"].(string)
		switch role {
		case "patient":
			patient = signatureTruthy(item["signed"])
		case "caregiver":
			caregiver = signatureTruthy(item["signed"])
		}
	}
	return patient, caregiver
}

func signatureTruthy(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case json.RawMessage:
		if len(x) == 0 {
			return false
		}
		var b bool
		if json.Unmarshal(x, &b) == nil {
			return b
		}
		return true
	case map[string]any:
		return len(x) > 0
	default:
		return v != nil
	}
}
