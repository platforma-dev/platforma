package auth

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
)

type db interface {
	NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error)
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type Repository struct {
	db db
}

func NewRepository(db db) *Repository {
	return &Repository{
		db: db,
	}
}

//go:embed migrations/*.sql
var migrations embed.FS

func (r *Repository) Migrations() fs.FS {
	m, _ := fs.Sub(migrations, "migrations")
	return m
}

func (r *Repository) Get(ctx context.Context, id string) (*User, error) {
	var user User
	err := r.db.GetContext(ctx, &user, "SELECT * FROM users WHERE id = $1", id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return &user, nil
}

func (r *Repository) GetByUsername(ctx context.Context, username string) (*User, error) {
	var user User
	err := r.db.GetContext(ctx, &user, "SELECT * FROM users WHERE username = $1", username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	return &user, nil
}

func (r *Repository) Create(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (id, username, password, salt, created, updated, status)
		VALUES (:id, :username, :password, :salt, :created, :updated, :status)
	`
	_, err := r.db.NamedExecContext(ctx, query, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *Repository) UpdatePassword(ctx context.Context, id, password, salt string) error {
	query := `
		UPDATE users
		SET password = $1, salt = $2, updated = CURRENT_TIMESTAMP
		WHERE id = $3
	`
	_, err := r.db.ExecContext(ctx, query, password, salt, id)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}
