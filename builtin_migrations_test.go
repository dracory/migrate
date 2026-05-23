package migrate_test

import (
	"context"
	"testing"

	"github.com/dracory/migrate"

	_ "modernc.org/sqlite"
)

func TestNewCreateSchemaMigrationsTable_HasCorrectID(t *testing.T) {
	migration := migrate.NewCreateSchemaMigrationsTable("migration_tracker", migrate.NamingFormatPrefixYYYY_MM_DD_HHMM)

	expected := migrate.BuiltinMigrationID
	if actual := migration.ID(); actual != expected {
		t.Errorf("Expected ID %s, got %s", expected, actual)
	}
}

func TestNewCreateSchemaMigrationsTable_HasDescription(t *testing.T) {
	migration := migrate.NewCreateSchemaMigrationsTable("migration_tracker", migrate.NamingFormatPrefixYYYY_MM_DD_HHMM)

	desc := migration.Description()
	if desc == "" {
		t.Error("Expected non-empty description")
	}
}

func TestNewCreateSchemaMigrationsTable_UpCreatesSchemaMigrationsTable(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	migration := migrate.NewCreateSchemaMigrationsTable("migration_tracker", migrate.NamingFormatPrefixYYYY_MM_DD_HHMM)

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	err = migration.Up(context.Background(), tx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", "migration_tracker").Scan(&tableName)
	if err != nil {
		t.Fatalf("Table should exist after Up migration: %v", err)
	}
	if tableName != "migration_tracker" {
		t.Errorf("Expected table name migration_tracker, got %s", tableName)
	}
}

func TestNewCreateSchemaMigrationsTable_DownDropsSchemaMigrationsTable(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// First create the table
	migration := migrate.NewCreateSchemaMigrationsTable("migration_tracker", migrate.NamingFormatPrefixYYYY_MM_DD_HHMM)

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	err = migration.Up(context.Background(), tx)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Now drop it
	tx2, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx2.Rollback()

	err = migration.Down(context.Background(), tx2)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if err := tx2.Commit(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", "migration_tracker").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query sqlite_master: %v", err)
	}
	if count != 0 {
		t.Error("Table should not exist after Down migration")
	}
}

func TestGetBuiltinMigrations(t *testing.T) {
	migrations := migrate.GetBuiltinMigrations("migration_tracker", migrate.NamingFormatPrefixYYYY_MM_DD_HHMM)

	if len(migrations) == 0 {
		t.Error("Expected at least one builtin migration")
	}

	firstMigration := migrations[0]
	if firstMigration.ID() != migrate.BuiltinMigrationID {
		t.Errorf("Expected first migration to be table_migration_tracker_create, got %s", firstMigration.ID())
	}
}

func TestNewCreateSchemaMigrationsTable_NNN(t *testing.T) {
	migration := migrate.NewCreateSchemaMigrationsTable("migration_tracker", migrate.NamingFormatPrefixYYYY_MM_DD_NNN)

	expected := migrate.GetBuiltinMigrationID(migrate.NamingFormatPrefixYYYY_MM_DD_NNN)
	if actual := migration.ID(); actual != expected {
		t.Errorf("Expected ID %s, got %s", expected, actual)
	}
}

func TestGetBuiltinMigrations_NNN(t *testing.T) {
	migrations := migrate.GetBuiltinMigrations("migration_tracker", migrate.NamingFormatPrefixYYYY_MM_DD_NNN)

	if len(migrations) == 0 {
		t.Error("Expected at least one builtin migration")
	}

	firstMigration := migrations[0]
	if firstMigration.ID() != migrate.GetBuiltinMigrationID(migrate.NamingFormatPrefixYYYY_MM_DD_NNN) {
		t.Errorf("Expected first migration to be table_migration_tracker_create with NNN format, got %s", firstMigration.ID())
	}
}
