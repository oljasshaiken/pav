package domain

import (
	"encoding/json"
	"fmt"
)

// CELRule is a config-driven rule evaluated via Common Expression Language.
type CELRule struct {
	ID      string `json:"id"`
	CEL     string `json:"cel"`
	Message string `json:"message"`
	Action  string `json:"action,omitempty"`
}

// LegacyValidationRule is the Phase 1 validation_rules shape (deprecated).
type LegacyValidationRule struct {
	Field     string `json:"field"`
	Rule      string `json:"rule"`
	Condition string `json:"condition"`
}

type PayerConfigBody struct {
	X12Version      string          `json:"x12_version"`
	Envelope        json.RawMessage `json:"envelope"`
	Mappings        json.RawMessage `json:"mappings"`
	EVVRules        json.RawMessage `json:"evv_rules,omitempty"`
	ValidationRules json.RawMessage `json:"validation_rules"`
	BusinessRules   json.RawMessage `json:"business_rules"`
}

// CELValidationRules parses validation_rules as CEL-shaped rules.
func (b PayerConfigBody) CELValidationRules() ([]CELRule, error) {
	return parseCELRules(b.ValidationRules, "validation_rules")
}

// CELEvvRules parses evv_rules as CEL-shaped rules.
func (b PayerConfigBody) CELEvvRules() ([]CELRule, error) {
	return parseCELRules(b.EVVRules, "evv_rules")
}

// CELBusinessRules parses business_rules as CEL-shaped rules (array or empty object).
func (b PayerConfigBody) CELBusinessRules() ([]CELRule, error) {
	if len(b.BusinessRules) == 0 || string(b.BusinessRules) == "{}" || string(b.BusinessRules) == "null" {
		return nil, nil
	}
	return parseCELRules(b.BusinessRules, "business_rules")
}

// ParseLegacyValidationRules parses Phase 1 validation_rules ({field, rule, condition}).
func ParseLegacyValidationRules(raw json.RawMessage) ([]LegacyValidationRule, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var rules []LegacyValidationRule
	if err := json.Unmarshal(raw, &rules); err != nil {
		return nil, fmt.Errorf("parse legacy validation_rules: %w", err)
	}
	return rules, nil
}

func parseCELRules(raw json.RawMessage, field string) ([]CELRule, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var rules []CELRule
	if err := json.Unmarshal(raw, &rules); err != nil {
		return nil, fmt.Errorf("parse %s: %w", field, err)
	}
	for i, rule := range rules {
		if rule.ID == "" {
			return nil, fmt.Errorf("parse %s: rule[%d] missing id", field, i)
		}
		if rule.CEL == "" {
			return nil, fmt.Errorf("parse %s: rule %q missing cel", field, rule.ID)
		}
	}
	return rules, nil
}

type PayerConfig struct {
	ID              string
	State           string
	PayerID         string
	TransactionType string
	ConfigVersion   int32
	Destination     json.RawMessage
	Active          bool
	Config          PayerConfigBody
	UpdatedBy       *string
}

type X12Template struct {
	ID              string
	Name            string
	TransactionType string
	X12Version      string
	Template        json.RawMessage
}

type TemplateOverride struct {
	ID              string
	TemplateID      string
	State           string
	PayerID         string
	OverrideVersion int32
	Mapper          json.RawMessage
	Destination     json.RawMessage
	Template        X12Template
}
