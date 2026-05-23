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

	// NamingFormatPrefixYYYY_MM_DD_HHMM uses timestamp-based format
	NamingFormatPrefixYYYY_MM_DD_HHMM NamingFormat = "YYYY_MM_DD_HHMM"
	// NamingFormatPrefixYYYY_MM_DD_NNN uses sequence-based format
	NamingFormatPrefixYYYY_MM_DD_NNN NamingFormat = "YYYY_MM_DD_NNN"
	// NamingFormatPrefixNone uses no prefix format restriction
	NamingFormatPrefixNone NamingFormat = "none"
)

// GetDefaultTableName returns the default table name, checking for environment variable override
func GetDefaultTableName() string {
	if envName := os.Getenv("MIGRATE_TABLE_NAME"); envName != "" {
		return envName
	}
	return DefaultTableName
}

// GetBuiltinMigrationID returns the builtin migration ID based on naming format
func GetBuiltinMigrationID(format NamingFormat) string {
	if format == NamingFormatPrefixYYYY_MM_DD_NNN {
		return "2022_01_01_000_create_schema_migrations"
	}
	return BuiltinMigrationID
}
