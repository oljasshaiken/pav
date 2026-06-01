package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/pavillio/pav-edi/internal/lambda/load"
	"github.com/pavillio/pav-edi/internal/lambda/persist"
	"github.com/pavillio/pav-edi/internal/lambda/rules"
	"github.com/pavillio/pav-edi/internal/lambda/transformer"
	"github.com/pavillio/pav-edi/internal/pipeline"
	"github.com/pavillio/pav-edi/internal/queue"
)

// Outbound runs LoadClaim → Rules(pre) → Transform → Rules(post) → Persist.
type Outbound struct {
	Load        *load.Handler
	Rules       *rules.Handler
	Transform   *transformer.Handler
	Persist     *persist.Handler
	DLQ         queue.Publisher
	Now         func() time.Time
	S3Bucket    string
	S3KeyPrefix string
	// SkipPersist generates EDI without writing to Postgres/S3 (compare / dry-run).
	SkipPersist bool
}

// Run executes the TX outbound claim workflow for a claim ID.
func (o *Outbound) Run(ctx context.Context, claimID string) (pipeline.GenerateResult, error) {
	loaded, err := o.Load.Handle(ctx, pipeline.LoadClaimRequest{ClaimID: claimID})
	if err != nil {
		return pipeline.GenerateResult{}, err
	}

	pre, err := o.Rules.Handle(ctx, pipeline.RulesEvaluateRequest{
		ClaimID:       loaded.ClaimID,
		Phase:         pipeline.RulesPhasePreTransform,
		ClaimContext:  loaded.ClaimContext,
		PayerConfig:   loaded.PayerConfig,
		ConfigVersion: loaded.ConfigVersion,
	})
	if err != nil {
		return pipeline.GenerateResult{}, err
	}
	if !pre.Valid {
		o.publishDLQ(ctx, loaded, pipeline.RulesPhasePreTransform, pre.Error)
		return pipeline.GenerateResult{}, workflowError(pre.Error)
	}

	now := time.Now().UTC()
	if o.Now != nil {
		now = o.Now()
	}
	transformed, err := o.Transform.Handle(ctx, pipeline.TransformRequest{
		ClaimID:       loaded.ClaimID,
		ConfigVersion: loaded.ConfigVersion,
		ClaimContext:  loaded.ClaimContext,
		PayerConfig:   loaded.PayerConfig,
		GeneratedAt:   now,
	})
	if err != nil {
		return pipeline.GenerateResult{}, err
	}

	post, err := o.Rules.Handle(ctx, pipeline.RulesEvaluateRequest{
		ClaimID:       loaded.ClaimID,
		Phase:         pipeline.RulesPhasePostTransform,
		ClaimContext:  loaded.ClaimContext,
		PayerConfig:   loaded.PayerConfig,
		ConfigVersion: loaded.ConfigVersion,
		Document:      &transformed.Document,
	})
	if err != nil {
		return pipeline.GenerateResult{}, err
	}
	if !post.Valid {
		o.publishDLQ(ctx, loaded, pipeline.RulesPhasePostTransform, post.Error)
		return pipeline.GenerateResult{}, workflowError(post.Error)
	}

	if o.SkipPersist {
		return pipeline.GenerateResult{
			ClaimID:       loaded.ClaimID,
			ConfigVersion: loaded.ConfigVersion,
			EDI:           transformed.Document.Raw,
			GeneratedAt:   transformed.Document.GeneratedAt,
		}, nil
	}

	saved, err := o.Persist.Handle(ctx, pipeline.PersistRequest{
		ClaimID:     loaded.ClaimID,
		Document:    transformed.Document,
		S3Bucket:    o.S3Bucket,
		S3KeyPrefix: o.S3KeyPrefix,
	})
	if err != nil {
		return pipeline.GenerateResult{}, err
	}

	return pipeline.GenerateResult{
		ClaimID:       loaded.ClaimID,
		ConfigVersion: loaded.ConfigVersion,
		EDI:           transformed.Document.Raw,
		S3Key:         saved.S3Key,
		GeneratedAt:   transformed.Document.GeneratedAt,
	}, nil
}

func (o *Outbound) publishDLQ(ctx context.Context, loaded pipeline.LoadClaimResult, phase pipeline.RulesPhase, werr *pipeline.WorkflowError) {
	if o.DLQ == nil || werr == nil {
		return
	}
	_ = o.DLQ.Publish(ctx, queue.DLQMessage{
		ClaimID: loaded.ClaimID,
		PayerID: loaded.ClaimContext.Claim.PayerID,
		State:   loaded.ClaimContext.Claim.State,
		Phase:   string(phase),
		Code:    werr.Code,
		Message: werr.Message,
		RuleID:  werr.RuleID,
	})
}

func workflowError(werr *pipeline.WorkflowError) error {
	if werr == nil {
		return fmt.Errorf("validation failed")
	}
	return fmt.Errorf("%s: %s", werr.Code, werr.Message)
}
