package migrate

import (
	"database/sql"
	"fmt"
	"log/slog"
)

// Options configures the Migrator behavior
type Options struct {
	// MigrationTableName is the name of the table used to track applied migrations.
	// Defaults to "schema_migrations" if not specified.
	MigrationTableName string

	// Logger is used for migration logging.
	// If nil, logging is disabled.
	Logger *slog.Logger

	// NamingFormat specifies the format for migration IDs.
	// Defaults to NamingFormatHHMM if not specified.
	NamingFormat NamingFormat
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

	namingFormat := opts.NamingFormat
	if namingFormat == "" {
		namingFormat = NamingFormatHHMM
	}

	return &migratorImplementation{
		db:           db,
		migrations:   make([]*migration, 0),
		tableName:    tableName,
		logger:       logger,
		namingFormat: namingFormat,
	}, nil
}
