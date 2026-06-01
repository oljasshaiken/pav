package pipeline

import (
	"time"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/pkg/x12"
)

// LoadClaimRequest loads claim context and payer config from Postgres.
type LoadClaimRequest struct {
	ClaimID string `json:"claim_id"`
}

// LoadClaimResult is the loaded state for downstream workflow steps.
type LoadClaimResult struct {
	ClaimID       string                 `json:"claim_id"`
	ClaimContext  domain.ClaimContext    `json:"claim_context"`
	PayerConfig   domain.PayerConfigBody `json:"payer_config"`
	ConfigVersion int32                  `json:"config_version"`
}

// DLQPublishRequest publishes a workflow failure to the dead-letter queue.
type DLQPublishRequest struct {
	ClaimID string         `json:"claim_id"`
	PayerID string         `json:"payer_id"`
	State   string         `json:"state,omitempty"`
	Phase   RulesPhase     `json:"phase,omitempty"`
	Error   *WorkflowError `json:"error"`
}

// GenerateRequest starts OutboundClaimWorkflow.
type GenerateRequest struct {
	ClaimID string `json:"claim_id"`
}

// GenerateResult is the final workflow output after persist.
type GenerateResult struct {
	ClaimID       string    `json:"claim_id"`
	ConfigVersion int32     `json:"config_version"`
	EDI           string    `json:"edi"`
	S3Key         string    `json:"s3_key,omitempty"`
	GeneratedAt   time.Time `json:"generated_at"`
}

// RulesEvaluateRequest runs CEL pre- or post-transform validation.
type RulesEvaluateRequest struct {
	ClaimID       string                 `json:"claim_id"`
	Phase         RulesPhase             `json:"phase"`
	ClaimContext  domain.ClaimContext    `json:"claim_context"`
	PayerConfig   domain.PayerConfigBody `json:"payer_config"`
	ConfigVersion int32                  `json:"config_version"`
	Document      *x12.Document          `json:"document,omitempty"`
}

// RulesPhase identifies which rule set to evaluate.
type RulesPhase string

const (
	RulesPhasePreTransform  RulesPhase = "pre_transform"
	RulesPhasePostTransform RulesPhase = "post_transform"
)

// RulesEvaluateResult returns validation outcome.
type RulesEvaluateResult struct {
	ClaimID string         `json:"claim_id"`
	Valid   bool           `json:"valid"`
	Error   *WorkflowError `json:"error,omitempty"`
}

// TransformRequest builds 837P from claim context and payer config.
type TransformRequest struct {
	ClaimID       string                 `json:"claim_id"`
	ConfigVersion int32                  `json:"config_version"`
	ClaimContext  domain.ClaimContext    `json:"claim_context"`
	PayerConfig   domain.PayerConfigBody `json:"payer_config"`
	GeneratedAt   time.Time              `json:"generated_at,omitempty"`
}

// TransformResult is X12 output from the transformer Lambda.
type TransformResult struct {
	ClaimID       string       `json:"claim_id"`
	ConfigVersion int32        `json:"config_version"`
	Document      x12.Document `json:"document"`
}

// PersistRequest writes EDI to Postgres and optional S3 outbound bucket.
type PersistRequest struct {
	ClaimID     string       `json:"claim_id"`
	Document    x12.Document `json:"document"`
	S3Bucket    string       `json:"s3_bucket,omitempty"`
	S3KeyPrefix string       `json:"s3_key_prefix,omitempty"`
}

// PersistResult confirms persistence.
type PersistResult struct {
	ClaimID           string `json:"claim_id"`
	SubmissionAttempt int32  `json:"submission_attempt"`
	S3Key             string `json:"s3_key,omitempty"`
}

// Parse277Request parses inbound 277/999 from S3 (InboundAckWorkflow).
type Parse277Request struct {
	S3Bucket string `json:"s3_bucket"`
	S3Key    string `json:"s3_key"`
	ClaimID  string `json:"claim_id,omitempty"`
}

// Parse277Result stores parsed acknowledgment text.
type Parse277Result struct {
	ClaimID     string `json:"claim_id"`
	Response277 string `json:"response_277"`
}

// Parse271Request parses inbound 271 eligibility responses from S3.
type Parse271Request struct {
	S3Bucket  string `json:"s3_bucket"`
	S3Key     string `json:"s3_key"`
	PatientID string `json:"patient_id,omitempty"`
	PayerID   string `json:"payer_id,omitempty"`
}

// Parse271Result stores parsed eligibility response metadata.
type Parse271Result struct {
	PatientID      string `json:"patient_id"`
	InquiryRef     string `json:"inquiry_ref"`
	PayerID        string `json:"payer_id"`
	CoverageStatus string `json:"coverage_status"`
	ServiceType    string `json:"service_type,omitempty"`
	Response271    string `json:"response_271"`
}

// WorkflowError is a structured failure for Step Functions catch/DLQ.
type WorkflowError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	RuleID  string `json:"rule_id,omitempty"`
}
