//go:build linux

package database_test

import (
	"context"
	"io/fs"
	"slices"
	"testing"
	"testing/fstest"
	"time"

	"github.com/platforma-dev/platforma/database"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestMigrate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctr, err := postgres.Run(
		ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("hostamat"),
		postgres.WithUsername("hostamat"),
		postgres.WithPassword("hostamat"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("failed to initialize database: %s", err.Error())
	}

	err = ctr.Snapshot(ctx)
	if err != nil {
		t.Fatalf("failed to create snapshot: %s", err.Error())
	}

	dbURL, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %s", err.Error())
	}

	t.Logf("db connection string: %s", dbURL)

	t.Run("initialize and migrate empty database", func(t *testing.T) {
		t.Cleanup(func() {
			err = ctr.Restore(ctx)
			if err != nil {
				t.Fatalf("failed to restore db: %s", err.Error())
			}
		})

		db, err := database.New(dbURL)
		if err != nil {
			t.Fatalf("failed to initialize database: %s", err.Error())
		}

		if db == nil {
			t.Fatalf("database is nil")
		}

		err = db.Migrate(ctx)
		if err != nil {
			t.Fatalf("failed to migrate database: %s", err.Error())
		}

		var migrationLogs []migrationLog
		err = db.Connection().SelectContext(ctx, &migrationLogs, "SELECT * FROM platforma_migrations")
		if err != nil {
			t.Fatalf("expected no errors, got: %s", err.Error())
		}

		if len(migrationLogs) != 1 {
			t.Fatalf("expected single migration, got: %d", len(migrationLogs))
		}

		if migrationLogs[0].Repository != "platforma_migration" {
			t.Fatalf("expected repository to be platforma_migration, got: %s", migrationLogs[0].Repository)
		}

		if migrationLogs[0].MigrationID != "init" {
			t.Fatalf("expected migration id to be init, got: %s", migrationLogs[0].MigrationID)
		}
	})

	// This test imitates multiple application startups
	// If application startups multiple times, migrations should only run once
	t.Run("initialize and migrate empty database twice", func(t *testing.T) {
		t.Cleanup(func() {
			err = ctr.Restore(ctx)
			if err != nil {
				t.Fatalf("failed to restore db: %s", err.Error())
			}
		})

		db, err := database.New(dbURL)
		if err != nil {
			t.Fatalf("failed to initialize database: %s", err.Error())
		}

		if db == nil {
			t.Fatalf("database is nil")
		}

		// First migration
		err = db.Migrate(ctx)
		if err != nil {
			t.Fatalf("failed to migrate database: %s", err.Error())
		}

		// Second migration
		err = db.Migrate(ctx)
		if err != nil {
			t.Fatalf("failed to migrate database: %s", err.Error())
		}

		var migrationLogs []migrationLog
		err = db.Connection().SelectContext(ctx, &migrationLogs, "SELECT * FROM platforma_migrations")
		if err != nil {
			t.Fatalf("expected no errors, got: %s", err.Error())
		}

		if len(migrationLogs) != 1 {
			t.Fatalf("expected single migration, got: %d", len(migrationLogs))
		}

		if migrationLogs[0].Repository != "platforma_migration" {
			t.Fatalf("expected repository to be platforma_migration, got: %s", migrationLogs[0].Repository)
		}

		if migrationLogs[0].MigrationID != "init" {
			t.Fatalf("expected migration id to be init, got: %s", migrationLogs[0].MigrationID)
		}
	})

	t.Run("migrate database with single repository", func(t *testing.T) {
		t.Cleanup(func() {
			err = ctr.Restore(ctx)
			if err != nil {
				t.Fatalf("failed to restore db: %s", err.Error())
			}
		})

		db, err := database.New(dbURL)
		if err != nil {
			t.Fatalf("failed to initialize database: %s", err.Error())
		}

		if db == nil {
			t.Fatalf("database is nil")
		}

		db.RegisterRepository("some_repo", simpleRepo{fsys: migrationFS(database.Migration{
			ID:   "001_init",
			Up:   "CREATE TABLE IF NOT EXISTS simple_repo (id TEXT)",
			Down: "DROP TABLE simple_repo",
		})})

		err = db.Migrate(ctx)
		if err != nil {
			t.Fatalf("failed to migrate database: %s", err.Error())
		}

		var migrationLogs []migrationLog
		err = db.Connection().SelectContext(ctx, &migrationLogs, "SELECT * FROM platforma_migrations")
		if err != nil {
			t.Fatalf("expected no errors, got: %s", err.Error())
		}

		// 2 = platforma_migrations + simple_repo
		if len(migrationLogs) != 2 {
			t.Fatalf("expected 2 migrations, got: %d", len(migrationLogs))
		}

		if !slices.ContainsFunc(migrationLogs, func(log migrationLog) bool {
			return log.Repository == "some_repo" && log.MigrationID == "001_init"
		}) {
			t.Fatalf("expected migration log to contain init migration for some_repo")
		}

		_, err = db.Connection().ExecContext(ctx, "SELECT * FROM simple_repo")
		if err != nil {
			t.Fatalf("expected no errors, got: %s", err.Error())
		}
	})

	t.Run("migrate database with multiple repositories", func(t *testing.T) {
		t.Cleanup(func() {
			err = ctr.Restore(ctx)
			if err != nil {
				t.Fatalf("failed to restore db: %s", err.Error())
			}
		})

		db, err := database.New(dbURL)
		if err != nil {
			t.Fatalf("failed to initialize database: %s", err.Error())
		}

		if db == nil {
			t.Fatalf("database is nil")
		}

		db.RegisterRepository("some_repo", simpleRepo{fsys: migrationFS(database.Migration{
			ID:   "001_init",
			Up:   "CREATE TABLE IF NOT EXISTS simple_repo (id TEXT)",
			Down: "DROP TABLE simple_repo",
		})})

		db.RegisterRepository("other_repo", simpleRepo{fsys: migrationFS(database.Migration{
			ID:   "001_init",
			Up:   "CREATE TABLE IF NOT EXISTS other_repo (id TEXT)",
			Down: "DROP TABLE other_repo",
		})})

		err = db.Migrate(ctx)
		if err != nil {
			t.Fatalf("failed to migrate database: %s", err.Error())
		}

		var migrationLogs []migrationLog
		err = db.Connection().SelectContext(ctx, &migrationLogs, "SELECT * FROM platforma_migrations")
		if err != nil {
			t.Fatalf("expected no errors, got: %s", err.Error())
		}

		// 3 = platforma_migrations + repos
		if len(migrationLogs) != 3 {
			t.Fatalf("expected 3 migrations, got: %d", len(migrationLogs))
		}

		if !slices.ContainsFunc(migrationLogs, func(log migrationLog) bool {
			return log.Repository == "some_repo" && log.MigrationID == "001_init"
		}) {
			t.Fatalf("expected migration log to contain init migration for some_repo")
		}

		_, err = db.Connection().ExecContext(ctx, "SELECT * FROM simple_repo")
		if err != nil {
			t.Fatalf("expected no errors, got: %s", err.Error())
		}

		if !slices.ContainsFunc(migrationLogs, func(log migrationLog) bool {
			return log.Repository == "other_repo" && log.MigrationID == "001_init"
		}) {
			t.Fatalf("expected migration log to contain init migration for other_repo, but only got: %s", migrationLogs)
		}

		_, err = db.Connection().ExecContext(ctx, "SELECT * FROM other_repo")
		if err != nil {
			t.Fatalf("expected no errors, got: %s", err.Error())
		}
	})

	t.Run("migrate database with failing migration", func(t *testing.T) {
		t.Cleanup(func() {
			err = ctr.Restore(ctx)
			if err != nil {
				t.Fatalf("failed to restore db: %s", err.Error())
			}
		})

		db, err := database.New(dbURL)
		if err != nil {
			t.Fatalf("failed to initialize database: %s", err.Error())
		}

		if db == nil {
			t.Fatalf("database is nil")
		}

		db.RegisterRepository("some_repo", simpleRepo{fsys: migrationFS(database.Migration{
			ID:   "001_init",
			Up:   "CREATE TABLE IF NOT EXISTS simple_repo (id TEXT)",
			Down: "DROP TABLE simple_repo",
		})})

		db.RegisterRepository("other_repo", simpleRepo{fsys: migrationFS(
			database.Migration{
				ID:   "001_init",
				Up:   "CREATE TABLE IF NOT EXISTS other_repo (id TEXT)",
				Down: "DROP TABLE other_repo",
			},
			database.Migration{
				ID:   "002_failing",
				Up:   "not even SQL here",
				Down: "no need for this",
			},
		)})

		err = db.Migrate(ctx)
		if err == nil {
			t.Fatalf("migration expected to fail")
		}
		t.Logf("migration error: %s", err.Error())

		var migrationLogs []migrationLog
		err = db.Connection().SelectContext(ctx, &migrationLogs, "SELECT * FROM platforma_migrations")
		if err != nil {
			t.Fatalf("expected no errors, got: %s", err.Error())
		}

		if len(migrationLogs) != 1 {
			t.Fatalf("expected 1 migration, got: %d", len(migrationLogs))
		}

		if migrationLogs[0].Repository != "platforma_migration" {
			t.Fatalf("expected repository to be platforma_migration, got: %s", migrationLogs[0].Repository)
		}

		if migrationLogs[0].MigrationID != "init" {
			t.Fatalf("expected migration id to be init, got: %s", migrationLogs[0].MigrationID)
		}

		// because migration should be reverted
		if slices.ContainsFunc(migrationLogs, func(log migrationLog) bool {
			return log.Repository == "some_repo" && log.MigrationID == "001_init"
		}) {
			t.Fatalf("expected migration log to not contain init migration for some_repo")
		}

		_, err = db.Connection().ExecContext(ctx, "SELECT * FROM simple_repo")
		if err == nil {
			t.Fatalf("expected error, got nill")
		}

		// because migration should be reverted
		if slices.ContainsFunc(migrationLogs, func(log migrationLog) bool {
			return log.Repository == "other_repo" && log.MigrationID == "001_init"
		}) {
			t.Fatalf("expected migration log to not contain init migration for other_repo, but only got: %s", migrationLogs)
		}

		_, err = db.Connection().ExecContext(ctx, "SELECT * FROM other_repo")
		if err == nil {
			t.Fatalf("expected error, got nill")
		}
	})

	t.Run("migrate database with failing migration and revert", func(t *testing.T) {
		t.Cleanup(func() {
			err = ctr.Restore(ctx)
			if err != nil {
				t.Fatalf("failed to restore db: %s", err.Error())
			}
		})

		db, err := database.New(dbURL)
		if err != nil {
			t.Fatalf("failed to initialize database: %s", err.Error())
		}

		if db == nil {
			t.Fatalf("database is nil")
		}

		db.RegisterRepository("some_repo", simpleRepo{fsys: migrationFS(database.Migration{
			ID:   "001_init",
			Up:   "CREATE TABLE IF NOT EXISTS simple_repo (id TEXT)",
			Down: "broken SQL",
		})})

		db.RegisterRepository("other_repo", simpleRepo{fsys: migrationFS(
			database.Migration{
				ID:   "001_init",
				Up:   "CREATE TABLE IF NOT EXISTS other_repo (id TEXT)",
				Down: "DROP TABLE other_repo",
			},
			database.Migration{
				ID:   "002_failing",
				Up:   "not even SQL here",
				Down: "no need for this",
			},
		)})

		err = db.Migrate(ctx)
		if err == nil {
			t.Fatalf("migration expected to fail")
		}
		t.Logf("migration error: %s", err.Error())

		var migrationLogs []migrationLog
		err = db.Connection().SelectContext(ctx, &migrationLogs, "SELECT * FROM platforma_migrations")
		if err != nil {
			t.Fatalf("expected no errors, got: %s", err.Error())
		}

		if len(migrationLogs) != 1 {
			t.Fatalf("expected 1 migration, got: %d", len(migrationLogs))
		}

		if migrationLogs[0].Repository != "platforma_migration" {
			t.Fatalf("expected repository to be platforma_migration, got: %s", migrationLogs[0].Repository)
		}

		if migrationLogs[0].MigrationID != "init" {
			t.Fatalf("expected migration id to be init, got: %s", migrationLogs[0].MigrationID)
		}

		// because migration should be reverted or not even attempted
		if slices.ContainsFunc(migrationLogs, func(log migrationLog) bool {
			return log.Repository == "some_repo" && log.MigrationID == "001_init"
		}) {
			t.Fatalf("expected migration log to not contain init migration for some_repo")
		}

		// because migration should be reverted
		if slices.ContainsFunc(migrationLogs, func(log migrationLog) bool {
			return log.Repository == "other_repo" && log.MigrationID == "001_init"
		}) {
			t.Fatalf("expected migration log to not contain init migration for other_repo, but only got: %s", migrationLogs)
		}

		_, err = db.Connection().ExecContext(ctx, "SELECT * FROM other_repo")
		if err == nil {
			t.Fatalf("expected error, got nill")
		}
	})
}

type migrationLog struct {
	Repository  string    `db:"repository"`
	MigrationID string    `db:"id"`
	Timestamp   time.Time `db:"timestamp"`
}

type simpleRepo struct {
	fsys fs.FS
}

func (r simpleRepo) Migrations() fs.FS {
	return r.fsys
}

func migrationFS(migrations ...database.Migration) fs.FS {
	mapFS := make(fstest.MapFS)
	for _, m := range migrations {
		content := "-- +migrate Up\n" + m.Up
		if m.Down != "" {
			content += "\n\n-- +migrate Down\n" + m.Down
		}
		mapFS[m.ID+".sql"] = &fstest.MapFile{Data: []byte(content)}
	}
	return mapFS
}
