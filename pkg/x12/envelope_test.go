package x12_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/pavillio/pav-edi/pkg/x12"
)

const fixturePayerConfig = "../../docs/fixtures/payer_config_837p_tx.json"

func TestEnvelopeFromConfig_ISA_GS_ST(t *testing.T) {
	cfg := loadEnvelopeConfig(t)
	opts := x12.FixedClockOptions{
		ISADate:      "260531",
		ISATime:      "1200",
		GSDate:       "20260531",
		GSTime:       "1200",
		ISAControl:   "000000001",
		GSControl:    "1",
		STControl:    "0001",
		ComponentSep: ":",
	}

	isa := x12.BuildISA(cfg.ISA, opts)
	wantISA := "ISA*00*          *00*          *ZZ*PAVILLIO       *ZZ*TX_MCO         *260531*1200*^*00501*000000001*0*T*:~"
	if isa.Serialize(x12.Separators{}) != wantISA {
		t.Fatalf("ISA = %q, want %q", isa.Serialize(x12.Separators{}), wantISA)
	}

	gs := x12.BuildGS(cfg.GS, opts, x12.DefaultX12Version)
	wantGS := "GS*HC*PAVILLIO*TX_MCO*20260531*1200*1*X*005010X222A1~"
	if gs.Serialize(x12.Separators{}) != wantGS {
		t.Fatalf("GS = %q, want %q", gs.Serialize(x12.Separators{}), wantGS)
	}

	st := x12.BuildST(cfg.ST, "005010X222A1", opts)
	wantST := "ST*837*0001*005010X222A1~"
	if st.Serialize(x12.Separators{}) != wantST {
		t.Fatalf("ST = %q, want %q", st.Serialize(x12.Separators{}), wantST)
	}
}

func TestEnvelopeTrailers_SE_GE_IEA(t *testing.T) {
	opts := x12.FixedClockOptions{
		ISAControl: "000000001",
		GSControl:  "1",
		STControl:  "0001",
	}

	se := x12.BuildSE(13, opts)
	wantSE := "SE*13*0001~"
	if se.Serialize(x12.Separators{}) != wantSE {
		t.Fatalf("SE = %q, want %q", se.Serialize(x12.Separators{}), wantSE)
	}

	ge := x12.BuildGE(opts)
	wantGE := "GE*1*1~"
	if ge.Serialize(x12.Separators{}) != wantGE {
		t.Fatalf("GE = %q, want %q", ge.Serialize(x12.Separators{}), wantGE)
	}

	iea := x12.BuildIEA(opts)
	wantIEA := "IEA*1*000000001~"
	if iea.Serialize(x12.Separators{}) != wantIEA {
		t.Fatalf("IEA = %q, want %q", iea.Serialize(x12.Separators{}), wantIEA)
	}
}

func loadEnvelopeConfig(t *testing.T) x12.EnvelopeConfig {
	t.Helper()
	data, err := os.ReadFile(fixturePayerConfig)
	if err != nil {
		t.Fatalf("read payer config: %v", err)
	}
	var raw struct {
		Envelope x12.EnvelopeConfig `json:"envelope"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return raw.Envelope
}
