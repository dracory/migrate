package migrate_test

import (
	"context"
	"testing"

	"github.com/dracory/migrate"

	_ "modernc.org/sqlite"
)

func TestNewCreateSchemaMigrationsTable_HasCorrectID(t *testing.T) {
	migration := migrate.NewCreateSchemaMigrationsTable("schema_migrations")

	expected := "2022_01_01_0000_create_schema_migrations"
	if actual := migration.ID(); actual != expected {
		t.Errorf("Expected ID %s, got %s", expected, actual)
	}
}

func TestNewCreateSchemaMigrationsTable_HasDescription(t *testing.T) {
	migration := migrate.NewCreateSchemaMigrationsTable("schema_migrations")

	desc := migration.Description()
	if desc == "" {
		t.Error("Expected non-empty description")
	}
}

func TestNewCreateSchemaMigrationsTable_UpCreatesSchemaMigrationsTable(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	migration := migrate.NewCreateSchemaMigrationsTable("schema_migrations")

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
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", "schema_migrations").Scan(&tableName)
	if err != nil {
		t.Fatalf("Table should exist after Up migration: %v", err)
	}
	if tableName != "schema_migrations" {
		t.Errorf("Expected table name schema_migrations, got %s", tableName)
	}
}

func TestNewCreateSchemaMigrationsTable_DownDropsSchemaMigrationsTable(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// First create the table
	migration := migrate.NewCreateSchemaMigrationsTable("schema_migrations")

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
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", "schema_migrations").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query sqlite_master: %v", err)
	}
	if count != 0 {
		t.Error("Table should not exist after Down migration")
	}
}

func TestGetBuiltinMigrations(t *testing.T) {
	migrations := migrate.GetBuiltinMigrations("schema_migrations")

	if len(migrations) == 0 {
		t.Error("Expected at least one builtin migration")
	}

	firstMigration := migrations[0]
	if firstMigration.ID() != "2022_01_01_0000_create_schema_migrations" {
		t.Errorf("Expected first migration to be create_schema_migrations, got %s", firstMigration.ID())
	}
}

func TestNewCreateSchemaMigrationsTableWithFormat_NNN(t *testing.T) {
	migration := migrate.NewCreateSchemaMigrationsTableWithFormat("schema_migrations", migrate.NamingFormatNNN)

	expected := "2022_01_01_000_create_schema_migrations"
	if actual := migration.ID(); actual != expected {
		t.Errorf("Expected ID %s, got %s", expected, actual)
	}
}

func TestGetBuiltinMigrationsWithFormat(t *testing.T) {
	migrations := migrate.GetBuiltinMigrationsWithFormat("schema_migrations", migrate.NamingFormatNNN)

	if len(migrations) == 0 {
		t.Error("Expected at least one builtin migration")
	}

	firstMigration := migrations[0]
	if firstMigration.ID() != "2022_01_01_000_create_schema_migrations" {
		t.Errorf("Expected first migration to be create_schema_migrations with NNN format, got %s", firstMigration.ID())
	}
}
