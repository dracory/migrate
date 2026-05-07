package migrate

import (
	"database/sql"
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
}

// New creates a new migrator instance
func New(db *sql.DB, opts *Options) MigratorInterface {
	if opts == nil {
		opts = &Options{}
	}

	tableName := opts.MigrationTableName
	if tableName == "" {
		tableName = DefaultTableName
	}

	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &migratorImplementation{
		db:         db,
		migrations: make([]*migration, 0),
		tableName:  tableName,
		logger:     logger,
	}
}
