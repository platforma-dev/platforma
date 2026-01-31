package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/platforma-dev/platforma/database"
	"github.com/platforma-dev/platforma/log"
)

// User represents a user in our system
type User struct {
	ID    int    `db:"id"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

// UserRepository handles database operations for users
type UserRepository struct {
	db *sqlx.DB
}

// NewUserRepository creates a new UserRepository with the given connection
func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

//go:embed *.sql
var migrations embed.FS

func (r *UserRepository) Migrations() fs.FS {
	return migrations
}

// Create inserts a new user into the database
func (r *UserRepository) Create(ctx context.Context, name, email string) (User, error) {
	var user User
	err := r.db.QueryRowxContext(ctx,
		"INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name, email",
		name, email,
	).StructScan(&user)
	return user, err
}

// GetAll retrieves all users from the database
func (r *UserRepository) GetAll(ctx context.Context) ([]User, error) {
	var users []User
	err := r.db.SelectContext(ctx, &users, "SELECT id, name, email FROM users")
	return users, err
}

func main() {
	ctx := context.Background()

	// Get database connection string from environment variable
	// Example: "postgres://user:password@localhost:5432/mydb?sslmode=disable"
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.ErrorContext(ctx, "DATABASE_URL environment variable is not set")
		os.Exit(1)
	}

	// Create new database connection
	db, err := database.New(connStr)
	if err != nil {
		log.ErrorContext(ctx, "failed to connect to database", "error", err)
		os.Exit(1)
	}

	log.InfoContext(ctx, "connected to database")

	// Create repository and register it with the database
	userRepo := NewUserRepository(db.Connection())
	db.RegisterRepository("users", userRepo)

	// Run migrations
	err = db.Migrate(ctx)
	if err != nil {
		log.ErrorContext(ctx, "failed to run migrations", "error", err)
		os.Exit(1)
	}

	log.InfoContext(ctx, "migrations completed successfully")

	// Create a new user
	user, err := userRepo.Create(ctx, "John Doe", "john@example.com")
	if err != nil {
		log.ErrorContext(ctx, "failed to create user", "error", err)
		os.Exit(1)
	}

	log.InfoContext(ctx, "user created", "id", user.ID, "name", user.Name, "email", user.Email)

	// Get all users
	users, err := userRepo.GetAll(ctx)
	if err != nil {
		log.ErrorContext(ctx, "failed to get users", "error", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d user(s):\n", len(users))
	for _, u := range users {
		fmt.Printf("  - ID: %d, Name: %s, Email: %s\n", u.ID, u.Name, u.Email)
	}
}
