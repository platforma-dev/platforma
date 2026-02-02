package database

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
)

const (
	markerUp   = "-- +migrate Up"
	markerDown = "-- +migrate Down"
	markerID   = "-- +migrate ID:"
)

var (
	errMissingUpSection  = errors.New("missing or empty Up section")
	errEmptyIDOverride   = errors.New("empty ID override")
	errDuplicateIDMarker = errors.New("duplicate ID override marker")
	errIDMarkerNotFirst  = errors.New("ID override marker must be the first marker")
)

// ParseMigrations parses SQL migration files from an fs.FS.
// Files must have .sql extension and contain -- +migrate Up marker.
// The -- +migrate Down marker is optional.
// Migration ID is derived from the filename without extension,
// unless overridden with -- +migrate ID: <custom_id> as the first marker.
// Only one ID override marker is allowed and it must appear before any other markers.
// Migrations are returned sorted lexicographically by filename.
func ParseMigrations(fsys fs.FS) ([]Migration, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var filenames []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		filenames = append(filenames, entry.Name())
	}

	slices.Sort(filenames)

	migrations := make([]Migration, 0, len(filenames))
	for _, filename := range filenames {
		migration, err := parseMigrationFile(fsys, filename)
		if err != nil {
			return nil, fmt.Errorf("failed to parse migration %s: %w", filename, err)
		}
		migrations = append(migrations, migration)
	}

	return migrations, nil
}

func parseMigrationFile(fsys fs.FS, filename string) (Migration, error) {
	file, err := fsys.Open(filename)
	if err != nil {
		return Migration{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	id := strings.TrimSuffix(filename, ".sql")
	idOverridden := false
	anyMarkerSeen := false

	var upBuilder, downBuilder strings.Builder
	var currentSection *strings.Builder

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Check for ID override marker
		if strings.HasPrefix(trimmed, markerID) {
			if idOverridden {
				return Migration{}, errDuplicateIDMarker
			}
			if anyMarkerSeen {
				return Migration{}, errIDMarkerNotFirst
			}
			overrideID := strings.TrimSpace(strings.TrimPrefix(trimmed, markerID))
			if overrideID == "" {
				return Migration{}, errEmptyIDOverride
			}
			id = overrideID
			idOverridden = true
			anyMarkerSeen = true
			continue
		}

		switch trimmed {
		case markerUp:
			currentSection = &upBuilder
			anyMarkerSeen = true
			continue
		case markerDown:
			currentSection = &downBuilder
			anyMarkerSeen = true
			continue
		}

		if currentSection != nil {
			currentSection.WriteString(line)
			currentSection.WriteString("\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return Migration{}, fmt.Errorf("failed to read file: %w", err)
	}

	up := strings.TrimSpace(upBuilder.String())
	if up == "" {
		return Migration{}, errMissingUpSection
	}

	return Migration{
		ID:   id,
		Up:   up,
		Down: strings.TrimSpace(downBuilder.String()),
	}, nil
}
