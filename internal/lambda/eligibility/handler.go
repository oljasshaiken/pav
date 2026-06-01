package eligibility

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/pipeline"
	"github.com/pavillio/pav-edi/internal/repository"
	"github.com/pavillio/pav-edi/pkg/x12"
)

// ObjectReader fetches inbound eligibility files (S3 in AWS, memory in tests).
type ObjectReader interface {
	GetObject(ctx context.Context, bucket, key string) ([]byte, error)
}

// Handler parses 271 eligibility responses and persists them.
type Handler struct {
	Store  *repository.Store
	Object ObjectReader
}

func (h *Handler) Handle(ctx context.Context, req pipeline.Parse271Request) (pipeline.Parse271Result, error) {
	if req.S3Bucket == "" || req.S3Key == "" {
		return pipeline.Parse271Result{}, fmt.Errorf("s3_bucket and s3_key required")
	}
	if h.Object == nil {
		return pipeline.Parse271Result{}, fmt.Errorf("object reader required")
	}
	if h.Store == nil {
		return pipeline.Parse271Result{}, fmt.Errorf("store required")
	}

	body, err := h.Object.GetObject(ctx, req.S3Bucket, req.S3Key)
	if err != nil {
		return pipeline.Parse271Result{}, fmt.Errorf("get object: %w", err)
	}

	parsed, err := x12.ParseEligibility271(string(body))
	if err != nil {
		return pipeline.Parse271Result{}, err
	}

	patientID, err := h.resolvePatientID(ctx, req.PatientID, parsed)
	if err != nil {
		return pipeline.Parse271Result{}, err
	}

	payerID := req.PayerID
	if payerID == "" {
		payerID = parsed.PayerID
	}
	if payerID == "" {
		return pipeline.Parse271Result{}, fmt.Errorf("payer_id required")
	}

	coverage := parsed.CoverageStatus
	if coverage == "" {
		coverage = domain.CoverageUnknown
	}

	raw := string(body)
	_, err = h.Store.SaveEligibilityResponse(ctx, domain.EligibilityResponse{
		PatientID:      patientID,
		PayerID:        payerID,
		InquiryRef:     parsed.InquiryRef,
		CoverageStatus: coverage,
		ServiceType:    parsed.ServiceType,
		Response271:    raw,
	})
	if err != nil {
		return pipeline.Parse271Result{}, err
	}

	return pipeline.Parse271Result{
		PatientID:      patientID.String(),
		InquiryRef:     parsed.InquiryRef,
		PayerID:        payerID,
		CoverageStatus: coverage,
		ServiceType:    parsed.ServiceType,
		Response271:    raw,
	}, nil
}

func (h *Handler) resolvePatientID(ctx context.Context, explicit string, parsed x12.Eligibility271) (uuid.UUID, error) {
	if explicit != "" {
		id, err := uuid.Parse(explicit)
		if err != nil {
			return uuid.Nil, fmt.Errorf("invalid patient_id: %w", err)
		}
		return id, nil
	}
	if parsed.MedicaidID != "" {
		id, err := h.Store.FindPatientIDByMedicaidID(ctx, parsed.MedicaidID)
		if err == nil {
			return id, nil
		}
		if !errors.Is(err, repository.ErrNotFound) {
			return uuid.Nil, err
		}
	}
	if parsed.InquiryRef != "" {
		claimID, err := h.Store.FindClaimIDByNumber(ctx, parsed.InquiryRef)
		if err != nil {
			return uuid.Nil, fmt.Errorf("resolve patient from inquiry ref %q: %w", parsed.InquiryRef, err)
		}
		return h.Store.PatientIDForClaim(ctx, claimID)
	}
	return uuid.Nil, fmt.Errorf("patient_id required for 271 without resolvable subscriber or inquiry reference")
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
