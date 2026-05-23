package migrate

import (
	"database/sql"
	"fmt"
	"log/slog"
)

// Options configures the Migrator behavior
type Options struct {
	// MigrationTableName is the name of the table used to track applied migrations.
	// Defaults to "migration_tracker" if not specified.
	MigrationTableName string

	// Logger is used for migration logging.
	// If nil, logging is disabled.
	Logger *slog.Logger

	// NamingFormatPrefix specifies the prefix format for migration IDs.
	// Use NamingFormatPrefixNone ("none") to disable prefix validation.
	// Empty string ("") defaults to NamingFormatPrefixYYYY_MM_DD_HHMM.
	NamingFormatPrefix NamingFormat
}

// New creates a new migrator instance
func New(db *sql.DB, opts *Options) (MigratorInterface, error) {
	if opts == nil {
		opts = &Options{}
	}

	tableName := opts.MigrationTableName
	if tableName == "" {
		tableName = GetDefaultTableName()
	}

	if err := ValidateTableName(tableName); err != nil {
		return nil, fmt.Errorf("invalid migration table name: %w", err)
	}

	logger := opts.Logger
	// If nil, logging is disabled (keep logger as nil)

	namingFormat := opts.NamingFormatPrefix
	if namingFormat == "" {
		namingFormat = NamingFormatPrefixYYYY_MM_DD_HHMM
	}

	return &migratorImplementation{
		db:           db,
		migrations:   make([]*migration, 0),
		tableName:    tableName,
		logger:       logger,
		namingFormat: namingFormat,
	}, nil
}
