// Package dto provides request/response types and conversion helpers for the HTTP API.
package dto

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"sentinel/internal/domain"
)

// CreateClientRequest is the JSON body for POST /clients.
type CreateClientRequest struct {
	Name string `json:"name"`
}

// UpdateClientRequest is the JSON body for PATCH /clients/{id}.
type UpdateClientRequest struct {
	Name   *string `json:"name"`
	Status *string `json:"status"`
}

// CreateRuleRequest is the JSON body for POST /rules.
type CreateRuleRequest struct {
	ClientID        string `json:"client_id"`
	API             string `json:"api"`
	RequestsAllowed int    `json:"requests_allowed"`
	WindowSeconds   int    `json:"window_seconds"`
}

// UpdateRuleRequest is the JSON body for PATCH /rules/{id}.
type UpdateRuleRequest struct {
	RequestsAllowed *int `json:"requests_allowed"`
	WindowSeconds   *int `json:"window_seconds"`
}

// ClientResponse is the JSON representation of a client.
type ClientResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// RuleResponse is the JSON representation of a rate rule.
type RuleResponse struct {
	ID              string `json:"id"`
	ClientID        string `json:"client_id"`
	API             string `json:"api"`
	RequestsAllowed int    `json:"requests_allowed"`
	WindowSeconds   int    `json:"window_seconds"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// CheckRequest is the JSON body for POST /v1/check.
type CheckRequest struct {
	ClientID string `json:"client_id"`
	API      string `json:"api"`
}

// CheckAllowedResponse is returned when a rate-limit check passes.
type CheckAllowedResponse struct {
	Allowed   bool `json:"allowed"`
	Remaining int  `json:"remaining"`
}

// CheckRejectedResponse is returned when a rate-limit check fails.
type CheckRejectedResponse struct {
	Allowed    bool `json:"allowed"`
	RetryAfter int  `json:"retry_after"`
}

// ErrorResponse is the JSON body for error responses.
type ErrorResponse struct {
	Error string `json:"error"`
}

// formatTime formats a time.Time as RFC 3339 in UTC.
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// ClientToResponse converts a domain.Client into its JSON response form.
func ClientToResponse(c domain.Client) ClientResponse {
	return ClientResponse{
		ID:        c.ID.String(),
		Name:      c.Name,
		Status:    string(c.Status),
		CreatedAt: formatTime(c.CreatedAt),
		UpdatedAt: formatTime(c.UpdatedAt),
	}
}

// RuleToResponse converts a domain.RateRule into its JSON response form.
func RuleToResponse(r domain.RateRule) RuleResponse {
	return RuleResponse{
		ID:              r.ID.String(),
		ClientID:        r.ClientID.String(),
		API:             r.API,
		RequestsAllowed: r.RequestsAllowed,
		WindowSeconds:   r.WindowSeconds,
		CreatedAt:       formatTime(r.CreatedAt),
		UpdatedAt:       formatTime(r.UpdatedAt),
	}
}

// ToDomain validates and converts an API create request into a domain.Client.
func (r CreateClientRequest) ToDomain() (domain.Client, error) {
	if err := domain.ValidateClientName(r.Name); err != nil {
		return domain.Client{}, err
	}
	return domain.Client{
		ID:     uuid.New(),
		Name:   r.Name,
		Status: domain.ClientStatusActive,
	}, nil
}

// ToDomain validates and converts an API create rule request into a domain.RateRule.
func (r CreateRuleRequest) ToDomain() (domain.RateRule, error) {
	clientID, err := uuid.Parse(r.ClientID)
	if err != nil {
		return domain.RateRule{}, fmt.Errorf("%w: %w", domain.ErrValidation, err)
	}
	rule := domain.RateRule{
		ID:              uuid.New(),
		ClientID:        clientID,
		API:             r.API,
		RequestsAllowed: r.RequestsAllowed,
		WindowSeconds:   r.WindowSeconds,
	}
	if err := domain.ValidateRateRule(rule); err != nil {
		return domain.RateRule{}, err
	}
	return rule, nil
}
