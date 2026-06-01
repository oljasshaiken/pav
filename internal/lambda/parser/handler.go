package parser

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/pipeline"
	"github.com/pavillio/pav-edi/internal/repository"
	"github.com/pavillio/pav-edi/pkg/x12"
)

// ObjectReader fetches inbound acknowledgment files (S3 in AWS, memory in tests).
type ObjectReader interface {
	GetObject(ctx context.Context, bucket, key string) ([]byte, error)
}

// Handler parses 277/999 acknowledgments and persists response_277.
type Handler struct {
	Store  *repository.Store
	Object ObjectReader
}

func (h *Handler) Handle(ctx context.Context, req pipeline.Parse277Request) (pipeline.Parse277Result, error) {
	if req.S3Bucket == "" || req.S3Key == "" {
		return pipeline.Parse277Result{}, fmt.Errorf("s3_bucket and s3_key required")
	}
	if h.Object == nil {
		return pipeline.Parse277Result{}, fmt.Errorf("object reader required")
	}
	if h.Store == nil {
		return pipeline.Parse277Result{}, fmt.Errorf("store required")
	}

	body, err := h.Object.GetObject(ctx, req.S3Bucket, req.S3Key)
	if err != nil {
		return pipeline.Parse277Result{}, fmt.Errorf("get object: %w", err)
	}

	ack, err := x12.ParseAcknowledgment(string(body))
	if err != nil {
		return pipeline.Parse277Result{}, err
	}

	claimID, err := h.resolveClaimID(ctx, req.ClaimID, ack.ClaimRef)
	if err != nil {
		return pipeline.Parse277Result{}, err
	}

	raw := string(body)
	if err := h.Store.SaveResponse277(ctx, claimID, raw); err != nil {
		return pipeline.Parse277Result{}, err
	}

	return pipeline.Parse277Result{
		ClaimID:     claimID.String(),
		Response277: raw,
	}, nil
}

func (h *Handler) resolveClaimID(ctx context.Context, explicit, ref string) (uuid.UUID, error) {
	if explicit != "" {
		id, err := uuid.Parse(explicit)
		if err != nil {
			return uuid.Nil, fmt.Errorf("invalid claim_id: %w", err)
		}
		return id, nil
	}
	if ref == "" {
		return uuid.Nil, fmt.Errorf("claim_id required for acknowledgment without claim reference")
	}
	if id, err := uuid.Parse(ref); err == nil {
		return id, nil
	}
	id, err := h.Store.FindClaimIDByNumber(ctx, ref)
	if errors.Is(err, repository.ErrNotFound) {
		return uuid.Nil, fmt.Errorf("claim not found for ref %q", ref)
	}
	return id, err
}

// MemoryObjectReader serves test fixtures keyed by bucket/key.
type MemoryObjectReader struct {
	Objects map[string][]byte
}

func (m *MemoryObjectReader) GetObject(_ context.Context, bucket, key string) ([]byte, error) {
	if m.Objects == nil {
		return nil, fmt.Errorf("object not found")
	}
	data, ok := m.Objects[bucket+"/"+key]
	if !ok {
		return nil, fmt.Errorf("object not found: %s/%s", bucket, key)
	}
	return data, nil
}
