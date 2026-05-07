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
			upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
			downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
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

	t.Run("panics on invalid table name with special characters", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for invalid table name with special characters")
			}
		}()

		New(db, &Options{MigrationTableName: "invalid-table-name"})
	})

	t.Run("panics on table name too long", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for table name too long")
			}
		}()

		longName := strings.Repeat("a", 65)
		New(db, &Options{MigrationTableName: longName})
	})

	t.Run("panics on table name with spaces", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for table name with spaces")
			}
		}()

		New(db, &Options{MigrationTableName: "table name"})
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
