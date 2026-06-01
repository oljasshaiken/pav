package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/api"
	"github.com/pavillio/pav-edi/internal/repository"
	"github.com/pavillio/pav-edi/internal/rules"
	"github.com/pavillio/pav-edi/internal/submission"
	"github.com/pavillio/pav-edi/internal/template"
	"github.com/pavillio/pav-edi/internal/testutil"
	"github.com/pavillio/pav-edi/internal/validation"
)

func TestGetClaimEDI_RulesEngine(t *testing.T) {
	pool := testutil.StartPostgres(t)
	store := repository.New(pool)
	claimID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	testutil.InsertGoldenFixtureClaim(t, pool, claimID)
	testutil.InsertPayerConfig(t, pool)

	srv := newRulesServer(store, goldenNow())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/claims/"+claimID.String()+"/edi", nil)
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Engine string `json:"engine"`
		EDI    string `json:"edi"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Engine != "rules" {
		t.Fatalf("engine = %q", body.Engine)
	}
	if !strings.HasPrefix(body.EDI, "ISA") {
		t.Fatalf("edi = %q", body.EDI[:min(20, len(body.EDI))])
	}
}

func TestGetClaimEDI_TemplateEngine(t *testing.T) {
	pool := testutil.StartPostgres(t)
	store := repository.New(pool)
	claimID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	testutil.InsertGoldenFixtureClaim(t, pool, claimID)
	testutil.InsertPayerConfig(t, pool)
	testutil.InsertTemplateOverride(t, pool)

	srv := newTemplateServer(store, goldenNow())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/claims/"+claimID.String()+"/edi", nil)
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Engine string `json:"engine"`
		EDI    string `json:"edi"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Engine != "template" {
		t.Fatalf("engine = %q", body.Engine)
	}
	if !strings.HasPrefix(body.EDI, "ISA") {
		t.Fatalf("edi = %q", body.EDI[:min(20, len(body.EDI))])
	}
}

func TestGetClaimEDI_ValidationFailed(t *testing.T) {
	pool := testutil.StartPostgres(t)
	store := repository.New(pool)
	claimID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	testutil.InsertGoldenFixtureClaim(t, pool, claimID)
	testutil.InsertPayerConfig(t, pool)

	_, err := pool.Exec(context.Background(), `
UPDATE claim_service_lines SET diagnosis_codes = NULL WHERE claim_id = $1`, claimID)
	if err != nil {
		t.Fatal(err)
	}

	srv := newRulesServer(store, goldenNow())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/claims/"+claimID.String()+"/edi", nil)
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetClaimEDI_ClaimNotFound(t *testing.T) {
	pool := testutil.StartPostgres(t)
	store := repository.New(pool)

	srv := newRulesServer(store, goldenNow())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/claims/"+uuid.New().String()+"/edi", nil)
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestGetClaimEDI_ConfigNotFound(t *testing.T) {
	pool := testutil.StartPostgres(t)
	store := repository.New(pool)
	claimID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	testutil.InsertGoldenFixtureClaim(t, pool, claimID)

	srv := newRulesServer(store, goldenNow())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/claims/"+claimID.String()+"/edi", nil)
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestSubmitDryRun(t *testing.T) {
	pool := testutil.StartPostgres(t)
	store := repository.New(pool)
	claimID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	testutil.InsertGoldenFixtureClaim(t, pool, claimID)
	testutil.InsertPayerConfig(t, pool)

	srv := newRulesServer(store, goldenNow())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/claims/"+claimID.String()+"/submit?dry_run=true", nil)
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		DryRun            bool   `json:"dry_run"`
		SubmissionAttempt int32  `json:"submission_attempt"`
		EDIHash           string `json:"edi_hash"`
		Status            string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.DryRun || body.SubmissionAttempt != 1 || body.Status != "DRAFT" {
		t.Fatalf("body = %+v", body)
	}
	if !strings.HasPrefix(body.EDIHash, "sha256:") {
		t.Fatalf("hash = %q", body.EDIHash)
	}
}

func TestSubmitNotImplemented(t *testing.T) {
	pool := testutil.StartPostgres(t)
	store := repository.New(pool)
	claimID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	testutil.InsertGoldenFixtureClaim(t, pool, claimID)

	srv := newRulesServer(store, goldenNow())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/claims/"+claimID.String()+"/submit?dry_run=false", nil)
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestGetClaimEDI_NilStore(t *testing.T) {
	srv := &api.Server{
		Engine:   &rules.RulesEngine{},
		Validate: validation.NoopPipeline{},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/claims/"+uuid.New().String()+"/edi", nil)
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}

func newRulesServer(store *repository.Store, now time.Time) *api.Server {
	return &api.Server{
		Engine: &rules.RulesEngine{
			Store: store,
			Now:   func() time.Time { return now },
		},
		EngineName: "rules",
		Store:      store,
		Validate:   validation.ConfigPipeline{},
		Submit:     &submission.DryRunService{Store: store},
	}
}

func newTemplateServer(store *repository.Store, now time.Time) *api.Server {
	return &api.Server{
		Engine: &template.Renderer{
			Store: store,
			Now:   func() time.Time { return now },
		},
		EngineName: "template",
		Store:      store,
		Validate:   validation.ConfigPipeline{},
		Submit:     &submission.DryRunService{Store: store},
	}
}

func goldenNow() time.Time {
	return time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
