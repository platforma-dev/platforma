package database_test

import (
	"testing"
	"testing/fstest"

	"github.com/platforma-dev/platforma/database"
)

func TestParseMigrations(t *testing.T) {
	t.Parallel()

	t.Run("parses single migration with up and down", func(t *testing.T) {
		t.Parallel()

		fsys := fstest.MapFS{
			"001_init.sql": &fstest.MapFile{
				Data: []byte("-- +migrate Up\nCREATE TABLE users (id INT);\n\n-- +migrate Down\nDROP TABLE users;"),
			},
		}

		migrations, err := database.ParseMigrations(fsys)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(migrations) != 1 {
			t.Fatalf("expected 1 migration, got %d", len(migrations))
		}

		if migrations[0].ID != "001_init" {
			t.Errorf("expected ID '001_init', got '%s'", migrations[0].ID)
		}

		if migrations[0].Up != "CREATE TABLE users (id INT);" {
			t.Errorf("expected Up 'CREATE TABLE users (id INT);', got '%s'", migrations[0].Up)
		}

		if migrations[0].Down != "DROP TABLE users;" {
			t.Errorf("expected Down 'DROP TABLE users;', got '%s'", migrations[0].Down)
		}
	})

	t.Run("parses migration with up only", func(t *testing.T) {
		t.Parallel()

		fsys := fstest.MapFS{
			"001_init.sql": &fstest.MapFile{
				Data: []byte("-- +migrate Up\nCREATE TABLE users (id INT);"),
			},
		}

		migrations, err := database.ParseMigrations(fsys)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(migrations) != 1 {
			t.Fatalf("expected 1 migration, got %d", len(migrations))
		}

		if migrations[0].Up != "CREATE TABLE users (id INT);" {
			t.Errorf("expected Up 'CREATE TABLE users (id INT);', got '%s'", migrations[0].Up)
		}

		if migrations[0].Down != "" {
			t.Errorf("expected empty Down, got '%s'", migrations[0].Down)
		}
	})

	t.Run("sorts migrations lexicographically", func(t *testing.T) {
		t.Parallel()

		fsys := fstest.MapFS{
			"002_second.sql": &fstest.MapFile{
				Data: []byte("-- +migrate Up\nSECOND"),
			},
			"001_first.sql": &fstest.MapFile{
				Data: []byte("-- +migrate Up\nFIRST"),
			},
			"003_third.sql": &fstest.MapFile{
				Data: []byte("-- +migrate Up\nTHIRD"),
			},
		}

		migrations, err := database.ParseMigrations(fsys)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(migrations) != 3 {
			t.Fatalf("expected 3 migrations, got %d", len(migrations))
		}

		if migrations[0].ID != "001_first" {
			t.Errorf("expected first migration ID '001_first', got '%s'", migrations[0].ID)
		}

		if migrations[1].ID != "002_second" {
			t.Errorf("expected second migration ID '002_second', got '%s'", migrations[1].ID)
		}

		if migrations[2].ID != "003_third" {
			t.Errorf("expected third migration ID '003_third', got '%s'", migrations[2].ID)
		}
	})

	t.Run("ignores non-sql files", func(t *testing.T) {
		t.Parallel()

		fsys := fstest.MapFS{
			"001_init.sql": &fstest.MapFile{
				Data: []byte("-- +migrate Up\nCREATE TABLE users (id INT);"),
			},
			"readme.txt": &fstest.MapFile{
				Data: []byte("This is a readme"),
			},
		}

		migrations, err := database.ParseMigrations(fsys)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(migrations) != 1 {
			t.Fatalf("expected 1 migration, got %d", len(migrations))
		}
	})

	t.Run("errors on missing up section", func(t *testing.T) {
		t.Parallel()

		fsys := fstest.MapFS{
			"001_init.sql": &fstest.MapFile{
				Data: []byte("-- +migrate Down\nDROP TABLE users;"),
			},
		}

		_, err := database.ParseMigrations(fsys)
		if err == nil {
			t.Fatal("expected error for missing Up section")
		}
	})

	t.Run("errors on empty up section", func(t *testing.T) {
		t.Parallel()

		fsys := fstest.MapFS{
			"001_init.sql": &fstest.MapFile{
				Data: []byte("-- +migrate Up\n\n-- +migrate Down\nDROP TABLE users;"),
			},
		}

		_, err := database.ParseMigrations(fsys)
		if err == nil {
			t.Fatal("expected error for empty Up section")
		}
	})

	t.Run("handles empty filesystem", func(t *testing.T) {
		t.Parallel()

		fsys := fstest.MapFS{}

		migrations, err := database.ParseMigrations(fsys)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(migrations) != 0 {
			t.Fatalf("expected 0 migrations, got %d", len(migrations))
		}
	})

	t.Run("handles multiline SQL statements", func(t *testing.T) {
		t.Parallel()

		fsys := fstest.MapFS{
			"001_init.sql": &fstest.MapFile{
				Data: []byte("-- +migrate Up\nCREATE TABLE users (\n\tid INT,\n\tname TEXT\n);\n\n-- +migrate Down\nDROP TABLE users;"),
			},
		}

		migrations, err := database.ParseMigrations(fsys)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := "CREATE TABLE users (\n\tid INT,\n\tname TEXT\n);"
		if migrations[0].Up != expected {
			t.Errorf("expected Up:\n%s\n\ngot:\n%s", expected, migrations[0].Up)
		}
	})

	t.Run("parses migration with ID override", func(t *testing.T) {
		t.Parallel()

		fsys := fstest.MapFS{
			"001_init.sql": &fstest.MapFile{
				Data: []byte("-- +migrate ID: custom_migration_id\n-- +migrate Up\nCREATE TABLE users (id INT);\n\n-- +migrate Down\nDROP TABLE users;"),
			},
		}

		migrations, err := database.ParseMigrations(fsys)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(migrations) != 1 {
			t.Fatalf("expected 1 migration, got %d", len(migrations))
		}

		if migrations[0].ID != "custom_migration_id" {
			t.Errorf("expected ID 'custom_migration_id', got '%s'", migrations[0].ID)
		}

		if migrations[0].Up != "CREATE TABLE users (id INT);" {
			t.Errorf("expected Up 'CREATE TABLE users (id INT);', got '%s'", migrations[0].Up)
		}
	})

	t.Run("errors on empty ID override", func(t *testing.T) {
		t.Parallel()

		fsys := fstest.MapFS{
			"001_init.sql": &fstest.MapFile{
				Data: []byte("-- +migrate ID:\n-- +migrate Up\nCREATE TABLE users (id INT);"),
			},
		}

		_, err := database.ParseMigrations(fsys)
		if err == nil {
			t.Fatal("expected error for empty ID override")
		}
	})

	t.Run("errors on ID override with only whitespace", func(t *testing.T) {
		t.Parallel()

		fsys := fstest.MapFS{
			"001_init.sql": &fstest.MapFile{
				Data: []byte("-- +migrate ID:   \n-- +migrate Up\nCREATE TABLE users (id INT);"),
			},
		}

		_, err := database.ParseMigrations(fsys)
		if err == nil {
			t.Fatal("expected error for ID override with only whitespace")
		}
	})

	t.Run("uses first ID override when multiple present", func(t *testing.T) {
		t.Parallel()

		fsys := fstest.MapFS{
			"001_init.sql": &fstest.MapFile{
				Data: []byte("-- +migrate ID: first_id\n-- +migrate ID: second_id\n-- +migrate Up\nCREATE TABLE users (id INT);"),
			},
		}

		migrations, err := database.ParseMigrations(fsys)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if migrations[0].ID != "first_id" {
			t.Errorf("expected ID 'first_id', got '%s'", migrations[0].ID)
		}
	})

	t.Run("ignores ID marker after Up section", func(t *testing.T) {
		t.Parallel()

		fsys := fstest.MapFS{
			"001_init.sql": &fstest.MapFile{
				Data: []byte("-- +migrate Up\nCREATE TABLE users (id INT);\n-- +migrate ID: should_be_ignored"),
			},
		}

		migrations, err := database.ParseMigrations(fsys)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if migrations[0].ID != "001_init" {
			t.Errorf("expected ID '001_init', got '%s'", migrations[0].ID)
		}
	})

	t.Run("parses mixed migrations with and without ID override", func(t *testing.T) {
		t.Parallel()

		fsys := fstest.MapFS{
			"001_first.sql": &fstest.MapFile{
				Data: []byte("-- +migrate ID: custom_first\n-- +migrate Up\nFIRST"),
			},
			"002_second.sql": &fstest.MapFile{
				Data: []byte("-- +migrate Up\nSECOND"),
			},
			"003_third.sql": &fstest.MapFile{
				Data: []byte("-- +migrate ID: custom_third\n-- +migrate Up\nTHIRD"),
			},
		}

		migrations, err := database.ParseMigrations(fsys)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(migrations) != 3 {
			t.Fatalf("expected 3 migrations, got %d", len(migrations))
		}

		if migrations[0].ID != "custom_first" {
			t.Errorf("expected first migration ID 'custom_first', got '%s'", migrations[0].ID)
		}

		if migrations[1].ID != "002_second" {
			t.Errorf("expected second migration ID '002_second', got '%s'", migrations[1].ID)
		}

		if migrations[2].ID != "custom_third" {
			t.Errorf("expected third migration ID 'custom_third', got '%s'", migrations[2].ID)
		}
	})
}
