package dto

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"sentinel/internal/domain"
)

type CreateClientRequest struct {
	Name string `json:"name"`
}

type UpdateClientRequest struct {
	Name   *string `json:"name"`
	Status *string `json:"status"`
}

type CreateRuleRequest struct {
	ClientID        string `json:"client_id"`
	API             string `json:"api"`
	RequestsAllowed int    `json:"requests_allowed"`
	WindowSeconds   int    `json:"window_seconds"`
}

type UpdateRuleRequest struct {
	RequestsAllowed *int `json:"requests_allowed"`
	WindowSeconds   *int `json:"window_seconds"`
}

type ClientResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type RuleResponse struct {
	ID              string `json:"id"`
	ClientID        string `json:"client_id"`
	API             string `json:"api"`
	RequestsAllowed int    `json:"requests_allowed"`
	WindowSeconds   int    `json:"window_seconds"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type CheckRequest struct {
	ClientID string `json:"client_id"`
	API      string `json:"api"`
}

type CheckAllowedResponse struct {
	Allowed   bool `json:"allowed"`
	Remaining int  `json:"remaining"`
}

type CheckRejectedResponse struct {
	Allowed    bool `json:"allowed"`
	RetryAfter int  `json:"retry_after"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func ClientToResponse(c domain.Client) ClientResponse {
	return ClientResponse{
		ID:        c.ID.String(),
		Name:      c.Name,
		Status:    string(c.Status),
		CreatedAt: formatTime(c.CreatedAt),
		UpdatedAt: formatTime(c.UpdatedAt),
	}
}

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
