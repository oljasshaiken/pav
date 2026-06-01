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
	"github.com/pavillio/pav-edi/internal/platform/observability"
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
	return o.RunWithTrace(ctx, claimID, nil)
}

// RunWithTrace executes the workflow and records step timing when rec is non-nil.
func (o *Outbound) RunWithTrace(ctx context.Context, claimID string, rec pipeline.StepRecorder) (pipeline.GenerateResult, error) {
	if rec == nil {
		rec = pipeline.NoopRecorder{}
	}

	start := time.Now()
	rec.Begin(pipeline.StepLoad)
	loaded, err := o.Load.Handle(ctx, pipeline.LoadClaimRequest{ClaimID: claimID})
	o.recordStep(ctx, rec, pipeline.StepLoad, start, err)
	if err != nil {
		return pipeline.GenerateResult{}, err
	}
	ctx = observability.WithWorkflow(ctx, observability.WorkflowFields{
		ClaimID: loaded.ClaimID,
		PayerID: loaded.ClaimContext.Claim.PayerID,
		State:   loaded.ClaimContext.Claim.State,
	})

	start = time.Now()
	rec.Begin(pipeline.StepRulesPre)
	pre, err := o.Rules.Handle(ctx, pipeline.RulesEvaluateRequest{
		ClaimID:       loaded.ClaimID,
		Phase:         pipeline.RulesPhasePreTransform,
		ClaimContext:  loaded.ClaimContext,
		PayerConfig:   loaded.PayerConfig,
		ConfigVersion: loaded.ConfigVersion,
	})
	if err != nil {
		o.recordStep(ctx, rec, pipeline.StepRulesPre, start, err)
		return pipeline.GenerateResult{}, err
	}
	if !pre.Valid {
		werr := workflowError(pre.Error)
		o.recordStep(ctx, rec, pipeline.StepRulesPre, start, werr)
		o.publishDLQ(ctx, loaded, pipeline.RulesPhasePreTransform, pre.Error)
		return pipeline.GenerateResult{}, werr
	}
	o.recordStep(ctx, rec, pipeline.StepRulesPre, start, nil)

	now := time.Now().UTC()
	if o.Now != nil {
		now = o.Now()
	}
	start = time.Now()
	rec.Begin(pipeline.StepTransform)
	transformed, err := o.Transform.Handle(ctx, pipeline.TransformRequest{
		ClaimID:       loaded.ClaimID,
		ConfigVersion: loaded.ConfigVersion,
		ClaimContext:  loaded.ClaimContext,
		PayerConfig:   loaded.PayerConfig,
		GeneratedAt:   now,
	})
	o.recordStep(ctx, rec, pipeline.StepTransform, start, err)
	if err != nil {
		return pipeline.GenerateResult{}, err
	}

	start = time.Now()
	rec.Begin(pipeline.StepRulesPost)
	post, err := o.Rules.Handle(ctx, pipeline.RulesEvaluateRequest{
		ClaimID:       loaded.ClaimID,
		Phase:         pipeline.RulesPhasePostTransform,
		ClaimContext:  loaded.ClaimContext,
		PayerConfig:   loaded.PayerConfig,
		ConfigVersion: loaded.ConfigVersion,
		Document:      &transformed.Document,
	})
	if err != nil {
		o.recordStep(ctx, rec, pipeline.StepRulesPost, start, err)
		return pipeline.GenerateResult{}, err
	}
	if !post.Valid {
		werr := workflowError(post.Error)
		o.recordStep(ctx, rec, pipeline.StepRulesPost, start, werr)
		o.publishDLQ(ctx, loaded, pipeline.RulesPhasePostTransform, post.Error)
		return pipeline.GenerateResult{}, werr
	}
	o.recordStep(ctx, rec, pipeline.StepRulesPost, start, nil)

	if o.SkipPersist {
		rec.Skip(pipeline.StepPersist)
		return pipeline.GenerateResult{
			ClaimID:       loaded.ClaimID,
			ConfigVersion: loaded.ConfigVersion,
			EDI:           transformed.Document.Raw,
			GeneratedAt:   transformed.Document.GeneratedAt,
		}, nil
	}

	start = time.Now()
	rec.Begin(pipeline.StepPersist)
	saved, err := o.Persist.Handle(ctx, pipeline.PersistRequest{
		ClaimID:     loaded.ClaimID,
		Document:    transformed.Document,
		S3Bucket:    o.S3Bucket,
		S3KeyPrefix: o.S3KeyPrefix,
	})
	o.recordStep(ctx, rec, pipeline.StepPersist, start, err)
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

func (o *Outbound) recordStep(ctx context.Context, rec pipeline.StepRecorder, step string, start time.Time, err error) {
	observability.LogWorkflowStep(ctx, step, start, err)
	rec.End(step, err)
}

func (o *Outbound) publishDLQ(ctx context.Context, loaded pipeline.LoadClaimResult, phase pipeline.RulesPhase, werr *pipeline.WorkflowError) {
	if o.DLQ == nil || werr == nil {
		return
	}
	msg := queue.DLQMessage{
		ClaimID: loaded.ClaimID,
		PayerID: loaded.ClaimContext.Claim.PayerID,
		State:   loaded.ClaimContext.Claim.State,
		Phase:   string(phase),
		Code:    werr.Code,
		Message: werr.Message,
		RuleID:  werr.RuleID,
	}
	observability.LogDLQAlert(ctx, msg)
	_ = o.DLQ.Publish(ctx, msg)
}

func workflowError(werr *pipeline.WorkflowError) error {
	if werr == nil {
		return fmt.Errorf("validation failed")
	}
	return fmt.Errorf("%s: %s", werr.Code, werr.Message)
}
