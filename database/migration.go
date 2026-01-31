package database

import (
	"io/fs"
	"time"
)

type migrationLog struct {
	Repository  string    `db:"repository"`
	MigrationID string    `db:"id"`
	Timestamp   time.Time `db:"timestamp"`
}

// Migration represents a database migration with up and down SQL statements.
type Migration struct {
	ID         string
	Up         string
	Down       string
	repository string
}

type migrator interface {
	Migrations() fs.FS
}
