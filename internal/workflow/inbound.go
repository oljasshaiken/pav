package workflow

import (
	"context"

	"github.com/pavillio/pav-edi/internal/lambda/parser"
	"github.com/pavillio/pav-edi/internal/pipeline"
)

// Inbound runs Parser(277/999) → persist response_277 on the claim.
type Inbound struct {
	Parser *parser.Handler
}

func (i *Inbound) Run(ctx context.Context, req pipeline.Parse277Request) (pipeline.Parse277Result, error) {
	return i.Parser.Handle(ctx, req)
}
