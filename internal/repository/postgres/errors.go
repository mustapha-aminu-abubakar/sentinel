// Package postgres implements repository interfaces against a Postgres database using sqlc-generated queries.
package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"sentinel/internal/domain"
)

// mapPgError converts Postgres and pgx errors into domain sentinel errors.
func mapPgError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case "23505":
		return domain.ErrConflict
	case "23514", "23503":
		return domain.ErrValidation
	default:
		return err
	}
}
