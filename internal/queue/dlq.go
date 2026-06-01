package queue

import (
	"context"
	"encoding/json"
	"sync"
)

// DLQMessage is a structured failure published when claim generation fails.
type DLQMessage struct {
	ClaimID string `json:"claim_id"`
	PayerID string `json:"payer_id"`
	State   string `json:"state,omitempty"`
	Phase   string `json:"phase,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
	RuleID  string `json:"rule_id,omitempty"`
}

// ClaimQueueMessage is the SQS FIFO payload grouped by payer_id.
type ClaimQueueMessage struct {
	ClaimID string `json:"claim_id"`
	PayerID string `json:"payer_id"`
}

// Publisher sends structured failures to a dead-letter queue.
type Publisher interface {
	Publish(ctx context.Context, msg DLQMessage) error
}

// MemoryPublisher records DLQ messages for tests.
type MemoryPublisher struct {
	mu       sync.Mutex
	Messages []DLQMessage
}

func (m *MemoryPublisher) Publish(_ context.Context, msg DLQMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Messages = append(m.Messages, msg)
	return nil
}

func (m *MemoryPublisher) Last() (DLQMessage, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.Messages) == 0 {
		return DLQMessage{}, false
	}
	return m.Messages[len(m.Messages)-1], true
}

// JSONPublisher encodes messages for SQS/API transport.
func JSONPublisher(send func(body []byte) error) Publisher {
	return PublisherFunc(func(_ context.Context, msg DLQMessage) error {
		raw, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		return send(raw)
	})
}

// PublisherFunc adapts a function to Publisher.
type PublisherFunc func(ctx context.Context, msg DLQMessage) error

func (f PublisherFunc) Publish(ctx context.Context, msg DLQMessage) error {
	return f(ctx, msg)
}
