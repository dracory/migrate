package migrate

import (
	"context"
	"database/sql"

	"github.com/dracory/database"
	"github.com/dracory/sb"
)

// createSchemaMigrationsTable is a builtin migration that creates the table
// used to track applied migrations
type createSchemaMigrationsTable struct {
	tableName    string
	namingFormat NamingFormat
}

// NewCreateSchemaMigrationsTable creates a new builtin migration for the schema migrations table
//
// Business Logic:
// - Creates a migration instance with the specified table name and naming format
// - The migration will create a table to track applied migrations
// - Table name can be customized (defaults to "migration_tracker")
// - Naming format determines the migration ID format
func NewCreateSchemaMigrationsTable(tableName string, format NamingFormat) MigrationInterface {
	return &createSchemaMigrationsTable{
		tableName:    tableName,
		namingFormat: format,
	}
}

func (m *createSchemaMigrationsTable) ID() string {
	return GetBuiltinMigrationID(m.namingFormat)
}

func (m *createSchemaMigrationsTable) Description() string {
	return "Create schema migrations tracking table"
}

// Up creates the schema migrations tracking table
//
// Business Logic:
// - Detects the database dialect from the transaction
// - Creates a table with columns: id (PK), batch, description, started_at, completed_at
// - Uses CREATE IF NOT EXISTS to avoid errors if table already exists
// - All columns are non-nullable except for the required tracking fields
// - Executes the SQL within the provided transaction
func (m *createSchemaMigrationsTable) Up(ctx context.Context, tx *sql.Tx) error {
	dialect := database.DatabaseType(tx)
	sql, err := sb.NewBuilder(dialect).
		Table(m.tableName).
		Column(sb.Column{
			Name:       ColumnID,
			Type:       sb.COLUMN_TYPE_STRING,
			Length:     255,
			PrimaryKey: true,
		}).
		Column(sb.Column{
			Name:     ColumnBatch,
			Type:     sb.COLUMN_TYPE_INTEGER,
			Nullable: false,
		}).
		Column(sb.Column{
			Name:     ColumnDescription,
			Type:     sb.COLUMN_TYPE_TEXT,
			Nullable: false,
		}).
		Column(sb.Column{
			Name:     ColumnStartedAt,
			Type:     sb.COLUMN_TYPE_DATETIME,
			Nullable: false,
		}).
		Column(sb.Column{
			Name:     ColumnCompletedAt,
			Type:     sb.COLUMN_TYPE_DATETIME,
			Nullable: false,
		}).
		CreateIfNotExists()

	if err != nil {
		return err
	}

	_, err = tx.Exec(sql)
	return err
}

// Down drops the schema migrations tracking table
//
// Business Logic:
// - Detects the database dialect from the transaction
// - Generates DROP TABLE SQL for the migrations table
// - Executes the SQL within the provided transaction
// - This will delete all migration tracking records
func (m *createSchemaMigrationsTable) Down(ctx context.Context, tx *sql.Tx) error {
	dialect := database.DatabaseType(tx)
	sql, err := sb.NewBuilder(dialect).
		Table(m.tableName).
		Drop()

	if err != nil {
		return err
	}

	_, err = tx.Exec(sql)
	return err
}

// GetBuiltinMigrations returns the built-in migrations with the specified naming format
//
// Business Logic:
// - Returns a slice containing all built-in migrations
// - Currently includes only the schema migrations table creation
// - The table name and naming format are passed to the migration
// - Built-in migrations are automatically added before user migrations
func GetBuiltinMigrations(tableName string, format NamingFormat) []MigrationInterface {
	return []MigrationInterface{
		NewCreateSchemaMigrationsTable(tableName, format),
	}
}
