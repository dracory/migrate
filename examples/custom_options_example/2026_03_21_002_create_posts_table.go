package main

import (
	"context"
	"database/sql"

	"github.com/dracory/database"
	"github.com/dracory/sb"
)

type CreatePostsTable struct{}

func (m *CreatePostsTable) ID() string {
	return "2026_03_21_0002_create_posts_table"
}

func (m *CreatePostsTable) Description() string {
	return "Create posts table with title, content, and user reference"
}

func (m *CreatePostsTable) Up(ctx context.Context, tx *sql.Tx) error {
	dialect := database.DatabaseType(tx)
	sql, err := sb.NewBuilder(dialect).
		Table("posts").
		Column(sb.Column{
			Name:       "id",
			Type:       sb.COLUMN_TYPE_INTEGER,
			PrimaryKey: true,
		}).
		Column(sb.Column{
			Name:     "title",
			Type:     sb.COLUMN_TYPE_STRING,
			Length:   255,
			Nullable: false,
		}).
		Column(sb.Column{
			Name:     "content",
			Type:     sb.COLUMN_TYPE_TEXT,
			Nullable: true,
		}).
		Column(sb.Column{
			Name:     "user_id",
			Type:     sb.COLUMN_TYPE_INTEGER,
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

func (m *CreatePostsTable) Down(ctx context.Context, tx *sql.Tx) error {
	dialect := database.DatabaseType(tx)
	sql, err := sb.NewBuilder(dialect).
		Table("posts").
		Drop()
	if err != nil {
		return err
	}
	_, err = tx.Exec(sql)
	return err
}
