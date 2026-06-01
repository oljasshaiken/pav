package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pavillio/pav-edi/internal/domain"
)

type claimRule struct {
	Field     string `json:"field"`
	Rule      string `json:"rule"`
	Condition string `json:"condition"`
}

// PreValidateClaim evaluates validation_rules against claim context before EDI generation.
func PreValidateClaim(_ context.Context, claim domain.ClaimContext, rulesJSON json.RawMessage) error {
	if len(rulesJSON) == 0 || string(rulesJSON) == "null" {
		return nil
	}
	var rules []claimRule
	if err := json.Unmarshal(rulesJSON, &rules); err != nil {
		return fmt.Errorf("parse validation rules: %w", err)
	}
	for _, rule := range rules {
		if rule.Rule != "required" {
			continue
		}
		if !conditionMatches(rule.Condition, claim.Authorization.ServiceType) {
			continue
		}
		if err := checkRequired(rule.Field, claim); err != nil {
			return err
		}
	}
	return nil
}

func conditionMatches(condition, serviceType string) bool {
	if condition == "" {
		return true
	}
	parts := strings.SplitN(condition, "=", 2)
	if len(parts) != 2 {
		return true
	}
	key := strings.TrimSpace(parts[0])
	val := strings.Trim(strings.TrimSpace(parts[1]), "'")
	if key != "service_type" {
		return true
	}
	return serviceType == val
}

func checkRequired(field string, claim domain.ClaimContext) error {
	switch field {
	case "diagnosis_code":
		if len(claim.ServiceLines) == 0 || len(claim.ServiceLines[0].DiagnosisCodes) == 0 {
			return fmt.Errorf("%w: diagnosis_code required", ErrValidationFailed)
		}
	default:
		return fmt.Errorf("%w: unsupported validation field %q", ErrValidationFailed, field)
	}
	return nil
}
