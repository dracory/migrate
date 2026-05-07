package migrate

import (
	"database/sql"
	"fmt"
	"log/slog"
	"unicode"
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

// validateTableName ensures the table name contains only safe characters
func validateTableName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("table name cannot be empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("table name too long (max 64 characters)")
	}

	// First character must be a letter or underscore (not a digit)
	firstRune := rune(name[0])
	if !unicode.IsLetter(firstRune) && firstRune != '_' {
		return fmt.Errorf("table name must start with a letter or underscore, not '%c'", firstRune)
	}

	for i, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return fmt.Errorf("table name contains invalid character '%c' at position %d (only alphanumeric and underscore allowed)", r, i)
		}
	}
	return nil
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

	if err := validateTableName(tableName); err != nil {
		panic(fmt.Sprintf("invalid migration table name: %v", err))
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
