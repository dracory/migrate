package migrate

import (
	"os"
)

const (
	// DefaultTableName is the default name for the migrations tracking table
	// Can be overridden by setting the MIGRATE_TABLE_NAME environment variable
	DefaultTableName = "schema_migrations"

	ColumnID          = "id"
	ColumnBatch       = "batch"
	ColumnDescription = "description"
	ColumnStartedAt   = "started_at"
	ColumnCompletedAt = "completed_at"

	DirectionUp   = "up"
	DirectionDown = "down"

	BuiltinMigrationID = "2022_01_01_0000_create_schema_migrations"
)

// GetDefaultTableName returns the default table name, checking for environment variable override
func GetDefaultTableName() string {
	if envName := os.Getenv("MIGRATE_TABLE_NAME"); envName != "" {
		return envName
	}
	return DefaultTableName
}
