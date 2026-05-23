package migrate

import (
	"context"
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
		m, err := New(db, nil)
		if err != nil {
			t.Fatalf("Failed to create migrator: %v", err)
		}
		if m == nil {
			t.Fatal("Expected migrator to be created")
		}
	})

	t.Run("creates migrator with custom table name", func(t *testing.T) {
		m, err := New(db, &Options{MigrationTableName: "custom_migrations"})
		if err != nil {
			t.Fatalf("Failed to create migrator: %v", err)
		}
		if m == nil {
			t.Fatal("Expected migrator to be created")
		}
	})

	t.Run("creates migrator with custom logger", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		m, err := New(db, &Options{Logger: logger})
		if err != nil {
			t.Fatalf("Failed to create migrator: %v", err)
		}
		if m == nil {
			t.Fatal("Expected migrator to be created")
		}
	})

	t.Run("creates migrator with nil options", func(t *testing.T) {
		m, err := New(db, nil)
		if err != nil {
			t.Fatalf("Failed to create migrator: %v", err)
		}
		if m == nil {
			t.Fatal("Expected migrator to be created")
		}
	})

	t.Run("rejects invalid migration ID format", func(t *testing.T) {
		m, err := New(db, nil)
		if err != nil {
			t.Fatalf("Failed to create migrator: %v", err)
		}

		// Create a migration with invalid ID format
		invalidMigration := &mockMigration{
			id:          "invalid_format",
			description: "Invalid ID",
			upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
			downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
		}

		addErr := m.AddMigration(invalidMigration)
		if addErr == nil {
			t.Error("Expected error for invalid migration ID format")
		}

		expectedError := "migration ID must follow format YYYY_MM_DD_HHMM_description"
		if !strings.Contains(addErr.Error(), expectedError) {
			t.Errorf("Expected error containing '%s', got: %v", expectedError, addErr)
		}
	})

	t.Run("returns error on invalid table name with special characters", func(t *testing.T) {
		_, err := New(db, &Options{MigrationTableName: "invalid-table-name"})
		if err == nil {
			t.Error("Expected error for invalid table name with special characters")
		}
	})

	t.Run("returns error on table name too long", func(t *testing.T) {
		longName := strings.Repeat("a", 65)
		_, err := New(db, &Options{MigrationTableName: longName})
		if err == nil {
			t.Error("Expected error for table name too long")
		}
	})

	t.Run("returns error on table name with spaces", func(t *testing.T) {
		_, err := New(db, &Options{MigrationTableName: "table name"})
		if err == nil {
			t.Error("Expected error for table name with spaces")
		}
	})
}

func TestValidateTableName(t *testing.T) {
	t.Run("accepts valid table names", func(t *testing.T) {
		validNames := []string{
			"schema_migrations",
			"migrations",
			"custom_table",
			"a",
			"Table123",
			"_underscore",
			"_123",
		}

		for _, name := range validNames {
			if err := validateTableName(name); err != nil {
				t.Errorf("Expected valid name '%s' to be accepted, got: %v", name, err)
			}
		}
	})

	t.Run("rejects empty table name", func(t *testing.T) {
		if err := validateTableName(""); err == nil {
			t.Error("Expected error for empty table name")
		}
	})

	t.Run("rejects table name too long", func(t *testing.T) {
		longName := strings.Repeat("a", 65)
		if err := validateTableName(longName); err == nil {
			t.Error("Expected error for table name too long")
		}
	})

	t.Run("rejects table name starting with digit", func(t *testing.T) {
		invalidNames := []string{
			"123_table",
			"9table",
			"0_migrations",
		}

		for _, name := range invalidNames {
			if err := validateTableName(name); err == nil {
				t.Errorf("Expected invalid name '%s' (starting with digit) to be rejected", name)
			}
		}
	})

	t.Run("rejects table name with special characters", func(t *testing.T) {
		invalidNames := []string{
			"table-name",
			"table.name",
			"table name",
			"table;name",
			"table'name",
			"table\"name",
			"table*name",
		}

		for _, name := range invalidNames {
			if err := validateTableName(name); err == nil {
				t.Errorf("Expected invalid name '%s' to be rejected", name)
			}
		}
	})
}
