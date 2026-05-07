package migrate

import "database/sql"

// MigrationInterface defines the contract that all migrations must implement
type MigrationInterface interface {
	// ID returns the unique identifier for this migration
	// Format: YYYYMMDD_NNN (e.g., 20260321_001)
	ID() string

	// Description returns a human-readable description for the migration
	// Example: "Create users table with email index"
	Description() string

	// Up executes the migration to apply database changes
	// Takes transaction for atomic operations
	Up(tx *sql.Tx) error

	// Down executes the rollback to revert database changes
	// Should undo exactly what Up() did
	Down(tx *sql.Tx) error
}
