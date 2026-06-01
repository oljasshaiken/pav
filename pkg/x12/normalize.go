package x12

import (
	"strings"
)

// DynamicPlaceholder replaces envelope fields ignored during integration comparison.
const DynamicPlaceholder = "*"

// Manifest is a normalized segment list for golden / parity comparison.
type Manifest struct {
	Segments []ManifestSegment `json:"segments"`
}

// ManifestSegment is one segment with loop context (matches 837p_tx_golden.normalized.json).
type ManifestSegment struct {
	Loop     string   `json:"loop"`
	Segment  string   `json:"segment"`
	Elements []string `json:"elements"`
}

// Normalize parses X12 into a segment manifest aligned with golden normalized fixtures.
func Normalize(raw string) (Manifest, error) {
	segs, err := parseSegments(raw)
	if err != nil {
		return Manifest{}, err
	}
	out := make([]ManifestSegment, 0, len(segs))
	loop := "envelope"
	for _, seg := range segs {
		loop = nextLoop(loop, seg)
		out = append(out, ManifestSegment{
			Loop:     loop,
			Segment:  seg.Tag,
			Elements: canonicalElements(seg.Tag, seg.Elements),
		})
	}
	return Manifest{Segments: out}, nil
}

// StripDynamicFields blanks envelope indices documented below for integration compares.
//
// Dynamic element indices (0-based element positions after segment tag):
//   - ISA: 8 date, 9 time, 12 interchange control number
//   - GS:  3 date, 4 time, 5 group control number
//   - ST:  1 transaction set control number
//   - BHT: 3 date, 4 time
//   - SE:  1 control number (matches ST)
//   - GE:  1 group control number (matches GS)
//   - IEA: 1 interchange control number (matches ISA)
func StripDynamicFields(m *Manifest) {
	for i := range m.Segments {
		seg := &m.Segments[i]
		switch seg.Segment {
		case "ISA":
			setIfInBounds(seg.Elements, 8, DynamicPlaceholder)
			setIfInBounds(seg.Elements, 9, DynamicPlaceholder)
			setIfInBounds(seg.Elements, 12, DynamicPlaceholder)
		case "GS":
			setIfInBounds(seg.Elements, 3, DynamicPlaceholder)
			setIfInBounds(seg.Elements, 4, DynamicPlaceholder)
			setIfInBounds(seg.Elements, 5, DynamicPlaceholder)
		case "ST":
			setIfInBounds(seg.Elements, 1, DynamicPlaceholder)
		case "BHT":
			setIfInBounds(seg.Elements, 3, DynamicPlaceholder)
			setIfInBounds(seg.Elements, 4, DynamicPlaceholder)
		case "SE":
			setIfInBounds(seg.Elements, 1, DynamicPlaceholder)
		case "GE":
			setIfInBounds(seg.Elements, 1, DynamicPlaceholder)
		case "IEA":
			setIfInBounds(seg.Elements, 1, DynamicPlaceholder)
		}
	}
}

func setIfInBounds(elems []string, idx int, value string) {
	if idx >= 0 && idx < len(elems) {
		elems[idx] = value
	}
}

func parseSegments(raw string) ([]Segment, error) {
	raw = strings.ReplaceAll(raw, "\n", "")
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, "~")
	out := make([]Segment, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		fields := strings.Split(part, "*")
		if len(fields) == 0 {
			continue
		}
		out = append(out, Segment{
			Tag:      fields[0],
			Elements: fields[1:],
		})
	}
	return out, nil
}

func nextLoop(current string, seg Segment) string {
	switch seg.Tag {
	case "ISA", "GS", "ST", "SE", "GE", "IEA":
		return "envelope"
	case "BHT":
		return "header"
	case "HL":
		if len(seg.Elements) >= 3 && seg.Elements[2] == "20" {
			return "2000A"
		}
		if len(seg.Elements) >= 3 && seg.Elements[2] == "22" {
			return "2000B"
		}
		return current
	case "NM1":
		if len(seg.Elements) > 0 && seg.Elements[0] == "IL" {
			return "2010BA"
		}
		if current == "2000B" || (len(seg.Elements) > 0 && seg.Elements[0] == "IL") {
			return "2010BA"
		}
		return "2000A"
	case "CLM", "HI":
		return "2300"
	case "LX", "SV1", "REF":
		return "2400"
	default:
		return current
	}
}

func canonicalElements(tag string, elements []string) []string {
	switch tag {
	case "ISA":
		return canonicalISA(elements)
	case "NM1":
		return canonicalNM1(elements)
	default:
		return trimElements(elements)
	}
}

func canonicalISA(elements []string) []string {
	out := trimElements(elements)
	for i := range out {
		if strings.TrimSpace(out[i]) == "" {
			out[i] = ""
		}
	}
	return out
}

func canonicalNM1(elements []string) []string {
	trimmed := trimElements(elements)
	if len(trimmed) == 0 {
		return trimmed
	}
	qual, id := nm1QualifierID(trimmed)
	switch trimmed[0] {
	case "85":
		name := ""
		if len(trimmed) > 2 {
			name = trimmed[2]
		}
		typ := ""
		if len(trimmed) > 1 {
			typ = trimmed[1]
		}
		return []string{trimmed[0], typ, name, "", "", "", qual, id}
	case "IL":
		last, first := "", ""
		if len(trimmed) > 2 {
			last = trimmed[2]
		}
		if len(trimmed) > 3 {
			first = trimmed[3]
		}
		return []string{trimmed[0], trimmed[1], last, first, "", "", qual, id}
	default:
		return trimmed
	}
}

func nm1QualifierID(elements []string) (qual, id string) {
	for i := len(elements) - 1; i >= 0; i-- {
		if elements[i] == "" {
			continue
		}
		if id == "" {
			id = elements[i]
			continue
		}
		qual = elements[i]
		return qual, id
	}
	return "", id
}

func trimElements(elements []string) []string {
	out := make([]string, len(elements))
	for i, el := range elements {
		out[i] = strings.TrimRight(el, " ")
	}
	return out
}
