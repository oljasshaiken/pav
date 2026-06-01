package queue_test

import (
	"context"
	"testing"

	"github.com/pavillio/pav-edi/internal/queue"
)

func TestMemoryPublisher_recordsMessages(t *testing.T) {
	pub := &queue.MemoryPublisher{}
	msg := queue.DLQMessage{ClaimID: "c1", Code: "VALIDATION_FAILED", Message: "diagnosis required"}
	if err := pub.Publish(context.Background(), msg); err != nil {
		t.Fatal(err)
	}
	got, ok := pub.Last()
	if !ok || got.ClaimID != "c1" {
		t.Fatalf("got %+v", got)
	}
}
