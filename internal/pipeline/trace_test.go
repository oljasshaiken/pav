package pipeline_test

import (
	"errors"
	"testing"

	"github.com/pavillio/pav-edi/internal/pipeline"
)

func TestMemoryRecorder_successFlow(t *testing.T) {
	r := pipeline.NewMemoryRecorder()
	r.Begin(pipeline.StepLoad)
	r.End(pipeline.StepLoad, nil)
	r.Begin(pipeline.StepRulesPre)
	r.End(pipeline.StepRulesPre, nil)
	r.Skip(pipeline.StepPersist)

	steps := r.Snapshot()
	if len(steps) != 5 {
		t.Fatalf("steps = %d", len(steps))
	}
	if steps[0].Status != pipeline.StepOK || steps[0].ID != pipeline.StepLoad {
		t.Fatalf("load step = %+v", steps[0])
	}
	if steps[4].Status != pipeline.StepSkipped {
		t.Fatalf("persist = %+v", steps[4])
	}
	if r.FailedStep() != "" {
		t.Fatalf("failed_step = %q", r.FailedStep())
	}
}

func TestMemoryRecorder_failureStopsAtStep(t *testing.T) {
	r := pipeline.NewMemoryRecorder()
	r.Begin(pipeline.StepLoad)
	r.End(pipeline.StepLoad, nil)
	r.Begin(pipeline.StepRulesPre)
	r.End(pipeline.StepRulesPre, errors.New("validation failed"))

	if r.FailedStep() != pipeline.StepRulesPre {
		t.Fatalf("failed_step = %q", r.FailedStep())
	}
	steps := r.Snapshot()
	if steps[2].Status != pipeline.StepPending {
		t.Fatalf("transform should stay pending, got %s", steps[2].Status)
	}
}
