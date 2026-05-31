package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/pavillio/pav-edi/internal/domain"
	"github.com/pavillio/pav-edi/internal/repository"
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
}

type ediResponse struct {
	ClaimID       string    `json:"claim_id"`
	Engine        string    `json:"engine"`
	ConfigVersion int32     `json:"config_version"`
	EDI           string    `json:"edi"`
	GeneratedAt   time.Time `json:"generated_at"`
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
	return r
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleEDI(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "claimID")
	claimID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_CLAIM_ID", "invalid claim id")
		return
	}
	if s.Store == nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error")
		return
	}

	ctx, err := s.Store.LoadClaimContext(r.Context(), claimID)
	if errors.Is(err, repository.ErrNotFound) || errors.Is(err, repository.ErrNoServiceLines) {
		writeError(w, http.StatusNotFound, "CLAIM_NOT_FOUND", "claim not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error")
		return
	}

	doc, err := s.Engine.Transform(r.Context(), ctx)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusUnprocessableEntity, "CONFIG_NOT_FOUND", "config not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error")
		return
	}

	if err := s.Validate.Validate(r.Context(), doc); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error")
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
