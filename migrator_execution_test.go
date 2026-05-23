package migrate

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/dracory/database"
	"github.com/dracory/sb"
	carbon "github.com/dromara/carbon/v2"
	_ "modernc.org/sqlite"
)

func TestUp_RunsPendingMigrationsInOrder(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if err := createTestTable(db, DefaultTableName); err != nil {
		t.Fatalf("Failed to create migrations table: %v", err)
	}

	m, err := New(db, &Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}

	executionOrder := []string{}

	migrations := []MigrationInterface{
		&mockMigration{
			id:          "2026_03_21_0003_third",
			description: "Third migration",
			upFunc: func(ctx context.Context, tx *sql.Tx) error {
				executionOrder = append(executionOrder, "third")
				return nil
			},
			downFunc: func(ctx context.Context, tx *sql.Tx) error { return nil },
		},
		&mockMigration{
			id:          "2026_03_21_0001_first",
			description: "First migration",
			upFunc: func(ctx context.Context, tx *sql.Tx) error {
				executionOrder = append(executionOrder, "first")
				return nil
			},
			downFunc: func(ctx context.Context, tx *sql.Tx) error { return nil },
		},
		&mockMigration{
			id:          "2026_03_21_0002_second",
			description: "Second migration",
			upFunc: func(ctx context.Context, tx *sql.Tx) error {
				executionOrder = append(executionOrder, "second")
				return nil
			},
			downFunc: func(ctx context.Context, tx *sql.Tx) error { return nil },
		},
	}

	if err := m.AddMigrations(migrations); err != nil {
		t.Fatalf("Failed to add migrations: %v", err)
	}

	if err := m.Up(context.Background()); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(executionOrder) != 3 {
		t.Fatalf("Expected 3 migrations to run, got %d", len(executionOrder))
	}

	if executionOrder[0] != "first" || executionOrder[1] != "second" || executionOrder[2] != "third" {
		t.Errorf("Expected order [first, second, third], got %v", executionOrder)
	}
}

func TestUp_SkipsAlreadyAppliedMigrations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if err := createTestTable(db, DefaultTableName); err != nil {
		t.Fatalf("Failed to create migrations table: %v", err)
	}

	dialect := database.DatabaseType(db)
	querySQL, params, err := sb.NewBuilder(dialect).
		Table(DefaultTableName).
		Insert(map[string]string{
			ColumnID:          "2026_03_21_0001_first",
			ColumnBatch:       "20260321120000",
			ColumnDescription: "First migration",
			ColumnStartedAt:   carbon.Now(carbon.UTC).ToDateTimeString(),
			ColumnCompletedAt: carbon.Now(carbon.UTC).ToDateTimeString(),
		})

	if err != nil {
		t.Fatalf("Failed to build insert SQL: %v", err)
	}

	_, err = db.Exec(querySQL, params...)
	if err != nil {
		t.Fatalf("Failed to insert test migration: %v", err)
	}

	m, err := New(db, &Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}

	executionOrder := []string{}

	migrations := []MigrationInterface{
		&mockMigration{
			id:          "2026_03_21_0001_first",
			description: "First migration",
			upFunc: func(ctx context.Context, tx *sql.Tx) error {
				executionOrder = append(executionOrder, "first")
				return nil
			},
			downFunc: func(ctx context.Context, tx *sql.Tx) error { return nil },
		},
		&mockMigration{
			id:          "2026_03_21_0002_second",
			description: "Second migration",
			upFunc: func(ctx context.Context, tx *sql.Tx) error {
				executionOrder = append(executionOrder, "second")
				return nil
			},
			downFunc: func(ctx context.Context, tx *sql.Tx) error { return nil },
		},
	}

	if err := m.AddMigrations(migrations); err != nil {
		t.Fatalf("Failed to add migrations: %v", err)
	}

	if err := m.Up(context.Background()); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(executionOrder) != 1 {
		t.Fatalf("Expected 1 migration to run, got %d", len(executionOrder))
	}

	if executionOrder[0] != "second" {
		t.Errorf("Expected 'second' to run, got %v", executionOrder)
	}
}

func TestUp_StopsOnFirstError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if err := createTestTable(db, DefaultTableName); err != nil {
		t.Fatalf("Failed to create migrations table: %v", err)
	}

	m, err := New(db, &Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}

	executionOrder := []string{}

	migrations := []MigrationInterface{
		&mockMigration{
			id:          "2026_03_21_0001_first",
			description: "First migration",
			upFunc: func(ctx context.Context, tx *sql.Tx) error {
				executionOrder = append(executionOrder, "first")
				return nil
			},
			downFunc: func(ctx context.Context, tx *sql.Tx) error { return nil },
		},
		&mockMigration{
			id:          "2026_03_21_0002_second",
			description: "Second migration (fails)",
			upFunc: func(ctx context.Context, tx *sql.Tx) error {
				executionOrder = append(executionOrder, "second")
				return errors.New("migration failed")
			},
			downFunc: func(ctx context.Context, tx *sql.Tx) error { return nil },
		},
		&mockMigration{
			id:          "2026_03_21_0003_third",
			description: "Third migration",
			upFunc: func(ctx context.Context, tx *sql.Tx) error {
				executionOrder = append(executionOrder, "third")
				return nil
			},
			downFunc: func(ctx context.Context, tx *sql.Tx) error { return nil },
		},
	}

	if err := m.AddMigrations(migrations); err != nil {
		t.Fatalf("Failed to add migrations: %v", err)
	}

	err = m.Up(context.Background())
	if err == nil {
		t.Error("Expected error from failing migration")
	}

	if len(executionOrder) != 2 {
		t.Fatalf("Expected 2 migrations to run, got %d", len(executionOrder))
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM "+DefaultTableName+" WHERE id = ?", "2026_03_21_0001_first").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query migrations table: %v", err)
	}
	if count != 1 {
		t.Error("Expected first migration to be recorded")
	}

	err = db.QueryRow("SELECT COUNT(*) FROM "+DefaultTableName+" WHERE id = ?", "2026_03_21_0002_second").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query migrations table: %v", err)
	}
	if count != 0 {
		t.Error("Expected second migration to NOT be recorded (rolled back)")
	}
}

func TestUp_HandlesEmptyMigrationList(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if err := createTestTable(db, DefaultTableName); err != nil {
		t.Fatalf("Failed to create migrations table: %v", err)
	}

	m, err := New(db, &Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}

	err = m.Up(context.Background())
	if err != nil {
		t.Errorf("Expected no error for empty migration list, got %v", err)
	}
}

func TestDown_RollsBackLastAppliedMigration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if err := createTestTable(db, DefaultTableName); err != nil {
		t.Fatalf("Failed to create migrations table: %v", err)
	}

	m, err := New(db, &Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}

	migrations := []MigrationInterface{
		&mockMigration{
			id:          "2026_03_21_0001_first",
			description: "First migration",
			upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
			downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
		},
		&mockMigration{
			id:          "2026_03_21_0002_second",
			description: "Second migration",
			upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
			downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
		},
	}

	if err := m.AddMigrations(migrations); err != nil {
		t.Fatalf("Failed to add migrations: %v", err)
	}

	if err := m.Up(context.Background()); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	if err := m.Down(context.Background()); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM "+DefaultTableName+" WHERE id = ?", "2026_03_21_0002_second").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query migrations table: %v", err)
	}
	if count != 0 {
		t.Error("Expected second migration to be rolled back")
	}

	err = db.QueryRow("SELECT COUNT(*) FROM "+DefaultTableName+" WHERE id = ?", "2026_03_21_0001_first").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query migrations table: %v", err)
	}
	if count != 1 {
		t.Error("Expected first migration to still be applied")
	}
}

func TestDown_HandlesNoMigrationsToRollback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if err := createTestTable(db, DefaultTableName); err != nil {
		t.Fatalf("Failed to create migrations table: %v", err)
	}

	m, err := New(db, &Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}

	err = m.Down(context.Background())
	if err != nil {
		t.Errorf("Expected no error when no migrations to rollback, got %v", err)
	}
}

func TestDown_HandlesRollbackError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if err := createTestTable(db, DefaultTableName); err != nil {
		t.Fatalf("Failed to create migrations table: %v", err)
	}

	m, err := New(db, &Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}

	migration := &mockMigration{
		id:          "2026_03_21_0001_test",
		description: "Test migration",
		upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
		downFunc: func(ctx context.Context, tx *sql.Tx) error {
			return errors.New("rollback failed")
		},
	}

	if err := m.AddMigration(migration); err != nil {
		t.Fatalf("Failed to add migration: %v", err)
	}

	if err := m.Up(context.Background()); err != nil {
		t.Fatalf("Failed to run migration: %v", err)
	}

	err = m.Down(context.Background())
	if err == nil {
		t.Error("Expected error from failing rollback")
	}
}

func TestStatus_ShowsMigrationStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if err := createTestTable(db, DefaultTableName); err != nil {
		t.Fatalf("Failed to create migrations table: %v", err)
	}

	m, err := New(db, &Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}

	migrations := []MigrationInterface{
		&mockMigration{
			id:          "2026_03_21_0001_first",
			description: "First migration",
			upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
			downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
		},
		&mockMigration{
			id:          "2026_03_21_0002_second",
			description: "Second migration",
			upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
			downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
		},
	}

	if err := m.AddMigrations(migrations); err != nil {
		t.Fatalf("Failed to add migrations: %v", err)
	}

	dialect := database.DatabaseType(db)
	querySQL, params, err := sb.NewBuilder(dialect).
		Table(DefaultTableName).
		Insert(map[string]string{
			ColumnID:          "2026_03_21_0001_first",
			ColumnBatch:       "20260321120000",
			ColumnDescription: "First migration",
			ColumnStartedAt:   carbon.Now(carbon.UTC).ToDateTimeString(),
			ColumnCompletedAt: carbon.Now(carbon.UTC).ToDateTimeString(),
		})

	if err != nil {
		t.Fatalf("Failed to build insert SQL: %v", err)
	}

	_, err = db.Exec(querySQL, params...)
	if err != nil {
		t.Fatalf("Failed to insert test migration: %v", err)
	}

	if err := m.Status(context.Background()); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
