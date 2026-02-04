# PLATFORMA

Go web framework with domain-driven architecture. Module: `github.com/platforma-dev/platforma`

## ESSENTIALS

- **Go 1.25.0** required
- **Task runner**: `task lint`, `task test`, `task check`
- **PostgreSQL driver**: `lib/pq` (not pgx)

## STRUCTURE

```
platforma/
├── application/     # Core lifecycle: Application, Domain interface
├── auth/            # Auth domain (see auth/AGENTS.md)
├── session/         # Session domain
├── httpserver/      # HTTP server with middleware
├── database/        # PostgreSQL with sqlx, migrations
├── queue/           # Job processor
├── scheduler/       # Periodic tasks
├── log/             # Structured logging
└── demo-app/        # Examples (excluded from linting)
```

## ANTI-PATTERNS

- **No testify** - use standard library assertions
- **No external mocking libraries** - hand-roll mocks per test
- **No global state** - except `log.Logger`
- **No init functions** - `gochecknoinits` enforced

## DEEP DIVES

- [Go Conventions](.agents/go-conventions.md) - JSON tags, error handling, interfaces, domains
- [Testing](.agents/testing.md) - Test patterns, mocking, integration tests
- [Linting](.agents/linting.md) - Strict linter rules and rationale
- [Architecture](.agents/architecture.md) - Domain patterns, task-to-file mapping
