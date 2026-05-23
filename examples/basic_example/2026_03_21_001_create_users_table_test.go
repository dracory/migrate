package basic_example_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/dracory/migrate"
	"github.com/dracory/migrate/examples/basic_example"
	_ "modernc.org/sqlite"
)

func TestCreateUsersTable(t *testing.T) {
	// Create in-memory SQLite database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	defer db.Close()

	// Test Up migration
	t.Run("Up", func(t *testing.T) {
		err := basic_example.RunMigrations(db)
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
		migrator, err := migrate.New(db, nil)
		if err != nil {
			t.Fatalf("Failed to create migrator: %v", err)
		}

		err = migrator.Down(context.TODO())
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
