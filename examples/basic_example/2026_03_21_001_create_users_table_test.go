package main

import (
	"database/sql"
	"testing"

	"github.com/dracory/migrate"
	_ "modernc.org/sqlite"
)

func TestCreateUsersTable(t *testing.T) {
	// Create in-memory SQLite database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	defer db.Close()

	// Create migrator
	migrator := migrate.New(db, nil)

	// Builtin migrations will be added automatically on first Up() call

	// Add migration
	migration := &CreateUsersTable{}
	migrator.AddMigration(migration)

	// Test Up migration
	t.Run("Up", func(t *testing.T) {
		err := migrator.Up()
		if err != nil {
			t.Fatalf("Migration Up failed: %v", err)
		}

		// Verify table exists
		var tableName string
		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Scan(&tableName)
		if err != nil {
			t.Fatalf("Users table should exist: %v", err)
		}
	})

	// Test Down migration
	t.Run("Down", func(t *testing.T) {
		err := migrator.Down()
		if err != nil {
			t.Fatalf("Migration Down failed: %v", err)
		}

		// Verify table doesn't exist
		var tableName string
		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Scan(&tableName)
		if err == nil {
			t.Error("Users table should not exist after rollback")
		}
	})
}
