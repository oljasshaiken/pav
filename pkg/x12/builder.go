package x12

import "strings"

// Builder assembles envelope headers, body segments, and trailers into one interchange.
type Builder struct {
	envelope EnvelopeConfig
	opts     FixedClockOptions
	sep      Separators
	body     []Segment
}

// NewBuilder creates a builder with fixed clock/control options for deterministic output.
func NewBuilder(envelope EnvelopeConfig, opts FixedClockOptions, sep Separators) *Builder {
	return &Builder{
		envelope: envelope,
		opts:     opts,
		sep:      sep,
		body:     nil,
	}
}

// AppendBody adds transaction segments (between ST and SE).
func (b *Builder) AppendBody(segs ...Segment) {
	b.body = append(b.body, segs...)
}

// Build renders ISA/GS/ST, body, SE/GE/IEA. SE segment count includes ST through SE inclusive.
func (b *Builder) Build() string {
	const x12Version = "005010X222A1"
	headers := []Segment{
		BuildISA(b.envelope.ISA, b.opts),
		BuildGS(b.envelope.GS, b.opts),
		BuildST(b.envelope.ST, x12Version, b.opts),
	}
	trailers := []Segment{
		BuildSE(b.seSegmentCount(), b.opts),
		BuildGE(b.opts),
		BuildIEA(b.opts),
	}
	all := make([]Segment, 0, len(headers)+len(b.body)+len(trailers))
	all = append(all, headers...)
	all = append(all, b.body...)
	all = append(all, trailers...)
	return b.render(all)
}

// BuildEnvelopePlusNM1 is a minimal strict-match helper: ISA/GS/ST + one NM1 (no trailers).
func (b *Builder) BuildEnvelopePlusNM1() string {
	const x12Version = "005010X222A1"
	headers := []Segment{
		BuildISA(b.envelope.ISA, b.opts),
		BuildGS(b.envelope.GS, b.opts),
		BuildST(b.envelope.ST, x12Version, b.opts),
	}
	all := append(headers, b.body...)
	return b.render(all)
}

func (b *Builder) seSegmentCount() int {
	if b.opts.SESegmentCount > 0 {
		return b.opts.SESegmentCount
	}
	// ST + body + SE
	return 1 + len(b.body) + 1
}

func (b *Builder) render(segs []Segment) string {
	var sb strings.Builder
	for _, seg := range segs {
		sb.WriteString(seg.Serialize(b.sep))
	}
	return sb.String()
}
