package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/api"
	"github.com/pavillio/pav-edi/internal/repository"
	"github.com/pavillio/pav-edi/internal/rules"
	"github.com/pavillio/pav-edi/internal/template"
	"github.com/pavillio/pav-edi/internal/testutil"
	"github.com/pavillio/pav-edi/internal/validation"
)

func TestGetClaimEDI_RulesEngine(t *testing.T) {
	pool := testutil.StartPostgres(t)
	store := repository.New(pool)
	claimID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	testutil.InsertFixtureClaim(t, pool, claimID)
	testutil.InsertPayerConfig(t, pool)

	srv := &api.Server{
		Engine:     &rules.StubEngine{Store: store},
		EngineName: "rules",
		Store:      store,
		Validate:   validation.NoopPipeline{},
	}

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
	if !strings.HasPrefix(body.EDI, "rules:") {
		t.Fatalf("edi = %q", body.EDI)
	}
}

func TestGetClaimEDI_TemplateEngine(t *testing.T) {
	pool := testutil.StartPostgres(t)
	store := repository.New(pool)
	claimID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	testutil.InsertFixtureClaim(t, pool, claimID)
	testutil.InsertTemplateOverride(t, pool)

	srv := &api.Server{
		Engine:     &template.StubRenderer{Store: store},
		EngineName: "template",
		Store:      store,
		Validate:   validation.NoopPipeline{},
	}

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
	if !strings.HasPrefix(body.EDI, "template:") {
		t.Fatalf("edi = %q", body.EDI)
	}
}

func TestGetClaimEDI_ClaimNotFound(t *testing.T) {
	pool := testutil.StartPostgres(t)
	store := repository.New(pool)

	srv := &api.Server{
		Engine:     &rules.StubEngine{Store: store},
		EngineName: "rules",
		Store:      store,
		Validate:   validation.NoopPipeline{},
	}

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
	testutil.InsertFixtureClaim(t, pool, claimID)

	srv := &api.Server{
		Engine:     &rules.StubEngine{Store: store},
		EngineName: "rules",
		Store:      store,
		Validate:   validation.NoopPipeline{},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/claims/"+claimID.String()+"/edi", nil)
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetClaimEDI_NilStore(t *testing.T) {
	srv := &api.Server{
		Engine:     &rules.StubEngine{},
		EngineName: "rules",
		Validate:   validation.NoopPipeline{},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/claims/"+uuid.New().String()+"/edi", nil)
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}
