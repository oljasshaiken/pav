package x12

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Acknowledgment is a parsed inbound 277 or 999 transaction set.
type Acknowledgment struct {
	TransactionSet string
	ClaimRef       string
	Raw            string
}

// ParseAcknowledgment parses a 277 or 999 X12 payload and extracts a claim reference when present.
func ParseAcknowledgment(raw string) (Acknowledgment, error) {
	if strings.TrimSpace(raw) == "" {
		return Acknowledgment{}, fmt.Errorf("empty x12 payload")
	}

	segs, err := SplitSegments(raw)
	if err != nil {
		return Acknowledgment{}, err
	}

	txSet := ""
	for _, seg := range segs {
		if seg.Tag != "ST" || len(seg.Elements) == 0 {
			continue
		}
		txSet = seg.Elements[0]
		break
	}
	if txSet != "277" && txSet != "999" {
		return Acknowledgment{}, fmt.Errorf("unsupported transaction set %q (expected 277 or 999)", txSet)
	}

	ref := extractClaimRef(segs)
	return Acknowledgment{
		TransactionSet: txSet,
		ClaimRef:       ref,
		Raw:            raw,
	}, nil
}

// SplitSegments splits raw X12 into segments using default ~ terminator and * element separator.
func SplitSegments(raw string) ([]Segment, error) {
	sep := Separators{}
	term := sep.segment()
	parts := strings.Split(strings.TrimSpace(raw), term)
	var segs []Segment
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		fields := strings.Split(part, sep.element())
		if len(fields) == 0 || fields[0] == "" {
			continue
		}
		segs = append(segs, Segment{
			Tag:      fields[0],
			Elements: fields[1:],
		})
	}
	if len(segs) == 0 {
		return nil, fmt.Errorf("no segments found")
	}
	return segs, nil
}

func extractClaimRef(segs []Segment) string {
	for _, seg := range segs {
		if seg.Tag == "REF" && len(seg.Elements) >= 2 && seg.Elements[0] == "D9" {
			return seg.Elements[1]
		}
	}
	for _, seg := range segs {
		if seg.Tag == "TRN" && len(seg.Elements) >= 2 {
			if _, err := uuid.Parse(seg.Elements[1]); err == nil {
				return seg.Elements[1]
			}
		}
	}
	for _, seg := range segs {
		for _, el := range seg.Elements {
			if _, err := uuid.Parse(el); err == nil {
				return el
			}
		}
	}
	return ""
}
