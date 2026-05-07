package migrate

import (
	"database/sql"
	"io"
	"log/slog"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	return db
}

func TestNew(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	t.Run("creates migrator with defaults", func(t *testing.T) {
		m := New(db, nil)
		if m == nil {
			t.Fatal("Expected migrator to be created")
		}
	})

	t.Run("creates migrator with custom table name", func(t *testing.T) {
		m := New(db, &Options{MigrationTableName: "custom_migrations"})
		if m == nil {
			t.Fatal("Expected migrator to be created")
		}
	})

	t.Run("creates migrator with custom logger", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		m := New(db, &Options{Logger: logger})
		if m == nil {
			t.Fatal("Expected migrator to be created")
		}
	})

	t.Run("creates migrator with nil options", func(t *testing.T) {
		m := New(db, nil)
		if m == nil {
			t.Fatal("Expected migrator to be created")
		}
	})

	t.Run("rejects invalid migration ID format", func(t *testing.T) {
		m := New(db, nil)

		// Create a migration with invalid ID format
		invalidMigration := &mockMigration{
			id:          "invalid_format",
			description: "Invalid ID",
			upFunc:      func(tx *sql.Tx) error { return nil },
			downFunc:    func(tx *sql.Tx) error { return nil },
		}

		err := m.AddMigration(invalidMigration)
		if err == nil {
			t.Error("Expected error for invalid migration ID format")
		}

		expectedError := "migration ID must follow format YYYY_MM_DD_description"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
		}
	})
}
