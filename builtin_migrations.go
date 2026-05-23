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

func NewCreateSchemaMigrationsTable(tableName string, format NamingFormat) MigrationInterface {
	return &createSchemaMigrationsTable{
		tableName:    tableName,
		namingFormat: format,
	}
}

func (m *createSchemaMigrationsTable) ID() string {
	if m.namingFormat == NamingFormatNNN {
		return "2022_01_01_000_create_schema_migrations"
	}
	return BuiltinMigrationID
}

func (m *createSchemaMigrationsTable) Description() string {
	return "Create schema migrations tracking table"
}

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
func GetBuiltinMigrations(tableName string, format NamingFormat) []MigrationInterface {
	return []MigrationInterface{
		NewCreateSchemaMigrationsTable(tableName, format),
	}
}
