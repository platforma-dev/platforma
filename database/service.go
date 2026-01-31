package database

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/platforma-dev/platforma/log"
)

type service struct {
	repo *repository
}

func newService(repo *repository) *service {
	return &service{repo: repo}
}

func (s *service) getMigrationLogs(ctx context.Context) ([]migrationLog, error) {
	logs, err := s.repo.getMigrationLogs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration logs: %w", err)
	}
	return logs, nil
}

func (s *service) saveMigrationLog(ctx context.Context, repository, migrationID string) error {
	err := s.repo.saveMigrationLog(ctx, migrationLog{Repository: repository, MigrationID: migrationID, Timestamp: time.Now()})
	if err != nil {
		return fmt.Errorf("failed to save migration log: %w", err)
	}
	return nil
}

func (s *service) saveMigrationLogs(ctx context.Context, migrations []Migration) error {
	masterErr := error(nil)
	for _, migr := range migrations {
		err := s.saveMigrationLog(ctx, migr.repository, migr.ID)
		if err != nil {
			masterErr = errors.Join(masterErr, err)
		}
	}

	return masterErr
}

func (s *service) migrateSelf(ctx context.Context) error {
	migrations := s.repo.migrations()
	appliedMigrations := []Migration{}
	migrationLogs, err := s.repo.getMigrationLogs(ctx)

	if err != nil {
		log.InfoContext(ctx, "migrations log table does not exist yet")
	}

	for _, migr := range migrations {
		if !slices.ContainsFunc(migrationLogs, func(l migrationLog) bool {
			return l.Repository == "platforma_migration" && l.MigrationID == migr.ID
		}) {
			err := s.applyMigration(ctx, migr)
			if err != nil {
				revertErr := s.revertMigrations(ctx, appliedMigrations)
				if revertErr != nil {
					log.ErrorContext(ctx, "got error(s) trying to revert migrations", "error", revertErr)
				}
				return err
			}
			log.InfoContext(ctx, "migration applied", "repository", "platforma_migration", "migrationId", migr.ID)
			migr.repository = "platforma_migration"
			appliedMigrations = append(appliedMigrations, migr)
		} else {
			log.InfoContext(ctx, "migration skipped", "repository", "platforma_migration", "migrationId", migr.ID)
		}
	}

	err = s.saveMigrationLogs(ctx, appliedMigrations)
	if err != nil {
		log.ErrorContext(ctx, "got error(s) trying to save migration logs", "error", err.Error())
	}

	return nil
}

func (s *service) applyMigration(ctx context.Context, migration Migration) error {
	err := s.repo.executeQuery(ctx, migration.Up)
	if err != nil {
		return fmt.Errorf("failed to apply migration: %w", err)
	}
	return nil
}

func (s *service) applyMigrations(ctx context.Context, migrations []Migration, migrationLogs []migrationLog) error {
	appliedMigrations := []Migration{}
	for _, migr := range migrations {
		if !slices.ContainsFunc(migrationLogs, func(l migrationLog) bool {
			return l.Repository == migr.repository && l.MigrationID == migr.ID
		}) {
			err := s.applyMigration(ctx, migr)
			if err != nil {
				revertErr := s.revertMigrations(ctx, appliedMigrations)
				if revertErr != nil {
					log.ErrorContext(ctx, "got error(s) trying to revert migrations", "error", revertErr)
				}
				return err
			}
			log.InfoContext(ctx, "migration applied", "repository", migr.repository, "migrationId", migr.ID)
			appliedMigrations = append(appliedMigrations, migr)
		} else {
			log.InfoContext(ctx, "migration skipped", "repository", migr.repository, "migrationId", migr.ID)
		}
	}

	err := s.saveMigrationLogs(ctx, appliedMigrations)
	if err != nil {
		log.ErrorContext(ctx, "got error(s) trying to save migration logs", "error", err.Error())
	}

	return nil
}

func (s *service) revertMigration(ctx context.Context, migration Migration) error {
	err := s.repo.executeQuery(ctx, migration.Down)
	if err != nil {
		return fmt.Errorf("failed to revert migration: %w", err)
	}
	return nil
}

func (s *service) revertMigrations(ctx context.Context, migrations []Migration) error {
	masterErr := error(nil)
	for _, migr := range slices.Backward(migrations) {
		err := s.revertMigration(ctx, migr)
		if err != nil {
			masterErr = errors.Join(masterErr, fmt.Errorf("failed to revert migration %s: %w", migr.ID, err))
		}
	}

	return masterErr
}
