// Package handlers implements HTTP handlers for the rate-limiting API.
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

// ClientsHandler serves client CRUD endpoints backed by a repository.ClientRepository.
type ClientsHandler struct {
	repo repository.ClientRepository
}

// NewClientsHandler creates a ClientsHandler with the given repository.
func NewClientsHandler(repo repository.ClientRepository) *ClientsHandler {
	return &ClientsHandler{repo: repo}
}

// List handles GET /clients with optional ?status=, ?limit=, and ?offset= filters.
func (h *ClientsHandler) List(w http.ResponseWriter, r *http.Request) {
	params := repository.ListClientsParams{
		Limit:  100,
		Offset: 0,
	}
	if statusFilter := r.URL.Query().Get("status"); statusFilter != "" {
		s := domain.ClientStatus(statusFilter)
		if err := domain.ValidateClientStatus(s); err != nil {
			httperr.WriteError(w, err)
			return
		}
		params.Status = &s
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if _, err := fmt.Sscanf(limitStr, "%d", &params.Limit); err != nil {
			httperr.WriteError(w, domain.ErrValidation)
			return
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if _, err := fmt.Sscanf(offsetStr, "%d", &params.Offset); err != nil {
			httperr.WriteError(w, domain.ErrValidation)
			return
		}
	}

	clients, err := h.repo.List(r.Context(), params)
	if err != nil {
		httperr.WriteError(w, err)
		return
	}

	resp := make([]dto.ClientResponse, len(clients))
	for i, c := range clients {
		resp[i] = dto.ClientToResponse(c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"clients": resp})
}

// Create handles POST /clients — validates input, stores client, returns 201.
func (h *ClientsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httperr.WriteError(w, err)
		return
	}

	client, err := req.ToDomain()
	if err != nil {
		httperr.WriteError(w, err)
		return
	}

	created, err := h.repo.Create(r.Context(), client)
	if err != nil {
		httperr.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dto.ClientToResponse(created))
}

// Get handles GET /clients/{id}.
func (h *ClientsHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httperr.WriteError(w, domain.ErrValidation)
		return
	}

	client, err := h.repo.Get(r.Context(), id)
	if err != nil {
		httperr.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dto.ClientToResponse(client))
}

// Update handles PATCH /clients/{id} with optional name/status fields.
func (h *ClientsHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httperr.WriteError(w, domain.ErrValidation)
		return
	}

	var req dto.UpdateClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httperr.WriteError(w, err)
		return
	}

	params := repository.ClientUpdate{}
	if req.Name != nil {
		if err := domain.ValidateClientName(*req.Name); err != nil {
			httperr.WriteError(w, err)
			return
		}
		params.Name = req.Name
	}
	if req.Status != nil {
		s := domain.ClientStatus(*req.Status)
		if err := domain.ValidateClientStatus(s); err != nil {
			httperr.WriteError(w, err)
			return
		}
		params.Status = &s
	}

	if params.Name == nil && params.Status == nil {
		httperr.WriteError(w, fmt.Errorf("%w: no fields to update", domain.ErrValidation))
		return
	}

	updated, err := h.repo.Update(r.Context(), id, params)
	if err != nil {
		httperr.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dto.ClientToResponse(updated))
}
