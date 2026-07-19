package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"sentinel/internal/analytics/emitter"
	"sentinel/internal/analytics/event"
	"sentinel/internal/domain"
	"sentinel/internal/engine"
	"sentinel/internal/http/dto"
	"sentinel/internal/http/httperr"
)

// CheckHandler serves the rate-limit check endpoint and emits analytics events.
type CheckHandler struct {
	engine  *engine.DecisionEngine
	emitter emitter.EventEmitter
}

// NewCheckHandler creates a handler that uses the decision engine for rate-limit checks.
func NewCheckHandler(eng *engine.DecisionEngine, em emitter.EventEmitter) *CheckHandler {
	return &CheckHandler{engine: eng, emitter: em}
}

// Check handles POST /v1/check — accepts client_id and api, returns allowed/remaining or rejected/retry_after.
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

	start := time.Now()
	dec, _, err := h.engine.Decide(r.Context(), req.ClientID, req.API)
	elapsed := time.Since(start)

	if err != nil {
		httperr.WriteError(w, err)
		return
	}

	status := event.StatusAllowed
	if !dec.Allowed {
		status = event.StatusRejected
	}
	h.emitter.Emit(event.CheckEvent{
		ClientID:  req.ClientID,
		API:       req.API,
		Timestamp: time.Now().UTC(),
		LatencyMS: int(elapsed.Milliseconds()),
		Status:    status,
	})

	w.Header().Set("Content-Type", "application/json")
	if dec.Allowed {
		json.NewEncoder(w).Encode(dto.CheckAllowedResponse{Allowed: true, Remaining: dec.Remaining})
	} else {
		json.NewEncoder(w).Encode(dto.CheckRejectedResponse{Allowed: false, RetryAfter: dec.RetryAfter})
	}
}
