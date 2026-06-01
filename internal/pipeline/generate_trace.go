package pipeline

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/pkg/x12"
)

// GenerateWithTrace runs the Option 1 pipeline and records step timing.
func (g *Generator) GenerateWithTrace(ctx context.Context, claimID uuid.UUID, rec StepRecorder) (x12.Document, error) {
	if rec == nil {
		rec = noopRecorder{}
	}

	rec.Begin(StepLoad)
	claimCtx, pc, err := g.load(ctx, claimID)
	rec.End(StepLoad, err)
	if err != nil {
		return x12.Document{}, err
	}

	rec.Begin(StepRulesPre)
	err = g.preValidate(ctx, claimCtx, pc.Config.ValidationRules, pc.Config.EVVRules)
	rec.End(StepRulesPre, err)
	if err != nil {
		return x12.Document{}, err
	}

	rec.Begin(StepTransform)
	doc, err := g.transform(ctx, claimCtx)
	rec.End(StepTransform, err)
	if err != nil {
		return x12.Document{}, err
	}

	rec.Begin(StepRulesPost)
	err = g.postValidate(ctx, doc, pc.Config.BusinessRules)
	rec.End(StepRulesPost, err)
	if err != nil {
		return x12.Document{}, err
	}
	return doc, nil
}

// PersistSubmitter dry-runs claim submission for Option 1 persist step.
type PersistSubmitter interface {
	SubmitDryRun(ctx context.Context, claimID uuid.UUID, doc x12.Document) (int32, error)
}

// GenerateAndPersistWithTrace runs generation and optionally persists via submitter.
func (g *Generator) GenerateAndPersistWithTrace(ctx context.Context, claimID uuid.UUID, rec StepRecorder, submitter PersistSubmitter, skipPersist bool) (x12.Document, error) {
	doc, err := g.GenerateWithTrace(ctx, claimID, rec)
	if err != nil {
		return doc, err
	}
	if skipPersist || submitter == nil {
		rec.Skip(StepPersist)
		return doc, nil
	}
	rec.Begin(StepPersist)
	_, err = submitter.SubmitDryRun(ctx, claimID, doc)
	rec.End(StepPersist, err)
	return doc, err
}

// TraceResult builds a RunTrace from a recorder after Option 1 execution.
func TraceResult(claimID uuid.UUID, rec StepRecorder, mode string, doc x12.Document, err error) RunTrace {
	trace := RunTrace{
		ClaimID: claimID.String(),
		Mode:    mode,
		Steps:   rec.Snapshot(),
		Success: err == nil,
	}
	if rec != nil {
		trace.FailedStep = rec.FailedStep()
	}
	if err == nil {
		trace.Result = &GenerateResult{
			ClaimID:       doc.ClaimID,
			ConfigVersion: doc.ConfigVersion,
			EDI:           doc.Raw,
			GeneratedAt:   doc.GeneratedAt,
		}
		if trace.Result.GeneratedAt.IsZero() {
			trace.Result.GeneratedAt = time.Now().UTC()
		}
	}
	return trace
}

// WorkflowTraceResult builds a RunTrace from Option 3 workflow execution.
func WorkflowTraceResult(rec StepRecorder, mode string, result GenerateResult, err error) RunTrace {
	trace := RunTrace{
		ClaimID: result.ClaimID,
		Mode:    mode,
		Steps:   rec.Snapshot(),
		Success: err == nil,
	}
	if rec != nil {
		trace.FailedStep = rec.FailedStep()
	}
	if err == nil {
		trace.Result = &result
	}
	return trace
}
