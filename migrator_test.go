package migrate

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

type mockMigration struct {
	id          string
	description string
	upFunc      func(*sql.Tx) error
	downFunc    func(*sql.Tx) error
}

func (m *mockMigration) ID() string            { return m.id }
func (m *mockMigration) Description() string   { return m.description }
func (m *mockMigration) Up(tx *sql.Tx) error   { return m.upFunc(tx) }
func (m *mockMigration) Down(tx *sql.Tx) error { return m.downFunc(tx) }

func TestAddMigration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	t.Run("adds valid migration", func(t *testing.T) {
		m := New(db, nil)
		migration := &mockMigration{
			id:          "2026_03_21_001_test",
			description: "Test migration",
			upFunc:      func(tx *sql.Tx) error { return nil },
			downFunc:    func(tx *sql.Tx) error { return nil },
		}

		err := m.AddMigration(migration)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})

	t.Run("rejects nil migration", func(t *testing.T) {
		m := New(db, nil)
		err := m.AddMigration(nil)
		if err == nil {
			t.Error("Expected error for nil migration")
		}
		if err.Error() != "migration cannot be nil" {
			t.Errorf("Expected 'migration cannot be nil' error, got %v", err)
		}
	})

	t.Run("rejects migration with empty ID", func(t *testing.T) {
		m := New(db, nil)
		migration := &mockMigration{
			id:          "",
			description: "Test",
			upFunc:      func(tx *sql.Tx) error { return nil },
			downFunc:    func(tx *sql.Tx) error { return nil },
		}

		err := m.AddMigration(migration)
		if err == nil {
			t.Error("Expected error for empty migration ID")
		}
		if err.Error() != "migration ID cannot be empty" {
			t.Errorf("Expected 'migration ID cannot be empty' error, got %v", err)
		}
	})

	t.Run("rejects duplicate migration ID", func(t *testing.T) {
		m := New(db, nil)
		migration1 := &mockMigration{
			id:          "2026_03_21_001_test",
			description: "Test migration 1",
			upFunc:      func(tx *sql.Tx) error { return nil },
			downFunc:    func(tx *sql.Tx) error { return nil },
		}
		migration2 := &mockMigration{
			id:          "2026_03_21_001_test",
			description: "Test migration 2",
			upFunc:      func(tx *sql.Tx) error { return nil },
			downFunc:    func(tx *sql.Tx) error { return nil },
		}

		err := m.AddMigration(migration1)
		if err != nil {
			t.Fatalf("Expected no error for first migration, got %v", err)
		}

		err = m.AddMigration(migration2)
		if err == nil {
			t.Error("Expected error for duplicate migration ID")
		}
		if err.Error() != "migration with ID 2026_03_21_001_test already exists" {
			t.Errorf("Unexpected error message: %v", err)
		}
	})
}

func TestAddMigrations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	t.Run("adds multiple valid migrations", func(t *testing.T) {
		m := New(db, nil)
		migrations := []MigrationInterface{
			&mockMigration{
				id:          "2026_03_21_001_test",
				description: "Test 1",
				upFunc:      func(tx *sql.Tx) error { return nil },
				downFunc:    func(tx *sql.Tx) error { return nil },
			},
			&mockMigration{
				id:          "2026_03_21_002_test",
				description: "Test 2",
				upFunc:      func(tx *sql.Tx) error { return nil },
				downFunc:    func(tx *sql.Tx) error { return nil },
			},
		}

		err := m.AddMigrations(migrations)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})

	t.Run("stops on first error", func(t *testing.T) {
		m := New(db, nil)
		migrations := []MigrationInterface{
			&mockMigration{
				id:          "2026_03_21_001_test",
				description: "Test 1",
				upFunc:      func(tx *sql.Tx) error { return nil },
				downFunc:    func(tx *sql.Tx) error { return nil },
			},
			nil,
			&mockMigration{
				id:          "2026_03_21_003_test",
				description: "Test 3",
				upFunc:      func(tx *sql.Tx) error { return nil },
				downFunc:    func(tx *sql.Tx) error { return nil },
			},
		}

		err := m.AddMigrations(migrations)
		if err == nil {
			t.Error("Expected error for nil migration")
		}
	})
}
