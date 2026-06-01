package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/lambda/load"
	"github.com/pavillio/pav-edi/internal/lambda/persist"
	lrules "github.com/pavillio/pav-edi/internal/lambda/rules"
	"github.com/pavillio/pav-edi/internal/lambda/transformer"
	"github.com/pavillio/pav-edi/internal/pipeline"
	"github.com/pavillio/pav-edi/internal/platform"
	"github.com/pavillio/pav-edi/internal/repository"
	rulesengine "github.com/pavillio/pav-edi/internal/rules"
	"github.com/pavillio/pav-edi/internal/submission"
	"github.com/pavillio/pav-edi/internal/validation"
	"github.com/pavillio/pav-edi/internal/workflow"
)

const (
	ModeOption1 = "option1"
	ModeOption3 = "option3"
)

// Server exposes workflow step traces for the Next.js dashboard.
type Server struct {
	Store       *repository.Store
	RulesURL    string
	SFN         *SFNClient
	S3Bucket    string
	S3KeyPrefix string
}

func (s *Server) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(corsMiddleware)
	r.Get("/health", s.handleHealth)
	r.Get("/api/backends", s.handleBackends)
	r.Post("/api/claims/{claimID}/run", s.handleRun)
	r.Post("/api/claims/{claimID}/run-sfn", s.handleRunSFN)
	r.Get("/api/executions", s.handleExecutionStatus)
	r.Get("/api/executions/*", s.handleExecutionStatus)
	return r
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type backendsResponse struct {
	Postgres        string `json:"postgres"`
	Pipeline        string `json:"pipeline"`
	RulesEngineHTTP string `json:"rules_engine_http,omitempty"`
	StepFunctions   string `json:"step_functions"`
}

func (s *Server) handleBackends(w http.ResponseWriter, r *http.Request) {
	resp := backendsResponse{
		Postgres:      "down",
		Pipeline:      "down",
		StepFunctions: "down",
	}
	if s.Store != nil {
		if err := s.Store.Ping(r.Context()); err == nil {
			resp.Postgres = "ok"
			resp.Pipeline = "ok"
		}
	}
	if s.RulesURL != "" {
		if probeURL(r.Context(), s.RulesURL+"/health") {
			resp.RulesEngineHTTP = "ok"
		} else {
			resp.RulesEngineHTTP = "down"
		}
	}
	if s.SFN != nil && s.SFN.Available(r.Context()) {
		resp.StepFunctions = "ok"
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	claimID, ok := parseClaimID(w, r)
	if !ok {
		return
	}
	mode := r.URL.Query().Get("mode")
	if mode != ModeOption1 && mode != ModeOption3 {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_REQUEST", "mode must be option1 or option3")
		return
	}
	skipPersist := parseSkipPersist(r)
	ctx, ok := requestContextWithGeneratedAt(w, r)
	if !ok {
		return
	}

	rec := pipeline.NewMemoryRecorder()
	var trace pipeline.RunTrace

	switch mode {
	case ModeOption1:
		trace = s.runOption1(ctx, claimID, rec, skipPersist)
	case ModeOption3:
		trace = s.runOption3(ctx, claimID.String(), rec, skipPersist)
	}

	status := http.StatusOK
	if !trace.Success {
		status = http.StatusUnprocessableEntity
	}
	writeJSON(w, status, trace)
}

func (s *Server) runOption1(ctx context.Context, claimID uuid.UUID, rec pipeline.StepRecorder, skipPersist bool) pipeline.RunTrace {
	nowFn := func() time.Time { return platform.ResolveNow(ctx, time.Now) }
	gen := pipeline.Generator{
		Store: s.Store,
		Engine: &rulesengine.RulesEngine{
			Store: s.Store,
			Now:   nowFn,
		},
		Validate: validation.ConfigPipeline{},
	}
	submitter := &submission.DryRunService{Store: s.Store}
	doc, err := gen.GenerateAndPersistWithTrace(ctx, claimID, rec, submitter, skipPersist)
	return pipeline.TraceResult(claimID, rec, ModeOption1, doc, err)
}

func (s *Server) runOption3(ctx context.Context, claimID string, rec pipeline.StepRecorder, skipPersist bool) pipeline.RunTrace {
	nowFn := func() time.Time { return platform.ResolveNow(ctx, time.Now) }
	mem := &persist.MemoryObjectStore{}
	wf := &workflow.Outbound{
		Load:        &load.Handler{Store: s.Store},
		Rules:       &lrules.Handler{},
		Transform:   &transformer.Handler{Now: nowFn},
		Persist:     &persist.Handler{Store: s.Store, Object: mem},
		Now:         nowFn,
		SkipPersist: skipPersist,
		S3Bucket:    s.S3Bucket,
		S3KeyPrefix: s.S3KeyPrefix,
	}
	result, err := wf.RunWithTrace(ctx, claimID, rec)
	return pipeline.WorkflowTraceResult(rec, ModeOption3, result, err)
}

func (s *Server) handleRunSFN(w http.ResponseWriter, r *http.Request) {
	if s.SFN == nil {
		writeError(w, http.StatusServiceUnavailable, "SFN_UNAVAILABLE", "step functions client not configured")
		return
	}
	claimID, ok := parseClaimID(w, r)
	if !ok {
		return
	}
	if !s.SFN.Available(r.Context()) {
		writeError(w, http.StatusServiceUnavailable, "SFN_UNAVAILABLE", "step functions not available; run make sam-deploy-localstack")
		return
	}
	exec, err := s.SFN.StartExecution(r.Context(), claimID.String())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "SFN_START_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, exec)
}

func (s *Server) handleExecutionStatus(w http.ResponseWriter, r *http.Request) {
	if s.SFN == nil {
		writeError(w, http.StatusServiceUnavailable, "SFN_UNAVAILABLE", "step functions client not configured")
		return
	}
	arn := chi.URLParam(r, "*")
	if arn == "" {
		arn = r.URL.Query().Get("arn")
	}
	if arn == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "execution arn required")
		return
	}
	status, err := s.SFN.ExecutionStatus(r.Context(), arn)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "SFN_STATUS_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func parseClaimID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	idStr := chi.URLParam(r, "claimID")
	claimID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_CLAIM_ID", "invalid claim id")
		return uuid.UUID{}, false
	}
	return claimID, true
}

func parseSkipPersist(r *http.Request) bool {
	raw := r.URL.Query().Get("skip_persist")
	if raw == "" {
		return true
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return true
	}
	return v
}

func requestContextWithGeneratedAt(w http.ResponseWriter, r *http.Request) (context.Context, bool) {
	ctx := r.Context()
	ga := r.URL.Query().Get("generated_at")
	if ga == "" {
		return ctx, true
	}
	t, ok := platform.ParseGeneratedAt(ga)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_REQUEST", "invalid generated_at (RFC3339 required)")
		return ctx, false
	}
	return platform.WithGeneratedAt(ctx, t), true
}

func probeURL(ctx context.Context, url string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	var body errorBody
	body.Error.Code = code
	body.Error.Message = message
	writeJSON(w, status, body)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
