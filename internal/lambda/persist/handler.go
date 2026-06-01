package persist

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/pipeline"
	"github.com/pavillio/pav-edi/internal/repository"
)

// ObjectStore uploads outbound EDI (S3 in AWS, LocalStack in dev).
type ObjectStore interface {
	PutObject(ctx context.Context, bucket, key string, body []byte) error
}

// NoopObjectStore skips S3 uploads (tests and DB-only dry-run).
type NoopObjectStore struct{}

func (NoopObjectStore) PutObject(context.Context, string, string, []byte) error {
	return nil
}

// Handler persists generated EDI to Postgres and optional S3.
type Handler struct {
	Store  *repository.Store
	Object ObjectStore
}

func (h *Handler) Handle(ctx context.Context, req pipeline.PersistRequest) (pipeline.PersistResult, error) {
	if req.ClaimID == "" {
		return pipeline.PersistResult{}, fmt.Errorf("claim_id required")
	}
	if req.Document.Raw == "" {
		return pipeline.PersistResult{}, fmt.Errorf("document required")
	}
	if h.Store == nil {
		return pipeline.PersistResult{}, fmt.Errorf("store required")
	}

	claimID, err := uuid.Parse(req.ClaimID)
	if err != nil {
		return pipeline.PersistResult{}, fmt.Errorf("invalid claim_id: %w", err)
	}

	attempt, err := h.Store.SaveGeneratedEDI(ctx, claimID, req.Document.Raw)
	if err != nil {
		return pipeline.PersistResult{}, err
	}

	s3Key := ""
	if req.S3Bucket != "" && h.Object != nil {
		s3Key = req.S3KeyPrefix + req.ClaimID + ".837"
		if err := h.Object.PutObject(ctx, req.S3Bucket, s3Key, []byte(req.Document.Raw)); err != nil {
			return pipeline.PersistResult{}, fmt.Errorf("s3 put: %w", err)
		}
	}

	return pipeline.PersistResult{
		ClaimID:           req.ClaimID,
		SubmissionAttempt: attempt,
		S3Key:             s3Key,
	}, nil
}

// S3PutAPI adapts aws-sdk PutObject for ObjectStore.
type S3PutAPI interface {
	PutObject(ctx context.Context, bucket, key string, body []byte) error
}

type s3Adapter struct {
	api S3PutAPI
}

func NewS3ObjectStore(api S3PutAPI) ObjectStore {
	return s3Adapter{api: api}
}

func (a s3Adapter) PutObject(ctx context.Context, bucket, key string, body []byte) error {
	return a.api.PutObject(ctx, bucket, key, body)
}

// MemoryObjectStore records uploads for tests.
type MemoryObjectStore struct {
	Objects map[string][]byte
}

func (m *MemoryObjectStore) PutObject(_ context.Context, bucket, key string, body []byte) error {
	if m.Objects == nil {
		m.Objects = make(map[string][]byte)
	}
	m.Objects[bucket+"/"+key] = bytes.Clone(body)
	return nil
}
