package dispatch_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pavillio/pav-edi/internal/lambda/dispatch"
	"github.com/pavillio/pav-edi/internal/queue"
)

func TestHandleEvent_singleClaimMessage(t *testing.T) {
	var ran string
	h := &dispatch.Handler{
		Workflow: dispatch.WorkflowAdapter{
			RunFn: func(_ context.Context, claimID string) error {
				ran = claimID
				return nil
			},
		},
	}
	raw, _ := json.Marshal(queue.ClaimQueueMessage{ClaimID: "abc", PayerID: "TX-MCO-001"})
	if err := h.HandleEvent(context.Background(), raw); err != nil {
		t.Fatal(err)
	}
	if ran != "abc" {
		t.Fatalf("ran = %q", ran)
	}
}

func TestHandleEvent_sqsBatch(t *testing.T) {
	count := 0
	h := &dispatch.Handler{
		Workflow: dispatch.WorkflowAdapter{
			RunFn: func(_ context.Context, _ string) error {
				count++
				return nil
			},
		},
	}
	body, _ := json.Marshal(queue.ClaimQueueMessage{ClaimID: "abc", PayerID: "TX-MCO-001"})
	event, _ := json.Marshal(map[string]any{
		"Records": []map[string]string{{"body": string(body)}},
	})
	if err := h.HandleEvent(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("count = %d", count)
	}
}
