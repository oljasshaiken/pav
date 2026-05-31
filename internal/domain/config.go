package domain

import "encoding/json"

type PayerConfigBody struct {
	X12Version      string          `json:"x12_version"`
	Envelope        json.RawMessage `json:"envelope"`
	Mappings        json.RawMessage `json:"mappings"`
	ValidationRules json.RawMessage `json:"validation_rules"`
	BusinessRules   json.RawMessage `json:"business_rules"`
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
