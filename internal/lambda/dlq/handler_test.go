package dlq_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/pavillio/pav-edi/internal/lambda/dlq"
	"github.com/pavillio/pav-edi/internal/pipeline"
	"github.com/pavillio/pav-edi/internal/queue"
)

func TestHandler_publishesAndLogsDLQAlert(t *testing.T) {
	var buf bytes.Buffer
	old := slog.Default()
	t.Cleanup(func() { slog.SetDefault(old) })
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	pub := &queue.MemoryPublisher{}
	err := (&dlq.Handler{Publisher: pub}).Handle(context.Background(), pipeline.DLQPublishRequest{
		ClaimID: "00000000-0000-4000-8000-000000000001",
		PayerID: "TX-MCO-001",
		State:   "TX",
		Phase:   pipeline.RulesPhasePreTransform,
		Error:   &pipeline.WorkflowError{Code: "VALIDATION_FAILED", Message: "diagnosis required", RuleID: "diagnosis_required"},
	})
	if err != nil {
		t.Fatal(err)
	}
	last, ok := pub.Last()
	if !ok || last.Code != "VALIDATION_FAILED" {
		t.Fatalf("publish = %+v ok=%v", last, ok)
	}

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	if entry["event"] != "dlq_alert" {
		t.Fatalf("event = %v", entry["event"])
	}
}
