package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"sentinel/internal/domain"
	"sentinel/internal/http/dto"
	"sentinel/internal/http/httperr"
	"sentinel/internal/repository"
)

// RulesHandler serves rate-rule CRUD endpoints backed by a repository.RateRuleRepository.
type RulesHandler struct {
	repo repository.RateRuleRepository
}

// NewRulesHandler creates a RulesHandler with the given repository.
func NewRulesHandler(repo repository.RateRuleRepository) *RulesHandler {
	return &RulesHandler{repo: repo}
}

// List handles GET /rules with optional ?client_id= and ?api= filters.
func (h *RulesHandler) List(w http.ResponseWriter, r *http.Request) {
	params := repository.ListRulesParams{}

	if clientIDStr := r.URL.Query().Get("client_id"); clientIDStr != "" {
		id, err := uuid.Parse(clientIDStr)
		if err != nil {
			httperr.WriteError(w, domain.ErrValidation)
			return
		}
		params.ClientID = &id
	}

	if api := r.URL.Query().Get("api"); api != "" {
		if err := domain.ValidateAPIIdentifier(api); err != nil {
			httperr.WriteError(w, err)
			return
		}
		params.API = &api
	}

	rules, err := h.repo.List(r.Context(), params)
	if err != nil {
		httperr.WriteError(w, err)
		return
	}

	resp := make([]dto.RuleResponse, len(rules))
	for i, rule := range rules {
		resp[i] = dto.RuleToResponse(rule)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"rules": resp})
}

// Create handles POST /rules — validates input, stores rule, returns 201.
func (h *RulesHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httperr.WriteError(w, err)
		return
	}

	rule, err := req.ToDomain()
	if err != nil {
		httperr.WriteError(w, err)
		return
	}

	created, err := h.repo.Create(r.Context(), rule)
	if err != nil {
		httperr.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dto.RuleToResponse(created))
}

// Get handles GET /rules/{id}.
func (h *RulesHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httperr.WriteError(w, domain.ErrValidation)
		return
	}

	rule, err := h.repo.Get(r.Context(), id)
	if err != nil {
		httperr.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dto.RuleToResponse(rule))
}

// Update handles PATCH /rules/{id} with optional requests_allowed/window_seconds fields.
func (h *RulesHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httperr.WriteError(w, domain.ErrValidation)
		return
	}

	var req dto.UpdateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httperr.WriteError(w, err)
		return
	}

	if req.RequestsAllowed == nil && req.WindowSeconds == nil {
		httperr.WriteError(w, fmt.Errorf("%w: no fields to update", domain.ErrValidation))
		return
	}

	params := repository.RateRuleUpdate{
		RequestsAllowed: req.RequestsAllowed,
		WindowSeconds:   req.WindowSeconds,
	}
	if params.RequestsAllowed != nil && *params.RequestsAllowed <= 0 {
		httperr.WriteError(w, domain.ErrValidation)
		return
	}
	if params.WindowSeconds != nil && (*params.WindowSeconds <= 0 || *params.WindowSeconds > 86400) {
		httperr.WriteError(w, domain.ErrValidation)
		return
	}

	updated, err := h.repo.Update(r.Context(), id, params)
	if err != nil {
		httperr.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dto.RuleToResponse(updated))
}
