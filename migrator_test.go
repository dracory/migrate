package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"

	_ "modernc.org/sqlite"
)

type mockMigration struct {
	id          string
	description string
	upFunc      func(ctx context.Context, tx *sql.Tx) error
	downFunc    func(ctx context.Context, tx *sql.Tx) error
}

func (m *mockMigration) ID() string                                 { return m.id }
func (m *mockMigration) Description() string                        { return m.description }
func (m *mockMigration) Up(ctx context.Context, tx *sql.Tx) error   { return m.upFunc(ctx, tx) }
func (m *mockMigration) Down(ctx context.Context, tx *sql.Tx) error { return m.downFunc(ctx, tx) }

func TestAddMigration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	t.Run("adds valid migration", func(t *testing.T) {
		m := New(db, nil)
		migration := &mockMigration{
			id:          "2026_03_21_001_test",
			description: "Test migration",
			upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
			downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
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
			upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
			downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
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
			upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
			downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
		}
		migration2 := &mockMigration{
			id:          "2026_03_21_001_test",
			description: "Test migration 2",
			upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
			downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
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
				upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
				downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
			},
			&mockMigration{
				id:          "2026_03_21_002_test",
				description: "Test 2",
				upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
				downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
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
				upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
				downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
			},
			nil,
			&mockMigration{
				id:          "2026_03_21_003_test",
				description: "Test 3",
				upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
				downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
			},
		}

		err := m.AddMigrations(migrations)
		if err == nil {
			t.Error("Expected error for nil migration")
		}
	})
}

func TestConcurrentMigrationAddition(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m := New(db, nil)

	// Add migrations concurrently from multiple goroutines
	const numGoroutines = 10
	const migrationsPerGoroutine = 5

	errChan := make(chan error, numGoroutines)
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < migrationsPerGoroutine; j++ {
				migrationID := fmt.Sprintf("2026_03_21_%03d_test", goroutineID*migrationsPerGoroutine+j)
				migration := &mockMigration{
					id:          migrationID,
					description: fmt.Sprintf("Test migration %d", goroutineID*migrationsPerGoroutine+j),
					upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
					downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
				}

				if err := m.AddMigration(migration); err != nil {
					errChan <- fmt.Errorf("goroutine %d: %w", goroutineID, err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Error(err)
	}
}

func TestAddMigrationInternal(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m := New(db, nil).(*migratorImplementation)

	t.Run("adds valid migration", func(t *testing.T) {
		m.mu.Lock()
		defer m.mu.Unlock()

		migration := &mockMigration{
			id:          "2026_03_21_001_test",
			description: "Test migration",
			upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
			downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
		}

		err := m.addMigrationInternal(migration)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(m.migrations) != 1 {
			t.Errorf("Expected 1 migration, got %d", len(m.migrations))
		}
	})

	t.Run("rejects nil migration", func(t *testing.T) {
		m.mu.Lock()
		defer m.mu.Unlock()

		err := m.addMigrationInternal(nil)
		if err == nil {
			t.Error("Expected error for nil migration")
		}
	})

	t.Run("rejects migration with empty ID", func(t *testing.T) {
		m.mu.Lock()
		defer m.mu.Unlock()

		migration := &mockMigration{
			id:          "",
			description: "Test",
			upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
			downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
		}

		err := m.addMigrationInternal(migration)
		if err == nil {
			t.Error("Expected error for empty ID")
		}
	})

	t.Run("rejects duplicate migration ID", func(t *testing.T) {
		m := New(db, nil).(*migratorImplementation)
		m.mu.Lock()
		defer m.mu.Unlock()

		migration1 := &mockMigration{
			id:          "2026_03_21_001_test",
			description: "Test 1",
			upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
			downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
		}

		migration2 := &mockMigration{
			id:          "2026_03_21_001_test",
			description: "Test 2",
			upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
			downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
		}

		err1 := m.addMigrationInternal(migration1)
		if err1 != nil {
			t.Fatalf("Expected no error for first migration, got %v", err1)
		}

		err2 := m.addMigrationInternal(migration2)
		if err2 == nil {
			t.Error("Expected error for duplicate migration ID")
		}
	})
}
