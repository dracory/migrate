package migrate

import (
	"os"
)

// NamingFormat defines the format for migration IDs
type NamingFormat string

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

	// NamingFormatHHMM uses YYYY_MM_DD_HHMM_description format
	NamingFormatHHMM NamingFormat = "hhmm"
	// NamingFormatNNN uses YYYY_MM_DD_NNN_description format
	NamingFormatNNN NamingFormat = "nnn"
)

// GetDefaultTableName returns the default table name, checking for environment variable override
func GetDefaultTableName() string {
	if envName := os.Getenv("MIGRATE_TABLE_NAME"); envName != "" {
		return envName
	}
	return DefaultTableName
}
