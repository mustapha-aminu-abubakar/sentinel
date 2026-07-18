package postgres

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"sentinel/internal/domain"
	"sentinel/internal/repository"
	"sentinel/internal/repository/db"
)

type RateRuleRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewRateRuleRepository(pool *pgxpool.Pool) *RateRuleRepository {
	return &RateRuleRepository{
		pool: pool,
		q:    db.New(pool),
	}
}

func (r *RateRuleRepository) Create(ctx context.Context, rule domain.RateRule) (domain.RateRule, error) {
	if err := domain.ValidateRateRule(rule); err != nil {
		return domain.RateRule{}, err
	}

	created, err := r.q.CreateRateRule(ctx, db.CreateRateRuleParams{
		ID:              rule.ID,
        ClientID:        pgtype.UUID{Bytes: [16]byte(rule.ClientID), Valid: true},
		Api:             rule.API,
		RequestsAllowed: int32(rule.RequestsAllowed),
		WindowSeconds:   int32(rule.WindowSeconds),
	})
	if err != nil {
		return domain.RateRule{}, fmt.Errorf("create rate rule: %w", mapPgError(err))
	}

	return toDomainRateRule(created), nil
}

func (r *RateRuleRepository) Get(ctx context.Context, id uuid.UUID) (domain.RateRule, error) {
	rule, err := r.q.GetRateRule(ctx, id)
	if err != nil {
		return domain.RateRule{}, fmt.Errorf("get rate rule: %w", mapPgError(err))
	}
	return toDomainRateRule(rule), nil
}

func (r *RateRuleRepository) ListByClient(ctx context.Context, clientID uuid.UUID) ([]domain.RateRule, error) {
	rows, err := r.q.ListRateRulesByClient(ctx, pgtype.UUID{Bytes: [16]byte(clientID), Valid: true})
	if err != nil {
		return nil, fmt.Errorf("list rate rules by client: %w", mapPgError(err))
	}

	result := make([]domain.RateRule, len(rows))
	for i, row := range rows {
		result[i] = toDomainRateRule(row)
	}
	return result, nil
}

func (r *RateRuleRepository) List(ctx context.Context, params repository.ListRulesParams) ([]domain.RateRule, error) {
	if params.Limit < 0 || int64(params.Limit) > math.MaxInt32 {
		return nil, fmt.Errorf("%w: limit out of range", domain.ErrValidation)
	}
	if params.Offset < 0 || int64(params.Offset) > math.MaxInt32 {
		return nil, fmt.Errorf("%w: offset out of range", domain.ErrValidation)
	}

	var clientID pgtype.UUID
	if params.ClientID != nil {
		clientID = pgtype.UUID{Bytes: [16]byte(*params.ClientID), Valid: true}
	}

	var api pgtype.Text
	if params.API != nil {
		api = pgtype.Text{String: *params.API, Valid: true}
	}

	rows, err := r.q.ListRateRules(ctx, db.ListRateRulesParams{
		ClientID: clientID,
		Api:      api,
		Limit:    int32(params.Limit),
		Offset:   int32(params.Offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list rate rules: %w", mapPgError(err))
	}

	result := make([]domain.RateRule, len(rows))
	for i, row := range rows {
		result[i] = toDomainRateRule(row)
	}
	return result, nil
}

func (r *RateRuleRepository) GetByClientAndAPI(ctx context.Context, clientID uuid.UUID, api string) (domain.RateRule, error) {
	rule, err := r.q.GetRateRuleByClientAndAPI(ctx, db.GetRateRuleByClientAndAPIParams{
		ClientID: pgtype.UUID{Bytes: [16]byte(clientID), Valid: true},
		Api:      api,
	})
	if err != nil {
		return domain.RateRule{}, fmt.Errorf("get rate rule by client and api: %w", mapPgError(err))
	}
	return toDomainRateRule(rule), nil
}

func (r *RateRuleRepository) Update(ctx context.Context, id uuid.UUID, params repository.RateRuleUpdate) (domain.RateRule, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.RateRule{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qt := r.q.WithTx(tx)

	current, err := qt.GetRateRuleForUpdate(ctx, id)
	if err != nil {
		return domain.RateRule{}, fmt.Errorf("update rate rule get: %w", mapPgError(err))
	}

	requestsAllowed := int(current.RequestsAllowed)
	if params.RequestsAllowed != nil {
		requestsAllowed = *params.RequestsAllowed
	}

	windowSeconds := int(current.WindowSeconds)
	if params.WindowSeconds != nil {
		windowSeconds = *params.WindowSeconds
	}

	merged := domain.RateRule{
		ID:              id,
		ClientID:        uuid.UUID(current.ClientID.Bytes),
		API:             current.Api,
		RequestsAllowed: requestsAllowed,
		WindowSeconds:   windowSeconds,
	}
	if err := domain.ValidateRateRule(merged); err != nil {
		return domain.RateRule{}, err
	}

	updated, err := qt.UpdateRateRule(ctx, db.UpdateRateRuleParams{
		ID:              id,
		RequestsAllowed: int32(requestsAllowed),
		WindowSeconds:   int32(windowSeconds),
	})
	if err != nil {
		return domain.RateRule{}, fmt.Errorf("update rate rule: %w", mapPgError(err))
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.RateRule{}, fmt.Errorf("commit tx: %w", err)
	}

	return toDomainRateRule(updated), nil
}

func toDomainRateRule(rr db.RateRule) domain.RateRule {
	return domain.RateRule{
		ID:              rr.ID,
		ClientID:        uuid.UUID(rr.ClientID.Bytes),
		API:             rr.Api,
		RequestsAllowed: int(rr.RequestsAllowed),
		WindowSeconds:   int(rr.WindowSeconds),
		CreatedAt:       rr.CreatedAt.Time,
		UpdatedAt:       rr.UpdatedAt.Time,
	}
}
