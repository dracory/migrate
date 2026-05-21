package migrate

import "context"

// MigrationStatus represents the status of a single migration
type MigrationStatus struct {
	ID          string
	Description string
	Applied     bool
}

// MigrationRecord represents a migration record from the database
type MigrationRecord struct {
	ID          string
	Batch       string
	Description string
	StartedAt   string
	CompletedAt string
}

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

	// Status shows migration status (prints to stdout)
	Status(ctx context.Context) error

	// GetStatus returns migration status as structured data
	GetStatus(ctx context.Context) ([]MigrationStatus, error)

	// GetHistory returns the migration execution history from the database
	GetHistory(ctx context.Context) ([]MigrationRecord, error)
}
