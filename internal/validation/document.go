package validation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/pavillio/pav-edi/internal/cel"
	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/pkg/x12"
)

var requiredSegmentTags = []string{"BHT", "NM1", "CLM", "HI", "SV1", "REF"}

// PostValidateDocument checks structural requirements on generated X12.
func PostValidateDocument(_ context.Context, doc x12.Document) error {
	if err := validateDocumentStructure(doc); err != nil {
		return err
	}
	return nil
}

// PostValidateBusinessRules evaluates CEL business_rules against the normalized segment manifest.
func PostValidateBusinessRules(_ context.Context, doc x12.Document, rulesJSON json.RawMessage) error {
	if len(rulesJSON) == 0 || string(rulesJSON) == "null" || string(rulesJSON) == "{}" {
		return nil
	}
	if err := validateDocumentStructure(doc); err != nil {
		return err
	}
	body := domain.PayerConfigBody{BusinessRules: rulesJSON}
	rules, err := body.CELBusinessRules()
	if err != nil {
		return fmt.Errorf("parse business_rules: %w", err)
	}
	if len(rules) == 0 {
		return nil
	}
	manifest, err := x12.Normalize(doc.Raw)
	if err != nil {
		return fmt.Errorf("%w: parse edi: %v", ErrValidationFailed, err)
	}
	if err := cel.EvaluateManifestAll(rules, cel.ManifestBindings(manifest)); err != nil {
		var ve *cel.ValidationError
		if errors.As(err, &ve) {
			return fmt.Errorf("%w: %s", ErrValidationFailed, ve.Message)
		}
		return fmt.Errorf("cel business rules: %w", err)
	}
	return nil
}

func validateDocumentStructure(doc x12.Document) error {
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
