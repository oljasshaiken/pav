package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pavillio/pav-edi/internal/api"
	"github.com/pavillio/pav-edi/internal/validation"
)

func TestHealth(t *testing.T) {
	srv := &api.Server{Validate: validation.NoopPipeline{}}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Fatalf("body = %v", body)
	}
}

func TestInvalidClaimID(t *testing.T) {
	srv := &api.Server{
		Store:    nil,
		Validate: validation.NoopPipeline{},
	}
	req := httptest.NewRequest(http.MethodGet, "/claims/not-a-uuid/edi", nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", rec.Code)
	}
}
