package migrate_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/dracory/migrate"

	_ "modernc.org/sqlite"
)

type testMigration struct {
	id          string
	description string
	upFunc      func(ctx context.Context, tx *sql.Tx) error
	downFunc    func(ctx context.Context, tx *sql.Tx) error
}

func (m *testMigration) ID() string                                 { return m.id }
func (m *testMigration) Description() string                        { return m.description }
func (m *testMigration) Up(ctx context.Context, tx *sql.Tx) error   { return m.upFunc(ctx, tx) }
func (m *testMigration) Down(ctx context.Context, tx *sql.Tx) error { return m.downFunc(ctx, tx) }

func TestMigrator_Up_RunsMigrationSuccessfully(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := migrate.New(db, &migrate.Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}

	executed := false
	migration := &testMigration{
		id:          "2026_03_21_1200_test",
		description: "Test migration",
		upFunc: func(ctx context.Context, tx *sql.Tx) error {
			executed = true
			_, err := tx.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY)")
			return err
		},
		downFunc: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.Exec("DROP TABLE test_table")
			return err
		},
	}

	if err := m.AddMigration(migration); err != nil {
		t.Fatalf("Failed to add migration: %v", err)
	}

	if err := m.Up(context.Background()); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !executed {
		t.Error("Expected migration Up to be executed")
	}

	status, err := m.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if len(status) != 2 {
		t.Errorf("Expected 2 migration statuses (1 builtin + 1 user), got %d", len(status))
	}

	if !status[0].Applied {
		t.Error("Expected migration to be marked as applied")
	}
}

func TestMigrator_Up_RollsBackOnMigrationError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := migrate.New(db, &migrate.Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}

	migration := &testMigration{
		id:          "2026_03_21_1300_test",
		description: "Failing migration",
		upFunc: func(ctx context.Context, tx *sql.Tx) error {
			return errors.New("migration failed")
		},
		downFunc: func(ctx context.Context, tx *sql.Tx) error { return nil },
	}

	if err := m.AddMigration(migration); err != nil {
		t.Fatalf("Failed to add migration: %v", err)
	}

	if err := m.Up(context.Background()); err == nil {
		t.Error("Expected error from failing migration")
	}

	status, err := m.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if len(status) != 2 {
		t.Errorf("Expected 2 migration statuses (1 builtin + 1 user), got %d", len(status))
	}

	// Check that the builtin migration is applied but the user migration is not
	builtinApplied := false
	userApplied := false
	for _, s := range status {
		if s.ID == "2022_01_01_0000_create_schema_migrations" && s.Applied {
			builtinApplied = true
		}
		if s.ID == migration.ID() && s.Applied {
			userApplied = true
		}
	}

	if !builtinApplied {
		t.Error("Expected builtin migration to be applied")
	}
	if userApplied {
		t.Error("Expected user migration to not be marked as applied after failure")
	}
}

func TestMigrator_Down_RollsBackLastMigration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := migrate.New(db, &migrate.Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}

	executed := false
	migration := &testMigration{
		id:          "2026_03_21_1400_test",
		description: "Test migration",
		upFunc: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY)")
			return err
		},
		downFunc: func(ctx context.Context, tx *sql.Tx) error {
			executed = true
			_, err := tx.Exec("DROP TABLE test_table")
			return err
		},
	}

	if err := m.AddMigration(migration); err != nil {
		t.Fatalf("Failed to add migration: %v", err)
	}

	if err := m.Up(context.Background()); err != nil {
		t.Fatalf("Failed to run Up: %v", err)
	}

	if err := m.Down(context.Background()); err != nil {
		t.Fatalf("Failed to run Down: %v", err)
	}

	if !executed {
		t.Error("Expected migration Down to be executed")
	}

	status, err := m.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if len(status) != 2 {
		t.Errorf("Expected 2 migration statuses (1 builtin + 1 user), got %d", len(status))
	}

	// Check that the builtin migration is still applied but the user migration is not
	builtinApplied := false
	userApplied := false
	for _, s := range status {
		if s.ID == "2022_01_01_0000_create_schema_migrations" && s.Applied {
			builtinApplied = true
		}
		if s.ID == migration.ID() && s.Applied {
			userApplied = true
		}
	}

	if !builtinApplied {
		t.Error("Expected builtin migration to still be applied after Down")
	}
	if userApplied {
		t.Error("Expected user migration to not be marked as applied after Down")
	}
}

func TestMigrator_GetStatus_ReturnsCorrectStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := migrate.New(db, &migrate.Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}

	migration1 := &testMigration{
		id:          "2026_03_21_1500_test1",
		description: "Test migration 1",
		upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
		downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
	}

	migration2 := &testMigration{
		id:          "2026_03_21_1600_test2",
		description: "Test migration 2",
		upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
		downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
	}

	if err := m.AddMigrations([]migrate.MigrationInterface{migration1, migration2}); err != nil {
		t.Fatalf("Failed to add migrations: %v", err)
	}

	if err := m.Up(context.Background()); err != nil {
		t.Fatalf("Failed to run Up: %v", err)
	}

	status, err := m.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if len(status) != 3 {
		t.Errorf("Expected 3 migration statuses (1 builtin + 2 user), got %d", len(status))
	}

	for _, s := range status {
		if !s.Applied {
			t.Errorf("Expected migration %s to be applied", s.ID)
		}
	}
}

func TestMigrator_GetHistory_ReturnsMigrationHistory(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := migrate.New(db, &migrate.Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}

	migration := &testMigration{
		id:          "2026_03_21_1700_test",
		description: "Test migration",
		upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
		downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
	}

	if err := m.AddMigration(migration); err != nil {
		t.Fatalf("Failed to add migration: %v", err)
	}

	if err := m.Up(context.Background()); err != nil {
		t.Fatalf("Failed to run Up: %v", err)
	}

	history, err := m.GetHistory(context.Background())
	if err != nil {
		t.Fatalf("Failed to get history: %v", err)
	}

	if len(history) != 2 {
		t.Errorf("Expected 2 history records (1 builtin + 1 user), got %d", len(history))
	}

	// The builtin migration should be first
	if history[0].ID != "2022_01_01_0000_create_schema_migrations" {
		t.Errorf("Expected first history record to be builtin migration, got %s", history[0].ID)
	}

	// The user migration should be second
	if history[1].ID != migration.ID() {
		t.Errorf("Expected second history record to be user migration %s, got %s", migration.ID(), history[1].ID)
	}

	if history[1].Description != migration.Description() {
		t.Errorf("Expected description %s, got %s", migration.Description(), history[1].Description)
	}
}

// Interface tests - migration registration

func TestAddMigration_AddsValidMigrationWithHHMMFormat(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := migrate.New(db, nil)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	migration := &testMigration{
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

	opts := &migrate.Options{
		NamingFormatPrefix: migrate.NamingFormatPrefixYYYY_MM_DD_NNN,
	}
	m, err := migrate.New(db, opts)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	migration := &testMigration{
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

	m, err := migrate.New(db, nil)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	migration := &testMigration{
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

	opts := &migrate.Options{
		NamingFormatPrefix: migrate.NamingFormatPrefixYYYY_MM_DD_NNN,
	}
	m, err := migrate.New(db, opts)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	migration := &testMigration{
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

	m, err := migrate.New(db, nil)
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

	m, err := migrate.New(db, nil)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	migration := &testMigration{
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

	m, err := migrate.New(db, nil)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	migration1 := &testMigration{
		id:          "2026_03_21_1200_test",
		description: "Test migration 1",
		upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
		downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
	}
	migration2 := &testMigration{
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

	m, err := migrate.New(db, nil)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	migrations := []migrate.MigrationInterface{
		&testMigration{
			id:          "2026_03_21_1200_test",
			description: "Test 1",
			upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
			downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
		},
		&testMigration{
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

	m, err := migrate.New(db, nil)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	migrations := []migrate.MigrationInterface{
		&testMigration{
			id:          "2026_03_21_1200_test",
			description: "Test 1",
			upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
			downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
		},
		nil,
		&testMigration{
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

	m, err := migrate.New(db, nil)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}

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
				migration := &testMigration{
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

	for err := range errChan {
		t.Error(err)
	}
}
