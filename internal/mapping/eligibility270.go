package mapping

import (
	"encoding/json"
	"fmt"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/pkg/x12"
)

type eligibility270Mappings struct {
	BHT struct {
		Reference string `json:"reference"`
	} `json:"bht"`
	Payer struct {
		Loop2100A map[string]string `json:"loop_2100A"`
	} `json:"payer"`
	Provider struct {
		Loop2100B map[string]string `json:"loop_2100B"`
	} `json:"provider"`
	Subscriber struct {
		Loop2100C map[string]string `json:"loop_2100C"`
		DMG02       string            `json:"DMG02"`
		DTP03       string            `json:"DTP03"`
		EQ01        string            `json:"EQ01"`
	} `json:"subscriber"`
}

// Build270BodySegments assembles 270 eligibility inquiry body segments from payer
// config mappings and claim context. BHT/DTP dates come from clock (GSDate/GSTime).
func Build270BodySegments(mappingsJSON json.RawMessage, ctx domain.ClaimContext, clock x12.FixedClockOptions) ([]x12.Segment, error) {
	var mappings eligibility270Mappings
	if err := json.Unmarshal(mappingsJSON, &mappings); err != nil {
		return nil, fmt.Errorf("parse 270 mappings: %w", err)
	}

	ref, err := resolveMappingValue(ctx, mappings.BHT.Reference)
	if err != nil {
		return nil, fmt.Errorf("BHT reference: %w", err)
	}

	payerNM1, err := buildNM1(ctx, "PR", mappings.Payer.Loop2100A)
	if err != nil {
		return nil, fmt.Errorf("payer NM1: %w", err)
	}
	providerNM1, err := buildNM1(ctx, "1P", mappings.Provider.Loop2100B)
	if err != nil {
		return nil, fmt.Errorf("provider NM1: %w", err)
	}
	subscriberNM1, err := buildNM1(ctx, "IL", mappings.Subscriber.Loop2100C)
	if err != nil {
		return nil, fmt.Errorf("subscriber NM1: %w", err)
	}

	dmgDOB, err := resolveMappingValue(ctx, mappings.Subscriber.DMG02)
	if err != nil {
		return nil, fmt.Errorf("DMG date of birth: %w", err)
	}
	dtpDate, err := resolveEligibilityDate(mappings.Subscriber.DTP03, ctx, clock)
	if err != nil {
		return nil, fmt.Errorf("DTP eligibility date: %w", err)
	}
	eqCode, err := resolveMappingValue(ctx, mappings.Subscriber.EQ01)
	if err != nil {
		return nil, fmt.Errorf("EQ service type: %w", err)
	}

	return []x12.Segment{
		{Tag: "BHT", Elements: []string{"0022", "13", ref, clock.GSDate, clock.GSTime}},
		{Tag: "HL", Elements: []string{"1", "", "20", "1"}},
		payerNM1,
		{Tag: "HL", Elements: []string{"2", "1", "21", "1"}},
		providerNM1,
		{Tag: "HL", Elements: []string{"3", "2", "22", "0"}},
		subscriberNM1,
		{Tag: "DMG", Elements: []string{"D8", dmgDOB}},
		{Tag: "DTP", Elements: []string{"291", "D8", dtpDate}},
		{Tag: "EQ", Elements: []string{eqCode}},
	}, nil
}

func resolveEligibilityDate(raw string, ctx domain.ClaimContext, clock x12.FixedClockOptions) (string, error) {
	if raw == "@gs_date" {
		return clock.GSDate, nil
	}
	return resolveMappingValue(ctx, raw)
}
