package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"sentinel/internal/domain"
	"sentinel/internal/engine"
	"sentinel/internal/http/dto"
	"sentinel/internal/http/httperr"
)

type CheckHandler struct {
	engine *engine.DecisionEngine
}

func NewCheckHandler(eng *engine.DecisionEngine) *CheckHandler {
	return &CheckHandler{engine: eng}
}

func (h *CheckHandler) Check(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req dto.CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httperr.WriteError(w, fmt.Errorf("%w: invalid request body: %w", domain.ErrValidation, err))
		return
	}

	if req.ClientID == "" || req.API == "" {
		httperr.WriteError(w, fmt.Errorf("%w: client_id and api are required", domain.ErrValidation))
		return
	}

	dec, _, err := h.engine.Decide(r.Context(), req.ClientID, req.API)
	if err != nil {
		httperr.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if dec.Allowed {
		json.NewEncoder(w).Encode(dto.CheckAllowedResponse{Allowed: true, Remaining: dec.Remaining})
	} else {
		json.NewEncoder(w).Encode(dto.CheckRejectedResponse{Allowed: false, RetryAfter: dec.RetryAfter})
	}
}
