package migrate

import "context"

// MigratorInterface defines the contract for database migration operations
type MigratorInterface interface {
	// AddMigration adds a new migration to the list
	AddMigration(migration MigrationInterface) error

	// AddMigrations adds multiple migrations to the runner
	AddMigrations(migrations []MigrationInterface) error

	// Up runs all pending migrations
	Up(ctx context.Context) error

	// Down rolls back the last migration
	Down(ctx context.Context) error

	// Status shows migration status
	Status(ctx context.Context) error
}
