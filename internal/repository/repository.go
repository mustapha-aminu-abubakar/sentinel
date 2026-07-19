// Package repository defines interfaces and parameter types for client and rate-rule data access.
package repository

import (
	"context"

	"github.com/google/uuid"

	"sentinel/internal/domain"
)

// ClientRepository defines the persistence contract for clients.
type ClientRepository interface {
	// Create inserts a new client and returns it with server-assigned timestamps.
	Create(ctx context.Context, client domain.Client) (domain.Client, error)
	// Get retrieves a client by its unique ID.
	Get(ctx context.Context, id uuid.UUID) (domain.Client, error)
	// List returns clients matching the provided filter parameters.
	List(ctx context.Context, params ListClientsParams) ([]domain.Client, error)
	// Update applies partial updates to an existing client.
	Update(ctx context.Context, id uuid.UUID, params ClientUpdate) (domain.Client, error)
	// Deactivate sets a client's status to inactive.
	Deactivate(ctx context.Context, id uuid.UUID) (domain.Client, error)
}

// RateRuleRepository defines the persistence contract for rate rules.
type RateRuleRepository interface {
	// Create inserts a new rate rule with validation.
	Create(ctx context.Context, rule domain.RateRule) (domain.RateRule, error)
	// Get retrieves a rate rule by its unique ID.
	Get(ctx context.Context, id uuid.UUID) (domain.RateRule, error)
	// ListByClient returns all rate rules for a given client.
	ListByClient(ctx context.Context, clientID uuid.UUID) ([]domain.RateRule, error)
	// List returns rules matching the provided filter parameters.
	List(ctx context.Context, params ListRulesParams) ([]domain.RateRule, error)
	// GetByClientAndAPI returns the rule for a specific client and API combination.
	GetByClientAndAPI(ctx context.Context, clientID uuid.UUID, api string) (domain.RateRule, error)
	// Update applies partial updates to an existing rate rule.
	Update(ctx context.Context, id uuid.UUID, params RateRuleUpdate) (domain.RateRule, error)
}

// ListClientsParams carries optional filters and pagination for listing clients.
type ListClientsParams struct {
	Status *domain.ClientStatus
	Limit  int
	Offset int
}

// ListRulesParams carries optional filters and pagination for listing rate rules.
type ListRulesParams struct {
	ClientID *uuid.UUID
	API      *string
	Limit    int
	Offset   int
}

// ClientUpdate carries optional fields for partially updating a client.
type ClientUpdate struct {
	Name   *string
	Status *domain.ClientStatus
}

// RateRuleUpdate carries optional fields for partially updating a rate rule.
type RateRuleUpdate struct {
	RequestsAllowed *int
	WindowSeconds   *int
}
