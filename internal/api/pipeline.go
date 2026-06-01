package api

import (
	"context"

	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/pipeline"
	"github.com/pavillio/pav-edi/pkg/x12"
)

func (s *Server) generateEDI(ctx context.Context, claimID uuid.UUID) (x12.Document, error) {
	return s.pipeline().Generate(ctx, claimID)
}

func (s *Server) pipeline() *pipeline.Generator {
	return &pipeline.Generator{
		Store:    s.Store,
		Engine:   s.Engine,
		Validate: s.Validate,
	}
}
