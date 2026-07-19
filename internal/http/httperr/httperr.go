// Package httperr maps domain errors to HTTP status codes and writes JSON error responses.
package httperr

import (
	"encoding/json"
	"errors"
	"net/http"

	"sentinel/internal/domain"
	"sentinel/internal/http/dto"
)

// WriteError maps a domain-level error to an HTTP status code and writes a JSON error body.
func WriteError(w http.ResponseWriter, err error) {
	var status int
	switch {
	case errors.Is(err, domain.ErrValidation):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrConflict):
		status = http.StatusConflict
	default:
		status = http.StatusInternalServerError
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(dto.ErrorResponse{Error: err.Error()})
}
