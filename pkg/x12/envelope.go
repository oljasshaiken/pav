package x12

import "strings"

// EnvelopeConfig is the payer config "envelope" block (ISA/GS/ST).
type EnvelopeConfig struct {
	ISA ISAConfig `json:"isa"`
	GS  GSConfig  `json:"gs"`
	ST  STConfig  `json:"st"`
}

type ISAConfig struct {
	SenderID                 string `json:"sender_id"`
	ReceiverID               string `json:"receiver_id"`
	InterchangeControlNumber string `json:"interchange_control_number"`
	UsageIndicator           string `json:"usage_indicator"`
}

type GSConfig struct {
	FunctionalID        string `json:"functional_id"`
	ApplicationSender   string `json:"application_sender"`
	ApplicationReceiver string `json:"application_receiver"`
	GroupControlNumber  string `json:"group_control_number"`
}

type STConfig struct {
	TransactionSetID string `json:"transaction_set_id"`
	ControlNumber    string `json:"control_number"`
}

// FixedClockOptions pins envelope dates/times and control numbers for deterministic tests.
type FixedClockOptions struct {
	ISADate        string
	ISATime        string
	GSDate         string
	GSTime         string
	ISAControl     string
	GSControl      string
	STControl      string
	ComponentSep   string // ISA16, default ":"
	SESegmentCount int    // 0 = auto (ST + body + SE)
}

func (o FixedClockOptions) componentSep() string {
	if o.ComponentSep == "" {
		return ":"
	}
	return o.ComponentSep
}

func (o FixedClockOptions) stControl(fallback string) string {
	if o.STControl != "" {
		return o.STControl
	}
	return fallback
}

func (o FixedClockOptions) gsControl(fallback string) string {
	if o.GSControl != "" {
		return o.GSControl
	}
	return fallback
}

func (o FixedClockOptions) isaControl(fallback string) string {
	if o.ISAControl != "" {
		return o.ISAControl
	}
	return fallback
}

// BuildISA builds the interchange header segment.
func BuildISA(cfg ISAConfig, opts FixedClockOptions) Segment {
	return Segment{
		Tag: "ISA",
		Elements: []string{
			"00",
			padRight("", 10),
			"00",
			padRight("", 10),
			"ZZ",
			padRight(cfg.SenderID, 15),
			"ZZ",
			padRight(cfg.ReceiverID, 15),
			opts.ISADate,
			opts.ISATime,
			"^",
			"00501",
			opts.isaControl(cfg.InterchangeControlNumber),
			"0",
			cfg.UsageIndicator,
			opts.componentSep(),
		},
	}
}

// BuildGS builds the functional group header.
func BuildGS(cfg GSConfig, opts FixedClockOptions, x12Version string) Segment {
	if x12Version == "" {
		x12Version = DefaultX12Version
	}
	return Segment{
		Tag: "GS",
		Elements: []string{
			cfg.FunctionalID,
			cfg.ApplicationSender,
			cfg.ApplicationReceiver,
			opts.GSDate,
			opts.GSTime,
			opts.gsControl(cfg.GroupControlNumber),
			"X",
			x12Version,
		},
	}
}

// BuildST builds the transaction set header.
func BuildST(cfg STConfig, x12Version string, opts FixedClockOptions) Segment {
	return Segment{
		Tag: "ST",
		Elements: []string{
			cfg.TransactionSetID,
			opts.stControl(cfg.ControlNumber),
			x12Version,
		},
	}
}

// BuildSE builds the transaction set trailer.
func BuildSE(segmentCount int, opts FixedClockOptions) Segment {
	return Segment{
		Tag: "SE",
		Elements: []string{
			itoa(int32(segmentCount)),
			opts.stControl("0001"),
		},
	}
}

// BuildGE builds the functional group trailer.
func BuildGE(opts FixedClockOptions) Segment {
	return Segment{
		Tag: "GE",
		Elements: []string{
			"1",
			opts.gsControl("1"),
		},
	}
}

// BuildIEA builds the interchange trailer.
func BuildIEA(opts FixedClockOptions) Segment {
	return Segment{
		Tag: "IEA",
		Elements: []string{
			"1",
			opts.isaControl("000000001"),
		},
	}
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}
