package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/rules"
)

type stateSpec struct {
	code, payer, claimNum, medicaid, agency, claimUUID string
}

func main() {
	states := []stateSpec{
		{"fl", "FL-MCO-001", "CLM-DEMO-FL-001", "SYN-FL-00001", "Demo Home Care FL", "00000000-0000-4000-8000-000000000002"},
		{"oh", "OH-MCO-001", "CLM-DEMO-OH-001", "SYN-OH-00001", "Demo Home Care OH", "00000000-0000-4000-8000-000000000003"},
		{"pa", "PA-MCO-001", "CLM-DEMO-PA-001", "SYN-PA-00001", "Demo Home Care PA", "00000000-0000-4000-8000-000000000004"},
		{"ny", "NY-MCO-001", "CLM-DEMO-NY-001", "SYN-NY-00001", "Demo Home Care NY", "00000000-0000-4000-8000-000000000005"},
		{"ca", "CA-MCO-001", "CLM-DEMO-CA-001", "SYN-CA-00001", "Demo Home Care CA", "00000000-0000-4000-8000-000000000006"},
		{"il", "IL-MCO-001", "CLM-DEMO-IL-001", "SYN-IL-00001", "Demo Home Care IL", "00000000-0000-4000-8000-000000000007"},
		{"ga", "GA-MCO-001", "CLM-DEMO-GA-001", "SYN-GA-00001", "Demo Home Care GA", "00000000-0000-4000-8000-000000000008"},
		{"mi", "MI-MCO-001", "CLM-DEMO-MI-001", "SYN-MI-00001", "Demo Home Care MI", "00000000-0000-4000-8000-000000000009"},
		{"nj", "NJ-MCO-001", "CLM-DEMO-NJ-001", "SYN-NJ-00001", "Demo Home Care NJ", "00000000-0000-4000-8000-000000000010"},
	}
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	for _, s := range states {
		data, err := os.ReadFile(fmt.Sprintf("docs/fixtures/payer_config_837p_%s.json", s.code))
		if err != nil {
			panic(err)
		}
		var cfg domain.PayerConfigBody
		if err := json.Unmarshal(data, &cfg); err != nil {
			panic(err)
		}
		claimNumber := s.claimNum
		amount := 100.00
		clockIn := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
		ctx := domain.ClaimContext{
			Claim: domain.Claim{
				ID: uuid.MustParse(s.claimUUID), PayerID: s.payer, State: strings.ToUpper(s.code), ClaimNumber: &claimNumber,
			},
			Authorization: domain.Authorization{ServiceType: "home_health"},
			Patient:       domain.Patient{FirstName: "Synthetic", LastName: "Patient", MedicaidID: s.medicaid},
			Agency:        domain.Agency{Name: s.agency, NPI: "1234567890", State: strings.ToUpper(s.code)},
			Visits:        []domain.Visit{{ClockInTime: &clockIn, EVVStatus: "VERIFIED"}},
			ServiceLines: []domain.ClaimServiceLine{{
				ProcedureCode: "T1019", Units: 4, Amount: &amount, DiagnosisCodes: []string{"Z9999"},
			}},
		}
		doc, err := rules.Transform837P(cfg, ctx, 1, now)
		if err != nil {
			panic(err)
		}
		out := fmt.Sprintf("docs/fixtures/837p_%s_golden.x12", s.code)
		if err := os.WriteFile(out, []byte(doc.Raw+"\n"), 0o644); err != nil {
			panic(err)
		}
		fmt.Println("wrote", out)
	}
}
