package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/pavillio/pav-edi/internal/lambda/rules"
	"github.com/pavillio/pav-edi/internal/pipeline"
	"github.com/pavillio/pav-edi/internal/platform/observability"
)

func main() {
	observability.InitLambda()
	lambda.Start(handle)
}

func handle(ctx context.Context, raw json.RawMessage) (pipeline.RulesEvaluateResult, error) {
	var req pipeline.RulesEvaluateRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return pipeline.RulesEvaluateResult{}, err
	}
	return (&rules.Handler{}).Handle(ctx, req)
}
