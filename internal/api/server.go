package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/repository"
	"github.com/pavillio/pav-edi/internal/submission"
	"github.com/pavillio/pav-edi/internal/validation"
	"github.com/pavillio/pav-edi/pkg/x12"
)

type Transformer interface {
	Transform(ctx context.Context, input domain.ClaimContext) (x12.Document, error)
}

type Server struct {
	Engine     Transformer
	EngineName string
	Store      *repository.Store
	Validate   validation.Pipeline
	Submit     submission.Service
}

type ediResponse struct {
	ClaimID       string    `json:"claim_id"`
	Engine        string    `json:"engine"`
	ConfigVersion int32     `json:"config_version"`
	EDI           string    `json:"edi"`
	GeneratedAt   time.Time `json:"generated_at"`
}

type submitResponse struct {
	ClaimID           string `json:"claim_id"`
	DryRun            bool   `json:"dry_run"`
	SubmissionAttempt int32  `json:"submission_attempt"`
	EDIHash           string `json:"edi_hash"`
	EDIPreview        string `json:"edi_preview"`
	Status            string `json:"status"`
}

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (s *Server) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/health", s.handleHealth)
	r.Get("/claims/{claimID}/edi", s.handleEDI)
	r.Post("/claims/{claimID}/submit", s.handleSubmit)
	return r
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleEDI(w http.ResponseWriter, r *http.Request) {
	claimID, ok := parseClaimID(w, r)
	if !ok {
		return
	}
	if s.Store == nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error")
		return
	}

	doc, err := s.generateEDI(r.Context(), claimID)
	if err := writeGenerateError(w, err); err != nil {
		return
	}

	writeJSON(w, http.StatusOK, ediResponse{
		ClaimID:       doc.ClaimID,
		Engine:        s.EngineName,
		ConfigVersion: doc.ConfigVersion,
		EDI:           doc.Raw,
		GeneratedAt:   doc.GeneratedAt,
	})
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	claimID, ok := parseClaimID(w, r)
	if !ok {
		return
	}
	if s.Store == nil || s.Submit == nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error")
		return
	}

	dryRun, err := parseDryRun(r)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_REQUEST", "invalid dry_run parameter")
		return
	}
	if !dryRun {
		writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "real submission transport is not available in Phase 1")
		return
	}

	doc, err := s.generateEDI(r.Context(), claimID)
	if err := writeGenerateError(w, err); err != nil {
		return
	}

	attempt, err := s.Submit.SubmitDryRun(r.Context(), claimID, doc)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, submitResponse{
		ClaimID:           claimID.String(),
		DryRun:            true,
		SubmissionAttempt: attempt,
		EDIHash:           hashEDI(doc.Raw),
		EDIPreview:        previewEDI(doc.Raw),
		Status:            "DRAFT",
	})
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

func parseDryRun(r *http.Request) (bool, error) {
	raw := r.URL.Query().Get("dry_run")
	if raw == "" {
		return false, nil
	}
	return strconv.ParseBool(raw)
}

func writeGenerateError(w http.ResponseWriter, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, errClaimNotFound) {
		writeError(w, http.StatusNotFound, "CLAIM_NOT_FOUND", "claim not found")
		return err
	}
	if errors.Is(err, errConfigNotFound) {
		writeError(w, http.StatusUnprocessableEntity, "CONFIG_NOT_FOUND", "config not found")
		return err
	}
	if validation.IsValidationError(err) {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", err.Error())
		return err
	}
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error")
	return err
}

func hashEDI(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func previewEDI(raw string) string {
	const max = 80
	if len(raw) <= max {
		return raw
	}
	return raw[:max]
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
