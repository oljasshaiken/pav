package edi

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/mapping"
	"github.com/pavillio/pav-edi/pkg/x12"
)

const x12Version = "005010X222A1"

// Generate837P builds a full 837P interchange from envelope config, mappings JSON, and claim context.
func Generate837P(envelopeJSON json.RawMessage, mappingsJSON json.RawMessage, ctx domain.ClaimContext, now time.Time) (string, error) {
	var envelope x12.EnvelopeConfig
	if err := json.Unmarshal(envelopeJSON, &envelope); err != nil {
		return "", fmt.Errorf("parse envelope: %w", err)
	}

	clock := ClockFromTime(now, envelope)
	body, err := mapping.BuildBodySegments(mappingsJSON, ctx, clock)
	if err != nil {
		return "", fmt.Errorf("build body segments: %w", err)
	}

	b := x12.NewBuilder(envelope, clock, x12.Separators{})
	b.AppendBody(body...)
	return b.Build(), nil
}

// ClockFromTime derives fixed clock fields from wall time and envelope control numbers.
func ClockFromTime(now time.Time, envelope x12.EnvelopeConfig) x12.FixedClockOptions {
	utc := now.UTC()
	return x12.FixedClockOptions{
		ISADate:        utc.Format("060102"),
		ISATime:        utc.Format("1504"),
		GSDate:         utc.Format("20060102"),
		GSTime:         utc.Format("1504"),
		ISAControl:     envelope.ISA.InterchangeControlNumber,
		GSControl:      envelope.GS.GroupControlNumber,
		STControl:      envelope.ST.ControlNumber,
		SESegmentCount: 13,
	}
}
