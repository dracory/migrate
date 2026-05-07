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

func TestUp(t *testing.T) {
	t.Run("runs pending migrations in order", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		if err := createTestTable(db, DefaultTableName); err != nil {
			t.Fatalf("Failed to create migrations table: %v", err)
		}

		m := New(db, &Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})

		executionOrder := []string{}

		migrations := []MigrationInterface{
			&mockMigration{
				id:          "2026_03_21_003_third",
				description: "Third migration",
				upFunc: func(ctx context.Context, tx *sql.Tx) error {
					executionOrder = append(executionOrder, "third")
					return nil
				},
				downFunc: func(ctx context.Context, tx *sql.Tx) error { return nil },
			},
			&mockMigration{
				id:          "2026_03_21_001_first",
				description: "First migration",
				upFunc: func(ctx context.Context, tx *sql.Tx) error {
					executionOrder = append(executionOrder, "first")
					return nil
				},
				downFunc: func(ctx context.Context, tx *sql.Tx) error { return nil },
			},
			&mockMigration{
				id:          "2026_03_21_002_second",
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
	})

	t.Run("skips already applied migrations", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		if err := createTestTable(db, DefaultTableName); err != nil {
			t.Fatalf("Failed to create migrations table: %v", err)
		}

		dialect := database.DatabaseType(db)
		querySQL, params, err := sb.NewBuilder(dialect).
			Table(DefaultTableName).
			Insert(map[string]string{
				ColumnID:          "2026_03_21_001_first",
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

		m := New(db, &Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})

		executionOrder := []string{}

		migrations := []MigrationInterface{
			&mockMigration{
				id:          "2026_03_21_001_first",
				description: "First migration",
				upFunc: func(ctx context.Context, tx *sql.Tx) error {
					executionOrder = append(executionOrder, "first")
					return nil
				},
				downFunc: func(ctx context.Context, tx *sql.Tx) error { return nil },
			},
			&mockMigration{
				id:          "2026_03_21_002_second",
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
	})

	t.Run("stops on first error", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		if err := createTestTable(db, DefaultTableName); err != nil {
			t.Fatalf("Failed to create migrations table: %v", err)
		}

		m := New(db, &Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})

		executionOrder := []string{}

		migrations := []MigrationInterface{
			&mockMigration{
				id:          "2026_03_21_001_first",
				description: "First migration",
				upFunc: func(ctx context.Context, tx *sql.Tx) error {
					executionOrder = append(executionOrder, "first")
					return nil
				},
				downFunc: func(ctx context.Context, tx *sql.Tx) error { return nil },
			},
			&mockMigration{
				id:          "2026_03_21_002_second",
				description: "Second migration (fails)",
				upFunc: func(ctx context.Context, tx *sql.Tx) error {
					executionOrder = append(executionOrder, "second")
					return errors.New("migration failed")
				},
				downFunc: func(ctx context.Context, tx *sql.Tx) error { return nil },
			},
			&mockMigration{
				id:          "2026_03_21_003_third",
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

		err := m.Up(context.Background())
		if err == nil {
			t.Error("Expected error from failing migration")
		}

		if len(executionOrder) != 2 {
			t.Fatalf("Expected 2 migrations to run, got %d", len(executionOrder))
		}

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM "+DefaultTableName+" WHERE id = ?", "2026_03_21_001_first").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query migrations table: %v", err)
		}
		if count != 1 {
			t.Error("Expected first migration to be recorded")
		}

		err = db.QueryRow("SELECT COUNT(*) FROM "+DefaultTableName+" WHERE id = ?", "2026_03_21_002_second").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query migrations table: %v", err)
		}
		if count != 0 {
			t.Error("Expected second migration to NOT be recorded (rolled back)")
		}
	})

	t.Run("handles empty migration list", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		if err := createTestTable(db, DefaultTableName); err != nil {
			t.Fatalf("Failed to create migrations table: %v", err)
		}

		m := New(db, &Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})

		err := m.Up(context.Background())
		if err != nil {
			t.Errorf("Expected no error for empty migration list, got %v", err)
		}
	})
}

func TestDown(t *testing.T) {
	t.Run("rolls back last applied migration", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		if err := createTestTable(db, DefaultTableName); err != nil {
			t.Fatalf("Failed to create migrations table: %v", err)
		}

		m := New(db, &Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})

		migrations := []MigrationInterface{
			&mockMigration{
				id:          "2026_03_21_001_first",
				description: "First migration",
				upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
				downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
			},
			&mockMigration{
				id:          "2026_03_21_002_second",
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
		err := db.QueryRow("SELECT COUNT(*) FROM "+DefaultTableName+" WHERE id = ?", "2026_03_21_002_second").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query migrations table: %v", err)
		}
		if count != 0 {
			t.Error("Expected second migration to be rolled back")
		}

		err = db.QueryRow("SELECT COUNT(*) FROM "+DefaultTableName+" WHERE id = ?", "2026_03_21_001_first").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query migrations table: %v", err)
		}
		if count != 1 {
			t.Error("Expected first migration to still be applied")
		}
	})

	t.Run("handles no migrations to rollback", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		if err := createTestTable(db, DefaultTableName); err != nil {
			t.Fatalf("Failed to create migrations table: %v", err)
		}

		m := New(db, &Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})

		err := m.Down(context.Background())
		if err != nil {
			t.Errorf("Expected no error when no migrations to rollback, got %v", err)
		}
	})

	t.Run("handles rollback error", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		if err := createTestTable(db, DefaultTableName); err != nil {
			t.Fatalf("Failed to create migrations table: %v", err)
		}

		m := New(db, &Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})

		migration := &mockMigration{
			id:          "2026_03_21_001_test",
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

		err := m.Down(context.Background())
		if err == nil {
			t.Error("Expected error from failing rollback")
		}
	})
}

func TestStatus(t *testing.T) {
	t.Run("shows migration status", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		if err := createTestTable(db, DefaultTableName); err != nil {
			t.Fatalf("Failed to create migrations table: %v", err)
		}

		m := New(db, &Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})

		migrations := []MigrationInterface{
			&mockMigration{
				id:          "2026_03_21_001_first",
				description: "First migration",
				upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
				downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
			},
			&mockMigration{
				id:          "2026_03_21_002_second",
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
				ColumnID:          "2026_03_21_001_first",
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
	})
}
