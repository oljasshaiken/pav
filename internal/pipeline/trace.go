package pipeline

import "time"

// Canonical outbound workflow step IDs shared by Option 1 and Option 3.
const (
	StepLoad      = "load"
	StepRulesPre  = "rules_pre"
	StepTransform = "transform"
	StepRulesPost = "rules_post"
	StepPersist   = "persist"
)

// AllSteps is the fixed execution order for outbound claim workflows.
var AllSteps = []string{StepLoad, StepRulesPre, StepTransform, StepRulesPost, StepPersist}

// StepStatus is the lifecycle state of a workflow step.
type StepStatus string

const (
	StepPending StepStatus = "pending"
	StepRunning StepStatus = "running"
	StepOK      StepStatus = "ok"
	StepFailed  StepStatus = "failed"
	StepSkipped StepStatus = "skipped"
)

// StepRecord captures one step's outcome for dashboard traces.
type StepRecord struct {
	ID          string     `json:"id"`
	Status      StepStatus `json:"status"`
	DurationMS  int64      `json:"duration_ms,omitempty"`
	Error       string     `json:"error,omitempty"`
	StartedAt   time.Time  `json:"started_at,omitempty"`
	CompletedAt time.Time  `json:"completed_at,omitempty"`
}

// RunTrace is the full step timeline for a workflow run.
type RunTrace struct {
	ClaimID    string       `json:"claim_id"`
	Mode       string       `json:"mode"`
	Success    bool         `json:"success"`
	Steps      []StepRecord `json:"steps"`
	FailedStep string       `json:"failed_step,omitempty"`
	Result     *GenerateResult `json:"result,omitempty"`
}

// StepRecorder collects step events during a workflow run.
type StepRecorder interface {
	Begin(step string)
	End(step string, err error)
	Skip(step string)
	Snapshot() []StepRecord
	FailedStep() string
}

// MemoryRecorder stores step records in memory for HTTP responses.
type MemoryRecorder struct {
	steps      map[string]*StepRecord
	order      []string
	failedStep string
}

// NewMemoryRecorder pre-allocates pending steps in canonical order.
func NewMemoryRecorder() *MemoryRecorder {
	r := &MemoryRecorder{
		steps: make(map[string]*StepRecord, len(AllSteps)),
		order: make([]string, len(AllSteps)),
	}
	copy(r.order, AllSteps)
	for _, id := range AllSteps {
		r.steps[id] = &StepRecord{ID: id, Status: StepPending}
	}
	return r
}

func (r *MemoryRecorder) Begin(step string) {
	rec, ok := r.steps[step]
	if !ok {
		rec = &StepRecord{ID: step, Status: StepPending}
		r.steps[step] = rec
		r.order = append(r.order, step)
	}
	rec.Status = StepRunning
	rec.StartedAt = time.Now().UTC()
}

func (r *MemoryRecorder) End(step string, err error) {
	rec := r.steps[step]
	if rec == nil {
		rec = &StepRecord{ID: step}
		r.steps[step] = rec
		r.order = append(r.order, step)
	}
	now := time.Now().UTC()
	rec.CompletedAt = now
	if !rec.StartedAt.IsZero() {
		rec.DurationMS = now.Sub(rec.StartedAt).Milliseconds()
	}
	if err != nil {
		rec.Status = StepFailed
		rec.Error = err.Error()
		if r.failedStep == "" {
			r.failedStep = step
		}
		return
	}
	rec.Status = StepOK
}

func (r *MemoryRecorder) Skip(step string) {
	rec := r.steps[step]
	if rec == nil {
		rec = &StepRecord{ID: step}
		r.steps[step] = rec
		r.order = append(r.order, step)
	}
	rec.Status = StepSkipped
	now := time.Now().UTC()
	rec.CompletedAt = now
}

func (r *MemoryRecorder) Snapshot() []StepRecord {
	out := make([]StepRecord, 0, len(r.order))
	for _, id := range r.order {
		if rec := r.steps[id]; rec != nil {
			out = append(out, *rec)
		}
	}
	return out
}

func (r *MemoryRecorder) FailedStep() string {
	return r.failedStep
}

// NoopRecorder discards step events.
type NoopRecorder struct{}

func (NoopRecorder) Begin(string)           {}
func (NoopRecorder) End(string, error)      {}
func (NoopRecorder) Skip(string)            {}
func (NoopRecorder) Snapshot() []StepRecord { return nil }
func (NoopRecorder) FailedStep() string     { return "" }

// noopRecorder is the default when no trace is needed.
type noopRecorder = NoopRecorder
