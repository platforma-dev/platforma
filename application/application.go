package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/platforma-dev/platforma/database"
	"github.com/platforma-dev/platforma/log"
)

// ErrUnknownCommand is returned when an unknown CLI command is provided.
var ErrUnknownCommand = errors.New("unknown command")

// ErrDatabaseMigrationFailed is an error type that represents a failed database migration.
type ErrDatabaseMigrationFailed struct {
	err error
}

// Error returns the formatted error message for ErrDatabaseMigrationFailed.
func (e *ErrDatabaseMigrationFailed) Error() string {
	return fmt.Sprintf("failed to migrate database: %v", e.err)
}

// Unwrap returns the underlying error for ErrDatabaseMigrationFailed.
func (e *ErrDatabaseMigrationFailed) Unwrap() error {
	return e.err
}

// Application manages startup tasks and services for the application lifecycle.
type Application struct {
	startupTasks   []startupTask
	services       map[string]Runner
	healthcheckers map[string]Healthchecker
	databases      map[string]*database.Database
	health         *ApplicationHealth
}

// New creates and returns a new Application instance.
func New() *Application {
	return &Application{services: make(map[string]Runner), healthcheckers: make(map[string]Healthchecker), databases: make(map[string]*database.Database), health: NewApplicationHealth()}
}

// Health returns the current health status of the application.
func (a *Application) Health(ctx context.Context) *ApplicationHealth {
	for hcName, hc := range a.healthcheckers {
		a.health.SetServiceData(hcName, hc.Healthcheck(ctx))
	}
	return a.health
}

// OnStart registers a new startup task with the given runner and configuration.
func (a *Application) OnStart(task Runner, config StartupTaskConfig) {
	a.startupTasks = append(a.startupTasks, startupTask{task, config})
}

func (a *Application) OnStartFunc(task RunnerFunc, config StartupTaskConfig) {
	a.startupTasks = append(a.startupTasks, startupTask{task, config})
}

// RegisterDatabase adds a database to the application.
func (a *Application) RegisterDatabase(dbName string, db *database.Database) {
	a.databases[dbName] = db
}

// RegisterRepository adds a repository to the application.
func (a *Application) RegisterRepository(dbName string, repoName string, repository any) {
	a.databases[dbName].RegisterRepository(repoName, repository)
}

// RegisterService adds a named service to the application.
func (a *Application) RegisterService(serviceName string, service Runner) {

	a.services[serviceName] = service
	a.health.Services[serviceName] = &ServiceHealth{Status: ServiceStatusNotStarted}

	healthcheckerService, ok := service.(Healthchecker)
	if ok {
		a.healthcheckers[serviceName] = healthcheckerService
	}
}

func (a *Application) RegisterDomain(name, dbName string, domain Domain) {
	if dbName != "" {
		repository := domain.GetRepository()
		a.RegisterRepository(dbName, name+"_repository", repository)
	}
}

func (a *Application) printUsage() {
	fmt.Println("Usage: <binary> <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  run       Start the application")
	fmt.Println("  migrate   Run database migrations")
}

func (a *Application) migrate(ctx context.Context) error {
	if len(a.databases) == 0 {
		log.WarnContext(ctx, "no databases registered")
		return nil
	}

	for dbName, db := range a.databases {
		log.InfoContext(ctx, "migrating database", "database", dbName)
		err := db.Migrate(ctx)
		if err != nil {
			log.ErrorContext(ctx, "error in database migration", "error", err, "database", dbName)
			return &ErrDatabaseMigrationFailed{err: err}
		}
	}

	return nil
}

func (a *Application) run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer cancel()

	log.InfoContext(ctx, "starting application", "startupTasks", len(a.startupTasks))

	for i, task := range a.startupTasks {
		log.InfoContext(ctx, "running task", "task", task.config.Name, "index", i)

		taskCtx := context.WithValue(ctx, log.StartupTaskKey, task.config.Name)

		err := task.runner.Run(taskCtx)
		if err != nil {
			log.ErrorContext(ctx, "error in startup task", "error", err, "task", task.config.Name)

			if task.config.AbortOnError {
				return &ErrStartupTaskFailed{err: err}
			}
		}
	}

	var wg sync.WaitGroup

	for serviceName, service := range a.services {
		wg.Add(1)

		serviceCtx := context.WithValue(ctx, log.ServiceNameKey, serviceName)

		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.ErrorContext(serviceCtx, "service panicked", string(log.ServiceNameKey), serviceName, "panic", r)
				}
			}()

			log.InfoContext(ctx, "starting service", string(log.ServiceNameKey), serviceName)
			a.health.StartService(serviceName)

			err := service.Run(serviceCtx)
			if err != nil {
				a.health.FailService(serviceName, err)
				log.ErrorContext(ctx, "error in service", string(log.ServiceNameKey), serviceName, "error", err)
			}
		}()
	}

	a.health.StartedAt = time.Now()

	wg.Wait()

	return nil
}

// Run parses CLI arguments and executes the appropriate command.
// Supported commands: run (start services), migrate (run database migrations).
// Returns nil on success, ErrUnknownCommand for unknown commands.
func (a *Application) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	args := os.Args
	if len(args) < 2 {
		a.printUsage()
		return nil
	}

	command := args[1]
	switch command {
	case "run":
		return a.run(ctx)
	case "migrate":
		return a.migrate(ctx)
	case "--help", "-h":
		a.printUsage()
		return nil
	default:
		a.printUsage()
		return ErrUnknownCommand
	}
}
