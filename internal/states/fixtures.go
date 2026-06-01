package states

import (
	"github.com/google/uuid"
)

// Fixture describes synthetic payer config + claim data for a target state.
type Fixture struct {
	State       string
	PayerID     string
	ClaimID     uuid.UUID
	ClaimNumber string
	MedicaidID  string
	AgencyName  string
	ConfigPath  string
	GoldenPath  string
	// ExtraValidationID is a state-specific CEL validation rule id in the config.
	ExtraValidationID string
}

var (
	FL = Fixture{
		State: "FL", PayerID: "FL-MCO-001",
		ClaimID:     uuid.MustParse("00000000-0000-4000-8000-000000000002"),
		ClaimNumber: "CLM-DEMO-FL-001", MedicaidID: "SYN-FL-00001",
		AgencyName:        "Demo Home Care FL",
		ConfigPath:        "docs/fixtures/payer_config_837p_fl.json",
		GoldenPath:        "docs/fixtures/837p_fl_golden.x12",
		ExtraValidationID: "fl_units_positive",
	}
	OH = Fixture{
		State: "OH", PayerID: "OH-MCO-001",
		ClaimID:     uuid.MustParse("00000000-0000-4000-8000-000000000003"),
		ClaimNumber: "CLM-DEMO-OH-001", MedicaidID: "SYN-OH-00001",
		AgencyName:        "Demo Home Care OH",
		ConfigPath:        "docs/fixtures/payer_config_837p_oh.json",
		GoldenPath:        "docs/fixtures/837p_oh_golden.x12",
		ExtraValidationID: "oh_procedure_required",
	}
	PA = Fixture{
		State: "PA", PayerID: "PA-MCO-001",
		ClaimID:     uuid.MustParse("00000000-0000-4000-8000-000000000004"),
		ClaimNumber: "CLM-DEMO-PA-001", MedicaidID: "SYN-PA-00001",
		AgencyName:        "Demo Home Care PA",
		ConfigPath:        "docs/fixtures/payer_config_837p_pa.json",
		GoldenPath:        "docs/fixtures/837p_pa_golden.x12",
		ExtraValidationID: "pa_medicaid_id_required",
	}
	NY = Fixture{
		State: "NY", PayerID: "NY-MCO-001",
		ClaimID:     uuid.MustParse("00000000-0000-4000-8000-000000000005"),
		ClaimNumber: "CLM-DEMO-NY-001", MedicaidID: "SYN-NY-00001",
		AgencyName:        "Demo Home Care NY",
		ConfigPath:        "docs/fixtures/payer_config_837p_ny.json",
		GoldenPath:        "docs/fixtures/837p_ny_golden.x12",
		ExtraValidationID: "ny_agency_state",
	}
	All = []Fixture{FL, OH, PA, NY}
)
