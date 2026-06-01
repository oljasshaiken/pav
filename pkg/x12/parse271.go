package x12

import "fmt"

// Eligibility271 is a parsed inbound 271 eligibility response.
type Eligibility271 struct {
	InquiryRef     string
	MedicaidID     string
	PayerID        string
	CoverageStatus string
	ServiceType    string
	Raw            string
}

// ParseEligibility271 parses a 271 X12 payload and extracts inquiry reference and coverage.
func ParseEligibility271(raw string) (Eligibility271, error) {
	segs, err := SplitSegments(raw)
	if err != nil {
		return Eligibility271{}, err
	}

	txSet := ""
	for _, seg := range segs {
		if seg.Tag == "ST" && len(seg.Elements) > 0 {
			txSet = seg.Elements[0]
			break
		}
	}
	if txSet != "271" {
		return Eligibility271{}, fmt.Errorf("unsupported transaction set %q (expected 271)", txSet)
	}

	out := Eligibility271{
		InquiryRef:     extract271InquiryRef(segs),
		MedicaidID:     extractSubscriberMedicaidID(segs),
		PayerID:        extractPayerID(segs),
		CoverageStatus: extractCoverageStatus(segs),
		ServiceType:    extractServiceType(segs),
		Raw:            raw,
	}
	if out.InquiryRef == "" && out.MedicaidID == "" {
		return Eligibility271{}, fmt.Errorf("271 missing inquiry reference and subscriber id")
	}
	return out, nil
}

func extract271InquiryRef(segs []Segment) string {
	for _, seg := range segs {
		if seg.Tag != "BHT" || len(seg.Elements) < 3 {
			continue
		}
		if seg.Elements[0] == "0022" {
			return seg.Elements[2]
		}
	}
	return ""
}

func extractSubscriberMedicaidID(segs []Segment) string {
	for _, seg := range segs {
		if seg.Tag != "NM1" || len(seg.Elements) < 9 {
			continue
		}
		if seg.Elements[0] != "IL" {
			continue
		}
		if seg.Elements[7] == "MI" {
			return seg.Elements[8]
		}
	}
	return ""
}

func extractPayerID(segs []Segment) string {
	for _, seg := range segs {
		if seg.Tag != "NM1" || len(seg.Elements) < 9 {
			continue
		}
		if seg.Elements[0] != "PR" {
			continue
		}
		if seg.Elements[7] == "PI" {
			return seg.Elements[8]
		}
	}
	return ""
}

func extractCoverageStatus(segs []Segment) string {
	for _, seg := range segs {
		if seg.Tag != "EB" || len(seg.Elements) == 0 {
			continue
		}
		switch seg.Elements[0] {
		case "1":
			return "ACTIVE"
		case "6":
			return "INACTIVE"
		default:
			return "UNKNOWN"
		}
	}
	return "UNKNOWN"
}

func extractServiceType(segs []Segment) string {
	for _, seg := range segs {
		if seg.Tag != "EB" || len(seg.Elements) < 3 {
			continue
		}
		if seg.Elements[2] != "" {
			return seg.Elements[2]
		}
	}
	return ""
}
