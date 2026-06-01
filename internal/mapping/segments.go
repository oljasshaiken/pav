package mapping

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/pkg/x12"
)

type configMappings struct {
	Patient struct {
		Loop2010BA map[string]string `json:"loop_2010BA"`
	} `json:"patient"`
	Agency struct {
		Loop2000A map[string]string `json:"loop_2000A"`
	} `json:"agency"`
	Claim struct {
		Loop2300 map[string]string `json:"loop_2300"`
	} `json:"claim"`
	ServiceLine struct {
		Loop2400 map[string]string `json:"loop_2400"`
	} `json:"service_line"`
	EVV struct {
		CustomRefSegment struct {
			Qualifier string `json:"qualifier"`
			ValuePath string `json:"value_path"`
		} `json:"custom_ref_segment"`
		LXLoopRequired bool `json:"lx_loop_required"`
	} `json:"evv"`
}

// BuildBodySegments assembles 837P body segments (BHT through REF) from payer
// config mappings and claim context. BHT date/time come from clock (GSDate/GSTime).
func BuildBodySegments(mappingsJSON json.RawMessage, ctx domain.ClaimContext, clock x12.FixedClockOptions) ([]x12.Segment, error) {
	var mappings configMappings
	if err := json.Unmarshal(mappingsJSON, &mappings); err != nil {
		return nil, fmt.Errorf("parse mappings: %w", err)
	}

	bhtClaim, err := ResolvePath(ctx, "claim.claim_number")
	if err != nil {
		return nil, err
	}

	agencyNM1, err := buildNM1(ctx, "85", mappings.Agency.Loop2000A)
	if err != nil {
		return nil, fmt.Errorf("agency NM1: %w", err)
	}
	patientNM1, err := buildNM1(ctx, "IL", mappings.Patient.Loop2010BA)
	if err != nil {
		return nil, fmt.Errorf("patient NM1: %w", err)
	}
	clm, err := buildCLM(ctx, mappings.Claim.Loop2300)
	if err != nil {
		return nil, fmt.Errorf("CLM: %w", err)
	}
	hi, err := buildHI(ctx, mappings.Claim.Loop2300)
	if err != nil {
		return nil, fmt.Errorf("HI: %w", err)
	}
	sv1, err := buildSV1(ctx, mappings.ServiceLine.Loop2400)
	if err != nil {
		return nil, fmt.Errorf("SV1: %w", err)
	}
	ref, err := buildEVVRef(ctx, mappings.EVV.CustomRefSegment)
	if err != nil {
		return nil, fmt.Errorf("REF EVV: %w", err)
	}

	segs := []x12.Segment{
		{Tag: "BHT", Elements: []string{"0019", "00", bhtClaim, clock.GSDate, clock.GSTime, "CH"}},
		{Tag: "HL", Elements: []string{"1", "", "20", "1"}},
		agencyNM1,
		{Tag: "HL", Elements: []string{"2", "1", "22", "0"}},
		patientNM1,
		clm,
		hi,
	}
	if mappings.EVV.LXLoopRequired {
		segs = append(segs, x12.Segment{Tag: "LX", Elements: []string{"1"}})
	}
	segs = append(segs, sv1, ref)
	return segs, nil
}

func buildNM1(ctx domain.ClaimContext, entityID string, loop map[string]string) (x12.Segment, error) {
	keys := []string{"NM102", "NM103", "NM104", "NM105", "NM106", "NM107", "NM108", "NM109"}
	elements := make([]string, 0, 9)
	elements = append(elements, entityID)
	for _, key := range keys {
		raw, ok := loop[key]
		if !ok {
			elements = append(elements, "")
			continue
		}
		val, err := resolveMappingValue(ctx, raw)
		if err != nil {
			return x12.Segment{}, fmt.Errorf("%s: %w", key, err)
		}
		elements = append(elements, val)
	}
	return x12.Segment{Tag: "NM1", Elements: elements}, nil
}

func buildCLM(ctx domain.ClaimContext, loop map[string]string) (x12.Segment, error) {
	clm01, err := resolveMappingValue(ctx, loop["CLM01"])
	if err != nil {
		return x12.Segment{}, err
	}
	clm02, err := resolveMappingValue(ctx, loop["CLM02"])
	if err != nil {
		return x12.Segment{}, err
	}
	return x12.Segment{
		Tag:      "CLM",
		Elements: []string{clm01, clm02, "", "", "11:B:1", "Y", "A", "Y", "Y"},
	}, nil
}

func buildHI(ctx domain.ClaimContext, loop map[string]string) (x12.Segment, error) {
	code, err := resolveMappingValue(ctx, loop["HI01"])
	if err != nil {
		return x12.Segment{}, err
	}
	return x12.Segment{Tag: "HI", Elements: []string{"ABK:" + code}}, nil
}

func buildSV1(ctx domain.ClaimContext, loop map[string]string) (x12.Segment, error) {
	proc, err := resolveMappingValue(ctx, loop["SV101"])
	if err != nil {
		return x12.Segment{}, err
	}
	amount, err := resolveMappingValue(ctx, loop["SV102"])
	if err != nil {
		return x12.Segment{}, err
	}
	unitQual, err := resolveMappingValue(ctx, loop["SV103"])
	if err != nil {
		return x12.Segment{}, err
	}
	units, err := resolveMappingValue(ctx, loop["SV104"])
	if err != nil {
		return x12.Segment{}, err
	}
	return x12.Segment{
		Tag:      "SV1",
		Elements: []string{"HC:" + proc, amount, unitQual, units, "", "", "1"},
	}, nil
}

func buildEVVRef(ctx domain.ClaimContext, ref struct {
	Qualifier string `json:"qualifier"`
	ValuePath string `json:"value_path"`
}) (x12.Segment, error) {
	val, err := ResolvePath(ctx, ref.ValuePath)
	if err != nil {
		return x12.Segment{}, err
	}
	return x12.Segment{Tag: "REF", Elements: []string{ref.Qualifier, val}}, nil
}

func resolveMappingValue(ctx domain.ClaimContext, raw string) (string, error) {
	if strings.Contains(raw, ".") {
		return ResolvePath(ctx, raw)
	}
	return raw, nil
}
