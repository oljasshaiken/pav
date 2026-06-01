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
	CA = Fixture{
		State: "CA", PayerID: "CA-MCO-001",
		ClaimID:     uuid.MustParse("00000000-0000-4000-8000-000000000006"),
		ClaimNumber: "CLM-DEMO-CA-001", MedicaidID: "SYN-CA-00001",
		AgencyName:        "Demo Home Care CA",
		ConfigPath:        "docs/fixtures/payer_config_837p_ca.json",
		GoldenPath:        "docs/fixtures/837p_ca_golden.x12",
		ExtraValidationID: "ca_units_cap",
	}
	IL = Fixture{
		State: "IL", PayerID: "IL-MCO-001",
		ClaimID:     uuid.MustParse("00000000-0000-4000-8000-000000000007"),
		ClaimNumber: "CLM-DEMO-IL-001", MedicaidID: "SYN-IL-00001",
		AgencyName:        "Demo Home Care IL",
		ConfigPath:        "docs/fixtures/payer_config_837p_il.json",
		GoldenPath:        "docs/fixtures/837p_il_golden.x12",
		ExtraValidationID: "il_agency_state",
	}
	GA = Fixture{
		State: "GA", PayerID: "GA-MCO-001",
		ClaimID:     uuid.MustParse("00000000-0000-4000-8000-000000000008"),
		ClaimNumber: "CLM-DEMO-GA-001", MedicaidID: "SYN-GA-00001",
		AgencyName:        "Demo Home Care GA",
		ConfigPath:        "docs/fixtures/payer_config_837p_ga.json",
		GoldenPath:        "docs/fixtures/837p_ga_golden.x12",
		ExtraValidationID: "ga_claim_state",
	}
	MI = Fixture{
		State: "MI", PayerID: "MI-MCO-001",
		ClaimID:     uuid.MustParse("00000000-0000-4000-8000-000000000009"),
		ClaimNumber: "CLM-DEMO-MI-001", MedicaidID: "SYN-MI-00001",
		AgencyName:        "Demo Home Care MI",
		ConfigPath:        "docs/fixtures/payer_config_837p_mi.json",
		GoldenPath:        "docs/fixtures/837p_mi_golden.x12",
		ExtraValidationID: "mi_payer_match",
	}
	NJ = Fixture{
		State: "NJ", PayerID: "NJ-MCO-001",
		ClaimID:     uuid.MustParse("00000000-0000-4000-8000-000000000010"),
		ClaimNumber: "CLM-DEMO-NJ-001", MedicaidID: "SYN-NJ-00001",
		AgencyName:        "Demo Home Care NJ",
		ConfigPath:        "docs/fixtures/payer_config_837p_nj.json",
		GoldenPath:        "docs/fixtures/837p_nj_golden.x12",
		ExtraValidationID: "nj_medicaid_prefix",
	}
	All = []Fixture{FL, OH, PA, NY, CA, IL, GA, MI, NJ}
)
