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

func TestAddMigration_AddsValidMigrationWithHHMMFormat(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := New(db, nil)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	migration := &mockMigration{
		id:          "2026_03_21_1200_test",
		description: "Test migration",
		upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
		downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
	}

	err = m.AddMigration(migration)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestAddMigration_AddsValidMigrationWithNNNFormat(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	opts := &Options{
		NamingFormat: NamingFormatNNN,
	}
	m, err := New(db, opts)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	migration := &mockMigration{
		id:          "2026_03_21_001_test",
		description: "Test migration",
		upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
		downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
	}

	err = m.AddMigration(migration)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestAddMigration_RejectsNNNFormatIDWhenUsingHHMMFormat(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := New(db, nil)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	migration := &mockMigration{
		id:          "2026_03_21_001_test",
		description: "Test migration",
		upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
		downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
	}

	err = m.AddMigration(migration)
	if err == nil {
		t.Fatal("Expected error for NNN format ID in HHMM mode")
	}
}

func TestAddMigration_RejectsHHMMFormatIDWhenUsingNNNFormat(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	opts := &Options{
		NamingFormat: NamingFormatNNN,
	}
	m, err := New(db, opts)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	migration := &mockMigration{
		id:          "2026_03_21_1200_test",
		description: "Test migration",
		upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
		downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
	}

	err = m.AddMigration(migration)
	if err == nil {
		t.Fatal("Expected error for HHMM format ID in NNN mode")
	}
}

func TestAddMigration_RejectsNilMigration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := New(db, nil)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	err = m.AddMigration(nil)
	if err == nil {
		t.Error("Expected error for nil migration")
	}
	if err.Error() != "migration cannot be nil" {
		t.Errorf("Expected 'migration cannot be nil' error, got %v", err)
	}
}

func TestAddMigration_RejectsMigrationWithEmptyID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := New(db, nil)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	migration := &mockMigration{
		id:          "",
		description: "Test",
		upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
		downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
	}

	err = m.AddMigration(migration)
	if err == nil {
		t.Error("Expected error for empty migration ID")
	}
	if err.Error() != "migration ID cannot be empty" {
		t.Errorf("Expected 'migration ID cannot be empty' error, got %v", err)
	}
}

func TestAddMigration_RejectsDuplicateMigrationID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := New(db, nil)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	migration1 := &mockMigration{
		id:          "2026_03_21_1200_test",
		description: "Test migration 1",
		upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
		downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
	}
	migration2 := &mockMigration{
		id:          "2026_03_21_1200_test",
		description: "Test migration 2",
		upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
		downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
	}

	if err := m.AddMigration(migration1); err != nil {
		t.Fatalf("Expected no error for first migration, got %v", err)
	}

	err = m.AddMigration(migration2)
	if err == nil {
		t.Error("Expected error for duplicate migration ID")
	}
	if err.Error() != "migration with ID 2026_03_21_1200_test already exists" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestAddMigrations_AddsMultipleValidMigrations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := New(db, nil)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	migrations := []MigrationInterface{
		&mockMigration{
			id:          "2026_03_21_1200_test",
			description: "Test 1",
			upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
			downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
		},
		&mockMigration{
			id:          "2026_03_21_1300_test",
			description: "Test 2",
			upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
			downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
		},
	}

	if err := m.AddMigrations(migrations); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestAddMigrations_StopsOnFirstError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := New(db, nil)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	migrations := []MigrationInterface{
		&mockMigration{
			id:          "2026_03_21_1200_test",
			description: "Test 1",
			upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
			downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
		},
		nil,
		&mockMigration{
			id:          "2026_03_21_1400_test",
			description: "Test 3",
			upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
			downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
		},
	}

	if err := m.AddMigrations(migrations); err == nil {
		t.Error("Expected error for nil migration")
	}
}

func TestConcurrentMigrationAddition(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := New(db, nil)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}

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
				hour := (goroutineID*migrationsPerGoroutine + j) / 60
				minute := (goroutineID*migrationsPerGoroutine + j) % 60
				migrationID := fmt.Sprintf("2026_03_21_%02d%02d_test", hour, minute)
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

func TestAddMigrationInternal_AddsValidMigration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := New(db, nil)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	mImpl := m.(*migratorImplementation)
	mImpl.mu.Lock()
	defer mImpl.mu.Unlock()

	migration := &mockMigration{
		id:          "2026_03_21_1200_test",
		description: "Test migration",
		upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
		downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
	}

	err = mImpl.addMigrationInternal(migration)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(mImpl.migrations) != 1 {
		t.Errorf("Expected 1 migration, got %d", len(mImpl.migrations))
	}
}

func TestAddMigrationInternal_RejectsNilMigration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := New(db, nil)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	mImpl := m.(*migratorImplementation)
	mImpl.mu.Lock()
	defer mImpl.mu.Unlock()

	err = mImpl.addMigrationInternal(nil)
	if err == nil {
		t.Error("Expected error for nil migration")
	}
}

func TestAddMigrationInternal_RejectsMigrationWithEmptyID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := New(db, nil)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	mImpl := m.(*migratorImplementation)
	mImpl.mu.Lock()
	defer mImpl.mu.Unlock()

	migration := &mockMigration{
		id:          "",
		description: "Test",
		upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
		downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
	}

	err = mImpl.addMigrationInternal(migration)
	if err == nil {
		t.Error("Expected error for empty ID")
	}
}

func TestAddMigrationInternal_RejectsDuplicateMigrationID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m2, err := New(db, nil)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	mImpl2 := m2.(*migratorImplementation)
	mImpl2.mu.Lock()
	defer mImpl2.mu.Unlock()

	migration1 := &mockMigration{
		id:          "2026_03_21_1200_test",
		description: "Test 1",
		upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
		downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
	}

	migration2 := &mockMigration{
		id:          "2026_03_21_1200_test",
		description: "Test 2",
		upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
		downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
	}

	err1 := mImpl2.addMigrationInternal(migration1)
	if err1 != nil {
		t.Fatalf("Expected no error for first migration, got %v", err1)
	}

	err2 := mImpl2.addMigrationInternal(migration2)
	if err2 == nil {
		t.Error("Expected error for duplicate migration ID")
	}
}
