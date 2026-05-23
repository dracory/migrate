package basic_example

import (
	"context"
	"database/sql"

	"github.com/dracory/database"
	"github.com/dracory/sb"
)

type CreateUsersTable struct{}

func (m *CreateUsersTable) ID() string {
	return "2026_03_21_0001_create_users_table"
}

func (m *CreateUsersTable) Description() string {
	return "Create users table with email and timestamps"
}

func (m *CreateUsersTable) Up(ctx context.Context, tx *sql.Tx) error {
	dialect := database.DatabaseType(tx)
	sql, err := sb.NewBuilder(dialect).
		Table("users").
		Column(sb.Column{
			Name:       "id",
			Type:       sb.COLUMN_TYPE_INTEGER,
			PrimaryKey: true,
		}).
		Column(sb.Column{
			Name:     "email",
			Type:     sb.COLUMN_TYPE_STRING,
			Length:   255,
			Nullable: false,
		}).
		Column(sb.Column{
			Name:     "created_at",
			Type:     sb.COLUMN_TYPE_DATETIME,
			Nullable: false,
		}).
		Column(sb.Column{
			Name:     "updated_at",
			Type:     sb.COLUMN_TYPE_DATETIME,
			Nullable: false,
		}).
		Create()
	if err != nil {
		return err
	}
	_, err = tx.Exec(sql)
	return err
}

func (m *CreateUsersTable) Down(ctx context.Context, tx *sql.Tx) error {
	dialect := database.DatabaseType(tx)
	sql, err := sb.NewBuilder(dialect).
		Table("users").
		Drop()
	if err != nil {
		return err
	}
	_, err = tx.Exec(sql)
	return err
}
