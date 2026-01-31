// Package database provides database connection and migration functionality.
package database

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// Database represents a database connection with migration capabilities.
type Database struct {
	conn         *sqlx.DB
	repositories map[string]any
	migrators    map[string]migrator
	service      *service
}

// New creates a new Database instance with the given connection string.
func New(connection string) (*Database, error) {
	db, err := sqlx.Connect("postgres", connection)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	repository := newRepository(db)
	service := newService(repository)
	return &Database{conn: db, repositories: make(map[string]any), migrators: make(map[string]migrator), service: service}, nil
}

// Connection returns the underlying sqlx database connection.
func (db *Database) Connection() *sqlx.DB {
	return db.conn
}

// RegisterRepository registers a repository in the database.
// If repository implements migrator interface, it will migrate when `Migrate` is called.
func (db *Database) RegisterRepository(name string, repository any) {
	db.repositories[name] = repository

	if migr, ok := repository.(migrator); ok {
		db.migrators[name] = migr
	}
}

// Migrate runs all pending migrations for registered repositories.
func (db *Database) Migrate(ctx context.Context) error {
	// Ensure that migration table exists
	err := db.service.migrateSelf(ctx)
	if err != nil {
		return err
	}

	// Get completed migrations
	migrationLogs, err := db.service.getMigrationLogs(ctx)
	if err != nil {
		return fmt.Errorf("failed to select migrations state: %w", err)
	}

	// Get migrations from all migrators
	migrations := []Migration{}
	for name, migrator := range db.migrators {
		parsed, err := ParseMigrations(migrator.Migrations())
		if err != nil {
			return fmt.Errorf("failed to parse migrations for %s: %w", name, err)
		}
		for _, migr := range parsed {
			migr.repository = name
			migrations = append(migrations, migr)
		}
	}

	err = db.service.applyMigrations(ctx, migrations, migrationLogs)
	if err != nil {
		return err
	}

	return nil
}
