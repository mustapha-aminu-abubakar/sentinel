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

// ClientRepository implements repository.ClientRepository using sqlc-generated queries on pgxpool.
type ClientRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// NewClientRepository creates a new Postgres-backed ClientRepository.
func NewClientRepository(pool *pgxpool.Pool) *ClientRepository {
	return &ClientRepository{
		pool: pool,
		q:    db.New(pool),
	}
}

// Create inserts a new client after validation and returns the result.
func (r *ClientRepository) Create(ctx context.Context, client domain.Client) (domain.Client, error) {
	if err := domain.ValidateClientName(client.Name); err != nil {
		return domain.Client{}, err
	}
	if err := domain.ValidateClientStatus(client.Status); err != nil {
		return domain.Client{}, err
	}

	now, err := r.q.CreateClient(ctx, db.CreateClientParams{
		ID:     client.ID,
		Name:   client.Name,
		Status: string(client.Status),
	})
	if err != nil {
		return domain.Client{}, fmt.Errorf("create client: %w", mapPgError(err))
	}

	return toDomainClient(now), nil
}

// Get retrieves a client by ID.
func (r *ClientRepository) Get(ctx context.Context, id uuid.UUID) (domain.Client, error) {
	c, err := r.q.GetClient(ctx, id)
	if err != nil {
		return domain.Client{}, fmt.Errorf("get client: %w", mapPgError(err))
	}
	return toDomainClient(c), nil
}

// List returns clients filtered by optional status with pagination.
func (r *ClientRepository) List(ctx context.Context, params repository.ListClientsParams) ([]domain.Client, error) {
	if params.Limit < 0 || int64(params.Limit) > math.MaxInt32 {
		return nil, fmt.Errorf("%w: limit out of range", domain.ErrValidation)
	}
	if params.Offset < 0 || int64(params.Offset) > math.MaxInt32 {
		return nil, fmt.Errorf("%w: offset out of range", domain.ErrValidation)
	}

	var status pgtype.Text
	if params.Status != nil {
		status = pgtype.Text{String: string(*params.Status), Valid: true}
	}

	rows, err := r.q.ListClients(ctx, db.ListClientsParams{
		Status: status,
		Limit:  int32(params.Limit),
		Offset: int32(params.Offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list clients: %w", mapPgError(err))
	}

	result := make([]domain.Client, len(rows))
	for i, row := range rows {
		result[i] = toDomainClient(row)
	}
	return result, nil
}

// Update applies partial updates to a client using a read-modify-write transaction.
func (r *ClientRepository) Update(ctx context.Context, id uuid.UUID, params repository.ClientUpdate) (domain.Client, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Client{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qt := r.q.WithTx(tx)

	current, err := qt.GetClientForUpdate(ctx, id)
	if err != nil {
		return domain.Client{}, fmt.Errorf("update client get: %w", mapPgError(err))
	}

	name := current.Name
	if params.Name != nil {
		name = *params.Name
	}

	status := current.Status
	if params.Status != nil {
		status = string(*params.Status)
	}

	if params.Name != nil {
		if err := domain.ValidateClientName(*params.Name); err != nil {
			return domain.Client{}, err
		}
	}
	if params.Status != nil {
		if err := domain.ValidateClientStatus(*params.Status); err != nil {
			return domain.Client{}, err
		}
	}

	updated, err := qt.UpdateClient(ctx, db.UpdateClientParams{
		ID:     id,
		Name:   name,
		Status: status,
	})
	if err != nil {
		return domain.Client{}, fmt.Errorf("update client: %w", mapPgError(err))
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Client{}, fmt.Errorf("commit tx: %w", err)
	}

	return toDomainClient(updated), nil
}

// Deactivate sets a client's status to inactive.
func (r *ClientRepository) Deactivate(ctx context.Context, id uuid.UUID) (domain.Client, error) {
	updated, err := r.q.DeactivateClient(ctx, id)
	if err != nil {
		return domain.Client{}, fmt.Errorf("deactivate client: %w", mapPgError(err))
	}
	return toDomainClient(updated), nil
}

func toDomainClient(c db.Client) domain.Client {
	return domain.Client{
		ID:        c.ID,
		Name:      c.Name,
		Status:    domain.ClientStatus(c.Status),
		CreatedAt: c.CreatedAt.Time,
		UpdatedAt: c.UpdatedAt.Time,
	}
}
