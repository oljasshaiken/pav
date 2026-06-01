package observability

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/pavillio/pav-edi/internal/queue"
)

type ctxKey int

const workflowKey ctxKey = 1

// WorkflowFields are correlation attributes attached to workflow logs.
type WorkflowFields struct {
	ClaimID string
	PayerID string
	State   string
}

// InitLambda configures JSON structured logging for CloudWatch Logs.
func InitLambda() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
}

// WithWorkflow attaches claim correlation fields to ctx for downstream logs.
func WithWorkflow(ctx context.Context, fields WorkflowFields) context.Context {
	return context.WithValue(ctx, workflowKey, fields)
}

// WorkflowFromContext returns workflow correlation fields when present.
func WorkflowFromContext(ctx context.Context) (WorkflowFields, bool) {
	fields, ok := ctx.Value(workflowKey).(WorkflowFields)
	return fields, ok
}

// LogWorkflowStep emits a structured workflow_step event with duration.
func LogWorkflowStep(ctx context.Context, step string, start time.Time, err error) {
	attrs := []any{
		slog.String("event", "workflow_step"),
		slog.String("step", step),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
	}
	attrs = append(attrs, workflowAttrs(ctx)...)
	if err != nil {
		attrs = append(attrs, slog.String("status", "error"), slog.String("error", err.Error()))
		slog.ErrorContext(ctx, "workflow step failed", attrs...)
		return
	}
	attrs = append(attrs, slog.String("status", "ok"))
	slog.InfoContext(ctx, "workflow step completed", attrs...)
}

// LogDLQAlert emits a structured dlq_alert event for CloudWatch alarms/Insights.
func LogDLQAlert(ctx context.Context, msg queue.DLQMessage) {
	attrs := []any{
		slog.String("event", "dlq_alert"),
		slog.String("claim_id", msg.ClaimID),
		slog.String("payer_id", msg.PayerID),
		slog.String("code", msg.Code),
		slog.String("message", msg.Message),
	}
	if msg.State != "" {
		attrs = append(attrs, slog.String("state", msg.State))
	}
	if msg.Phase != "" {
		attrs = append(attrs, slog.String("phase", msg.Phase))
	}
	if msg.RuleID != "" {
		attrs = append(attrs, slog.String("rule_id", msg.RuleID))
	}
	slog.WarnContext(ctx, "claim routed to DLQ", attrs...)
}

func workflowAttrs(ctx context.Context) []any {
	fields, ok := WorkflowFromContext(ctx)
	if !ok {
		return nil
	}
	attrs := make([]any, 0, 3)
	if fields.ClaimID != "" {
		attrs = append(attrs, slog.String("claim_id", fields.ClaimID))
	}
	if fields.PayerID != "" {
		attrs = append(attrs, slog.String("payer_id", fields.PayerID))
	}
	if fields.State != "" {
		attrs = append(attrs, slog.String("state", fields.State))
	}
	return attrs
}
