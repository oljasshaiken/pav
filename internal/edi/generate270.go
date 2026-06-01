package edi

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/mapping"
	"github.com/pavillio/pav-edi/pkg/x12"
)

const x12Version270 = "005010X279A1"

// Generate270 builds a full 270 eligibility inquiry interchange.
func Generate270(envelopeJSON json.RawMessage, mappingsJSON json.RawMessage, ctx domain.ClaimContext, now time.Time) (string, error) {
	var envelope x12.EnvelopeConfig
	if err := json.Unmarshal(envelopeJSON, &envelope); err != nil {
		return "", fmt.Errorf("parse envelope: %w", err)
	}

	clock := ClockFromTime270(now, envelope)
	body, err := mapping.Build270BodySegments(mappingsJSON, ctx, clock)
	if err != nil {
		return "", fmt.Errorf("build 270 body segments: %w", err)
	}

	b := x12.NewBuilder(envelope, clock, x12.Separators{}).WithX12Version(x12Version270)
	b.AppendBody(body...)
	return b.Build(), nil
}

// ClockFromTime270 derives fixed clock fields for 270 interchange (11 segments ST through SE).
func ClockFromTime270(now time.Time, envelope x12.EnvelopeConfig) x12.FixedClockOptions {
	clock := ClockFromTime(now, envelope)
	clock.SESegmentCount = 11
	return clock
}
