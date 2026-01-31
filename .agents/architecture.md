# ARCHITECTURE

Domain-driven design patterns and navigation for Platforma.

## WHERE TO LOOK

Quick reference for common tasks:

| Task | Location | Notes |
|------|----------|-------|
| Add new domain | `internal/cli/templates/domain/` | Use `go run . generate domain <name>` |
| Implement Domain interface | `application/domain.go` | Must return `GetRepository() any` |
| Add HTTP routes | `httpserver/handlergroup.go` | `Handle("METHOD /path", handler)` |
| Add middleware | `httpserver/middleware.go` | Implement `Wrap(http.Handler) http.Handler` |
| Database migrations | Repository's `Migrations()` method | Returns `[]database.Migration` |
| Register with app | `application/application.go` | `RegisterDomain()`, `RegisterService()`, `RegisterDatabase()` |
| Background jobs | `queue/processor.go` | Implement `Handler[T]` interface |
| Scheduled tasks | `scheduler/scheduler.go` | Pass `Runner` + duration |
| Logging with context | `log/log.go` | Use `InfoContext(ctx, msg, args...)` |

## DOMAIN PATTERN

A domain aggregates all components for a bounded context:

```go
type Domain struct {
    Repository *Repository
    Service    *Service
    API        *httpserver.HandlerGroup
    Middleware httpserver.Middleware
}

func (d *Domain) GetRepository() any {
    return d.Repository
}
```

See `auth/domain.go` for complete implementation.

### Domain Components

| Component | Required | Purpose |
|-----------|----------|---------|
| Repository | Yes | Database operations, migrations |
| Service | Yes | Business logic |
| HandlerGroup | No | HTTP endpoints |
| Middleware | No | Request processing |

## APPLICATION LIFECYCLE

Register components with the application:

```go
app := application.New()

// Register database (runs migrations)
app.RegisterDatabase(db)

// Register domain (repository gets registered for migrations)
app.RegisterDomain(authDomain)

// Register standalone services
app.RegisterService(emailService)

// Start (runs healthchecks, starts HTTP server)
app.Run(ctx)
```

## HTTP ROUTING

Use `HandlerGroup` for route organization:

```go
api := httpserver.NewHandlerGroup()
api.Handle("POST /users", createUserHandler)
api.Handle("GET /users/{id}", getUserHandler)
api.Handle("DELETE /users/{id}", deleteUserHandler)

// Apply middleware to group
api.Use(authMiddleware)

// Mount to server
server.Handle("/api", api)
```

## DATABASE MIGRATIONS

Migrations are written in `.sql` files and embedded via `fs.FS`. Each file contains up and down scripts separated by markers:

```sql
-- 001_create_users.sql
-- +migrate Up
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email TEXT UNIQUE NOT NULL
);

-- +migrate Down
DROP TABLE users;
```

Repository returns embedded migrations:

```go
//go:embed *.sql
var migrations embed.FS

func (r *Repository) Migrations() fs.FS {
    return migrations
}
```

Migration files are sorted lexicographically by filename. Use numeric prefixes for ordering: `001_`, `002_`, etc.

## BACKGROUND JOBS

Implement the queue handler interface:

```go
type EmailHandler struct{}

func (h *EmailHandler) Handle(ctx context.Context, job EmailJob) error {
    // Process job
    return nil
}

// Register with processor
processor := queue.NewProcessor(provider, &EmailHandler{})
```

## PACKAGE REFERENCE

| Package | Key Types | Role |
|---------|-----------|------|
| `application` | `Application`, `Domain`, `Runner`, `Healthchecker` | Lifecycle orchestration |
| `httpserver` | `HTTPServer`, `HandlerGroup`, `Middleware` | HTTP layer |
| `database` | `Database`, `Migration` | PostgreSQL + migrations |
| `queue` | `Processor[T]`, `Handler[T]`, `Provider[T]` | Job processing |
| `scheduler` | `Scheduler` | Periodic execution |
| `log` | `Logger`, context keys | Structured logging |
