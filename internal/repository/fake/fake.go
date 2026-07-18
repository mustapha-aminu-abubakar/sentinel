package fake

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"sentinel/internal/domain"
	"sentinel/internal/repository"
)

type ClientRepository struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]domain.Client
}

func NewClientRepository() *ClientRepository {
	return &ClientRepository{
		clients: make(map[uuid.UUID]domain.Client),
	}
}

func (r *ClientRepository) Create(ctx context.Context, client domain.Client) (domain.Client, error) {
	if err := domain.ValidateClientName(client.Name); err != nil {
		return domain.Client{}, err
	}
	if err := domain.ValidateClientStatus(client.Status); err != nil {
		return domain.Client{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.clients[client.ID]; exists {
		return domain.Client{}, fmt.Errorf("%w: client already exists", domain.ErrConflict)
	}

	now := time.Now().UTC()
	client.CreatedAt = now
	client.UpdatedAt = now
	r.clients[client.ID] = client
	return client, nil
}

func (r *ClientRepository) Get(ctx context.Context, id uuid.UUID) (domain.Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	c, ok := r.clients[id]
	if !ok {
		return domain.Client{}, fmt.Errorf("%w: client not found", domain.ErrNotFound)
	}
	return c, nil
}

func (r *ClientRepository) List(ctx context.Context, params repository.ListClientsParams) ([]domain.Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []domain.Client
	for _, c := range r.clients {
		if params.Status != nil && c.Status != *params.Status {
			continue
		}
		result = append(result, c)
	}
	if result == nil {
		return []domain.Client{}, nil
	}
	return result, nil
}

func (r *ClientRepository) Update(ctx context.Context, id uuid.UUID, params repository.ClientUpdate) (domain.Client, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	c, ok := r.clients[id]
	if !ok {
		return domain.Client{}, fmt.Errorf("%w: client not found", domain.ErrNotFound)
	}

	if params.Name != nil {
		if err := domain.ValidateClientName(*params.Name); err != nil {
			return domain.Client{}, err
		}
		c.Name = *params.Name
	}
	if params.Status != nil {
		if err := domain.ValidateClientStatus(*params.Status); err != nil {
			return domain.Client{}, err
		}
		c.Status = *params.Status
	}

	c.UpdatedAt = time.Now().UTC()
	r.clients[id] = c
	return c, nil
}

func (r *ClientRepository) Deactivate(ctx context.Context, id uuid.UUID) (domain.Client, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	c, ok := r.clients[id]
	if !ok {
		return domain.Client{}, fmt.Errorf("%w: client not found", domain.ErrNotFound)
	}

	c.Status = domain.ClientStatusInactive
	c.UpdatedAt = time.Now().UTC()
	r.clients[id] = c
	return c, nil
}

type RateRuleRepository struct {
	mu    sync.RWMutex
	rules map[uuid.UUID]domain.RateRule
}

func NewRateRuleRepository() *RateRuleRepository {
	return &RateRuleRepository{
		rules: make(map[uuid.UUID]domain.RateRule),
	}
}

func (r *RateRuleRepository) Create(ctx context.Context, rule domain.RateRule) (domain.RateRule, error) {
	if err := domain.ValidateRateRule(rule); err != nil {
		return domain.RateRule{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.rules {
		if existing.ClientID == rule.ClientID && existing.API == rule.API {
			return domain.RateRule{}, fmt.Errorf("%w: rate rule for client %s and api %s already exists", domain.ErrConflict, rule.ClientID, rule.API)
		}
	}

	now := time.Now().UTC()
	rule.CreatedAt = now
	rule.UpdatedAt = now
	r.rules[rule.ID] = rule
	return rule, nil
}

func (r *RateRuleRepository) Get(ctx context.Context, id uuid.UUID) (domain.RateRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rule, ok := r.rules[id]
	if !ok {
		return domain.RateRule{}, fmt.Errorf("%w: rate rule not found", domain.ErrNotFound)
	}
	return rule, nil
}

func (r *RateRuleRepository) ListByClient(ctx context.Context, clientID uuid.UUID) ([]domain.RateRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []domain.RateRule
	for _, rule := range r.rules {
		if rule.ClientID == clientID {
			result = append(result, rule)
		}
	}
	if result == nil {
		return []domain.RateRule{}, nil
	}
	return result, nil
}

func (r *RateRuleRepository) List(ctx context.Context, params repository.ListRulesParams) ([]domain.RateRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []domain.RateRule
	for _, rule := range r.rules {
		if params.ClientID != nil && rule.ClientID != *params.ClientID {
			continue
		}
		if params.API != nil && rule.API != *params.API {
			continue
		}
		result = append(result, rule)
	}
	if result == nil {
		return []domain.RateRule{}, nil
	}
	return result, nil
}

func (r *RateRuleRepository) GetByClientAndAPI(ctx context.Context, clientID uuid.UUID, api string) (domain.RateRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, rule := range r.rules {
		if rule.ClientID == clientID && rule.API == api {
			return rule, nil
		}
	}
	return domain.RateRule{}, fmt.Errorf("%w: rate rule not found", domain.ErrNotFound)
}

func (r *RateRuleRepository) Update(ctx context.Context, id uuid.UUID, params repository.RateRuleUpdate) (domain.RateRule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rule, ok := r.rules[id]
	if !ok {
		return domain.RateRule{}, fmt.Errorf("%w: rate rule not found", domain.ErrNotFound)
	}

	if params.RequestsAllowed != nil {
		rule.RequestsAllowed = *params.RequestsAllowed
	}
	if params.WindowSeconds != nil {
		rule.WindowSeconds = *params.WindowSeconds
	}

	merged := domain.RateRule{
		ID:              rule.ID,
		ClientID:        rule.ClientID,
		API:             rule.API,
		RequestsAllowed: rule.RequestsAllowed,
		WindowSeconds:   rule.WindowSeconds,
	}
	if err := domain.ValidateRateRule(merged); err != nil {
		return domain.RateRule{}, err
	}

	rule.UpdatedAt = time.Now().UTC()
	r.rules[id] = rule
	return rule, nil
}
