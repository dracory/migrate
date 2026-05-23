package custom_options_example_test

import (
	"database/sql"
	"testing"

	"github.com/dracory/migrate/examples/custom_options_example"
	_ "modernc.org/sqlite"
)

func TestCreatePostsTable(t *testing.T) {
	// Create in-memory SQLite database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	defer db.Close()

	// Test Up migration
	t.Run("Up", func(t *testing.T) {
		err := custom_options_example.RunMigrations(db)
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
		err := custom_options_example.RollbackMigrations(db)
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
