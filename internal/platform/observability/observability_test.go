package observability_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/pavillio/pav-edi/internal/platform/observability"
	"github.com/pavillio/pav-edi/internal/queue"
)

func TestLogDLQAlert_emitsStructuredEvent(t *testing.T) {
	var buf bytes.Buffer
	old := slog.Default()
	t.Cleanup(func() { slog.SetDefault(old) })
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	observability.LogDLQAlert(context.Background(), queue.DLQMessage{
		ClaimID: "claim-1",
		PayerID: "TX-MCO-001",
		State:   "TX",
		Phase:   "pre_transform",
		Code:    "VALIDATION_FAILED",
		Message: "diagnosis required",
		RuleID:  "diagnosis_required",
	})

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output: %v\nraw: %s", err, buf.String())
	}
	if entry["event"] != "dlq_alert" {
		t.Fatalf("event = %v", entry["event"])
	}
	if entry["claim_id"] != "claim-1" {
		t.Fatalf("claim_id = %v", entry["claim_id"])
	}
	if entry["rule_id"] != "diagnosis_required" {
		t.Fatalf("rule_id = %v", entry["rule_id"])
	}
	if !strings.Contains(entry["msg"].(string), "DLQ") {
		t.Fatalf("msg = %v", entry["msg"])
	}
}

func TestLogWorkflowStep_emitsDurationAndStatus(t *testing.T) {
	var buf bytes.Buffer
	old := slog.Default()
	t.Cleanup(func() { slog.SetDefault(old) })
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	ctx := observability.WithWorkflow(context.Background(), observability.WorkflowFields{
		ClaimID: "claim-1",
		PayerID: "TX-MCO-001",
		State:   "TX",
	})
	start := time.Now().Add(-10 * time.Millisecond)
	observability.LogWorkflowStep(ctx, "transform", start, nil)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	if entry["event"] != "workflow_step" {
		t.Fatalf("event = %v", entry["event"])
	}
	if entry["step"] != "transform" {
		t.Fatalf("step = %v", entry["step"])
	}
	if entry["status"] != "ok" {
		t.Fatalf("status = %v", entry["status"])
	}
	if entry["claim_id"] != "claim-1" {
		t.Fatalf("claim_id = %v", entry["claim_id"])
	}
}
