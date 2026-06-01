package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/pavillio/pav-edi/internal/lambda/transformer"
	"github.com/pavillio/pav-edi/internal/pipeline"
)

func main() {
	lambda.Start(handle)
}

func handle(ctx context.Context, raw json.RawMessage) (pipeline.TransformResult, error) {
	var req pipeline.TransformRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return pipeline.TransformResult{}, err
	}
	return (&transformer.Handler{}).Handle(ctx, req)
}
