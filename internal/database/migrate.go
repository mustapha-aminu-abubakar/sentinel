package database

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/pressly/goose/v3"
)

func RunMigrations(db *sql.DB, migrationsDir string) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose set dialect: %w", err)
	}
	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	log.Println("migrations applied successfully")
	return nil
}
