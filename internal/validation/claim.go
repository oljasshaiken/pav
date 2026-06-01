package validation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/pavillio/pav-edi/internal/cel"
	"github.com/pavillio/pav-edi/internal/domain"
)

type claimRule struct {
	Field     string `json:"field"`
	Rule      string `json:"rule"`
	Condition string `json:"condition"`
}

// PreValidateClaim evaluates validation_rules against claim context before EDI generation.
// Supports CEL-shaped rules (preferred) and legacy {field, rule, condition} objects.
func PreValidateClaim(_ context.Context, claim domain.ClaimContext, rulesJSON json.RawMessage) error {
	if len(rulesJSON) == 0 || string(rulesJSON) == "null" {
		return nil
	}

	body := domain.PayerConfigBody{ValidationRules: rulesJSON}
	celRules, celErr := body.CELValidationRules()
	if celErr == nil && len(celRules) > 0 {
		if err := cel.EvaluateAll(celRules, cel.ClaimBindings(claim)); err != nil {
			var ve *cel.ValidationError
			if errors.As(err, &ve) {
				return fmt.Errorf("%w: %s", ErrValidationFailed, ve.Message)
			}
			return fmt.Errorf("cel validation: %w", err)
		}
		return nil
	}

	return preValidateLegacy(claim, rulesJSON)
}

func preValidateLegacy(claim domain.ClaimContext, rulesJSON json.RawMessage) error {
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

// PreValidateEVV evaluates evv_rules via CEL before EDI generation.
func PreValidateEVV(_ context.Context, claim domain.ClaimContext, evvRulesJSON json.RawMessage) error {
	if len(evvRulesJSON) == 0 || string(evvRulesJSON) == "null" {
		return nil
	}
	body := domain.PayerConfigBody{EVVRules: evvRulesJSON}
	rules, err := body.CELEvvRules()
	if err != nil {
		return fmt.Errorf("parse evv_rules: %w", err)
	}
	if len(rules) == 0 {
		return nil
	}
	if err := cel.EvaluateAll(rules, cel.ClaimBindings(claim)); err != nil {
		var ve *cel.ValidationError
		if errors.As(err, &ve) {
			return fmt.Errorf("%w: %s", ErrValidationFailed, ve.Message)
		}
		return fmt.Errorf("cel evv validation: %w", err)
	}
	return nil
}
