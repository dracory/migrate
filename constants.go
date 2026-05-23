package migrate

import (
	"os"
)

// NamingFormat defines the format for migration IDs
type NamingFormat string

const (
	// DefaultTableName is the default name for the migrations tracking table
	// Can be overridden by setting the MIGRATE_TABLE_NAME environment variable
	DefaultTableName = "migration_tracker"

	ColumnID          = "id"
	ColumnBatch       = "batch"
	ColumnDescription = "description"
	ColumnStartedAt   = "started_at"
	ColumnCompletedAt = "completed_at"

	DirectionUp   = "up"
	DirectionDown = "down"

	BuiltinMigrationIDBase = "table_migration_tracker_create"

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
	defaultID := "2001_01_01_0000_" + BuiltinMigrationIDBase

	if format == NamingFormatPrefixYYYY_MM_DD_NNN {
		return "2001_01_01_000_" + BuiltinMigrationIDBase
	}

	if format == NamingFormatPrefixYYYY_MM_DD_HHMM {
		return "2001_01_01_0000_" + BuiltinMigrationIDBase
	}

	if format == NamingFormatPrefixNone {
		return "_" + BuiltinMigrationIDBase
	}

	return defaultID
}
