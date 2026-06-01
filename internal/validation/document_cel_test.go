package validation_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pavillio/pav-edi/internal/validation"
	"github.com/pavillio/pav-edi/pkg/x12"
)

func TestPostValidateBusinessRules_requiresREFSegment(t *testing.T) {
	rules := json.RawMessage(`[{
		"id":"evv_ref_present",
		"cel":"manifest.segment_tags.exists(t, t == \"REF\")",
		"message":"REF segment required",
		"action":"reject"
	}]`)
	doc := x12.Document{Raw: sample837WithoutREF()}
	err := validation.PostValidateBusinessRules(context.Background(), doc, rules)
	if !validation.IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestPostValidateBusinessRules_passesWithREF(t *testing.T) {
	rules := json.RawMessage(`[{
		"id":"evv_ref_present",
		"cel":"manifest.segment_tags.exists(t, t == \"REF\")",
		"message":"REF segment required"
	}]`)
	doc := x12.Document{Raw: sample837WithREF()}
	if err := validation.PostValidateBusinessRules(context.Background(), doc, rules); err != nil {
		t.Fatal(err)
	}
}

func sample837WithoutREF() string {
	return "ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *260531*1200*U*00501*000000001*0*T*:~" +
		"GS*HC*SENDER*RECEIVER*20260531*1200*1*X*005010X222A1~" +
		"ST*837*0001*005010X222A1~" +
		"BHT*0019*00*CLM1*20260531*1200*CH~" +
		"HL*1**20*1~" +
		"NM1*85*2*Agency*****XX*1234567890~" +
		"HL*2*1*22*0~" +
		"NM1*IL*1*Last*First****MI*MED123~" +
		"CLM*CLM1*100***11:B:1*Y*A*Y*Y~" +
		"HI*ABK:Z9999~" +
		"SV1*HC:T1019*100*UN*4***1~" +
		"SE*10*0001~" +
		"GE*1*1~" +
		"IEA*1*000000001~"
}

func sample837WithREF() string {
	raw := sample837WithoutREF()
	// Insert REF before SE
	const insert = "REF*EVV*20260501090000~"
	idx := len(raw) - len("SE*10*0001~GE*1*1~IEA*1*000000001~")
	return raw[:idx] + insert + raw[idx:]
}
