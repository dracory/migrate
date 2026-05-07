package migrate

import (
	"context"
	"database/sql"
)

// MigrationInterface defines the contract that all migrations must implement
type MigrationInterface interface {
	// ID returns the unique identifier for this migration
	// Format: YYYYMMDD_NNN (e.g., 20260321_001)
	ID() string

	// Description returns a human-readable description for the migration
	// Example: "Create users table with email index"
	Description() string

	// Up executes the migration to apply database changes
	// Takes context for cancellation support and transaction for atomic operations
	Up(ctx context.Context, tx *sql.Tx) error

	// Down executes the rollback to revert database changes
	// Takes context for cancellation support and transaction for atomic operations
	// Should undo exactly what Up() did
	Down(ctx context.Context, tx *sql.Tx) error
}
