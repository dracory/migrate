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

func TestNew_CreatesMigratorWithDefaults(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := New(db, nil)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	if m == nil {
		t.Fatal("Expected migrator to be created")
	}
}

func TestNew_CreatesMigratorWithCustomTableName(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := New(db, &Options{MigrationTableName: "custom_migrations"})
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	if m == nil {
		t.Fatal("Expected migrator to be created")
	}
}

func TestNew_CreatesMigratorWithCustomLogger(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	m, err := New(db, &Options{Logger: logger})
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	if m == nil {
		t.Fatal("Expected migrator to be created")
	}
}

func TestNew_CreatesMigratorWithNilOptions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := New(db, nil)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	if m == nil {
		t.Fatal("Expected migrator to be created")
	}
}

func TestNew_RejectsInvalidMigrationIDFormat(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := New(db, nil)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}

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

	if !strings.Contains(addErr.Error(), "invalid migration ID") {
		t.Errorf("Expected error containing 'invalid migration ID', got: %v", addErr)
	}
}

func TestNew_ReturnsErrorOnInvalidTableNameWithSpecialCharacters(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := New(db, &Options{MigrationTableName: "invalid-table-name"})
	if err == nil {
		t.Error("Expected error for invalid table name with special characters")
	}
}

func TestNew_ReturnsErrorOnTableNameTooLong(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	longName := strings.Repeat("a", 65)
	_, err := New(db, &Options{MigrationTableName: longName})
	if err == nil {
		t.Error("Expected error for table name too long")
	}
}

func TestNew_ReturnsErrorOnTableNameWithSpaces(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := New(db, &Options{MigrationTableName: "table name"})
	if err == nil {
		t.Error("Expected error for table name with spaces")
	}
}
