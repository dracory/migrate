package main

import (
	"context"
	"database/sql"
	"testing"

	"github.com/dracory/migrate"
	_ "modernc.org/sqlite"
)

func TestCreatePostsTable(t *testing.T) {
	// Create in-memory SQLite database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	defer db.Close()

	// Create migrator with custom options
	migrator, err := migrate.New(db, &migrate.Options{
		MigrationTableName: "custom_migrations",
		Logger:             nil, // Disable logging for tests
	})
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}

	// Add migration
	migration := &CreatePostsTable{}
	migrator.AddMigration(migration)

	// Test Up migration
	t.Run("Up", func(t *testing.T) {
		err := migrator.Up(context.Background())
		if err != nil {
			t.Fatalf("Migration Up failed: %v", err)
		}

		// Verify table exists
		var tableName string
		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='posts'").Scan(&tableName)
		if err != nil {
			t.Fatalf("Posts table should exist: %v", err)
		}

		// Verify custom migrations table exists
		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='custom_migrations'").Scan(&tableName)
		if err != nil {
			t.Fatalf("Custom migrations table should exist: %v", err)
		}
	})

	// Test Down migration
	t.Run("Down", func(t *testing.T) {
		err := migrator.Down(context.Background())
		if err != nil {
			t.Fatalf("Migration Down failed: %v", err)
		}

		// Verify table doesn't exist
		var tableName string
		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='posts'").Scan(&tableName)
		if err == nil {
			t.Error("Posts table should not exist after rollback")
		}
	})
}
