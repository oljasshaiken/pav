package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/pavillio/pav-edi/pkg/x12"
)

var requiredSegmentTags = []string{"BHT", "NM1", "CLM", "HI", "SV1", "REF"}

// PostValidateDocument checks structural requirements on generated X12.
func PostValidateDocument(_ context.Context, doc x12.Document) error {
	if strings.TrimSpace(doc.Raw) == "" {
		return fmt.Errorf("%w: empty edi document", ErrValidationFailed)
	}
	if !strings.HasPrefix(doc.Raw, "ISA") {
		return fmt.Errorf("%w: edi must start with ISA envelope", ErrValidationFailed)
	}
	manifest, err := x12.Normalize(doc.Raw)
	if err != nil {
		return fmt.Errorf("%w: parse edi: %v", ErrValidationFailed, err)
	}
	seen := map[string]bool{}
	for _, seg := range manifest.Segments {
		seen[seg.Segment] = true
	}
	for _, tag := range requiredSegmentTags {
		if !seen[tag] {
			return fmt.Errorf("%w: missing required segment %s", ErrValidationFailed, tag)
		}
	}
	return nil
}
