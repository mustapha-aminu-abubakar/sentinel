package repository

import (
	"context"

	"github.com/google/uuid"

	"sentinel/internal/domain"
)

type ClientRepository interface {
	Create(ctx context.Context, client domain.Client) (domain.Client, error)
	Get(ctx context.Context, id uuid.UUID) (domain.Client, error)
	List(ctx context.Context, params ListClientsParams) ([]domain.Client, error)
	Update(ctx context.Context, id uuid.UUID, params ClientUpdate) (domain.Client, error)
	Deactivate(ctx context.Context, id uuid.UUID) (domain.Client, error)
}

type RateRuleRepository interface {
	Create(ctx context.Context, rule domain.RateRule) (domain.RateRule, error)
	Get(ctx context.Context, id uuid.UUID) (domain.RateRule, error)
	ListByClient(ctx context.Context, clientID uuid.UUID) ([]domain.RateRule, error)
	List(ctx context.Context, params ListRulesParams) ([]domain.RateRule, error)
	GetByClientAndAPI(ctx context.Context, clientID uuid.UUID, api string) (domain.RateRule, error)
	Update(ctx context.Context, id uuid.UUID, params RateRuleUpdate) (domain.RateRule, error)
}

type ListClientsParams struct {
	Status *domain.ClientStatus
	Limit  int
	Offset int
}

type ListRulesParams struct {
	ClientID *uuid.UUID
	API      *string
	Limit    int
	Offset   int
}

type ClientUpdate struct {
	Name   *string
	Status *domain.ClientStatus
}

type RateRuleUpdate struct {
	RequestsAllowed *int
	WindowSeconds   *int
}
