package migrate

import (
	"testing"

	_ "modernc.org/sqlite"
)

func TestCreateSchemaMigrationsTable(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	migration := NewCreateSchemaMigrationsTable(DefaultTableName)

	t.Run("has_correct_ID", func(t *testing.T) {
		expected := BuiltinMigrationID
		if actual := migration.ID(); actual != expected {
			t.Errorf("Expected ID %s, got %s", expected, actual)
		}
	})

	t.Run("has description", func(t *testing.T) {
		desc := migration.Description()
		if desc == "" {
			t.Error("Expected non-empty description")
		}
	})

	t.Run("up creates schema_migrations table", func(t *testing.T) {
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}
		defer tx.Rollback()

		err = migration.Up(tx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if err := tx.Commit(); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}

		var tableName string
		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", DefaultTableName).Scan(&tableName)
		if err != nil {
			t.Fatalf("Table should exist after Up migration: %v", err)
		}
		if tableName != DefaultTableName {
			t.Errorf("Expected table name %s, got %s", DefaultTableName, tableName)
		}
	})

	t.Run("down drops schema_migrations table", func(t *testing.T) {
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}
		defer tx.Rollback()

		err = migration.Down(tx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if err := tx.Commit(); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", DefaultTableName).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query sqlite_master: %v", err)
		}
		if count != 0 {
			t.Error("Table should not exist after Down migration")
		}
	})
}

func TestGetBuiltinMigrations(t *testing.T) {
	migrations := GetBuiltinMigrations(DefaultTableName)

	if len(migrations) == 0 {
		t.Error("Expected at least one builtin migration")
	}

	firstMigration := migrations[0]
	if firstMigration.ID() != BuiltinMigrationID {
		t.Errorf("Expected first migration to be create_schema_migrations, got %s", firstMigration.ID())
	}
}
