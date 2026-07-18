package dbtest

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func StartPostgres() (*pgxpool.Pool, func(), error) {
	pool, _, teardown, err := startPostgres()
	if err != nil {
		return nil, nil, err
	}
	return pool, teardown, nil
}

func StartPostgresWithDSN() (*pgxpool.Pool, string, func(), error) {
	return startPostgres()
}

func startPostgres() (*pgxpool.Pool, string, func(), error) {
	dbName := "sentinel_test"
	dbPwd := "password"
	dbUser := "user"

	container, err := postgres.Run(
		context.Background(),
		"postgres:latest",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPwd),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		return nil, "", nil, fmt.Errorf("start postgres container: %w", err)
	}

	host, err := container.Host(context.Background())
	if err != nil {
		return nil, "", nil, fmt.Errorf("get container host: %w", err)
	}

	port, err := container.MappedPort(context.Background(), "5432/tcp")
	if err != nil {
		return nil, "", nil, fmt.Errorf("get container port: %w", err)
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=public",
		dbUser, dbPwd, host, port.Port(), dbName)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, "", nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, "", nil, fmt.Errorf("ping pool: %w", err)
	}

	teardown := func() {
		pool.Close()
		if err := container.Terminate(context.Background()); err != nil {
			log.Printf("terminate container: %v", err)
		}
	}

	return pool, dsn, teardown, nil
}
