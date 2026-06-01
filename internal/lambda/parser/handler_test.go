package parser_test

import (
	"context"
	"testing"

	"github.com/pavillio/pav-edi/internal/lambda/parser"
	"github.com/pavillio/pav-edi/internal/pipeline"
)

func TestHandler_requiresS3Location(t *testing.T) {
	_, err := (&parser.Handler{}).Handle(context.Background(), pipeline.Parse277Request{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHandler_requiresObjectReader(t *testing.T) {
	_, err := (&parser.Handler{}).Handle(context.Background(), pipeline.Parse277Request{
		S3Bucket: "b",
		S3Key:    "k",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
