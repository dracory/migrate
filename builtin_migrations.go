package migrate

import (
	"database/sql"

	"github.com/dracory/database"
	"github.com/dracory/sb"
)

type createSchemaMigrationsTable struct {
	tableName string
}

func NewCreateSchemaMigrationsTable(tableName string) *createSchemaMigrationsTable {
	return &createSchemaMigrationsTable{tableName: tableName}
}

func (m *createSchemaMigrationsTable) ID() string {
	return BuiltinMigrationID
}

func (m *createSchemaMigrationsTable) Description() string {
	return "Create schema_migrations table for tracking applied migrations"
}

func (m *createSchemaMigrationsTable) Up(tx *sql.Tx) error {
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

func (m *createSchemaMigrationsTable) Down(tx *sql.Tx) error {
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

// GetBuiltinMigrations returns the built-in migrations that must always run first
func GetBuiltinMigrations(tableName string) []MigrationInterface {
	return []MigrationInterface{
		NewCreateSchemaMigrationsTable(tableName),
	}
}
