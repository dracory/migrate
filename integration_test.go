package migrate

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"

	"github.com/dracory/database"
	"github.com/dracory/sb"
	_ "modernc.org/sqlite"
)

func TestIntegrationFullMigrationCycle(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := New(db, &Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}

	userMigrations := []MigrationInterface{
		&mockMigration{
			id:          "2026_03_21_0001_create_users",
			description: "Create users table",
			upFunc: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.Exec(`CREATE TABLE users (
					id INTEGER PRIMARY KEY,
					email TEXT NOT NULL UNIQUE,
					created_at DATETIME NOT NULL
				)`)
				return err
			},
			downFunc: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.Exec("DROP TABLE users")
				return err
			},
		},
		&mockMigration{
			id:          "2026_03_21_0002_create_posts",
			description: "Create posts table",
			upFunc: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.Exec(`CREATE TABLE posts (
					id INTEGER PRIMARY KEY,
					title TEXT NOT NULL,
					user_id INTEGER NOT NULL
				)`)
				return err
			},
			downFunc: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.Exec("DROP TABLE posts")
				return err
			},
		},
	}

	if err := m.AddMigrations(userMigrations); err != nil {
		t.Fatalf("Failed to add user migrations: %v", err)
	}

	t.Run("runs all migrations in order", func(t *testing.T) {
		err := m.Up(context.Background())
		if err != nil {
			t.Fatalf("Failed to run migrations: %v", err)
		}

		var count int
		dialect := database.DatabaseType(db)
		countSQL, countParams, err := sb.NewBuilder(dialect).
			Table(DefaultTableName).
			Select([]string{"COUNT(*)"})
		if err != nil {
			t.Fatalf("Failed to build count SQL: %v", err)
		}
		err = db.QueryRow(countSQL, countParams...).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query migrations table: %v", err)
		}
		if count != 3 {
			t.Errorf("Expected 3 migrations (1 builtin + 2 user), got %d", count)
		}

		var tableName string
		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Scan(&tableName)
		if err != nil {
			t.Fatalf("Users table should exist: %v", err)
		}

		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='posts'").Scan(&tableName)
		if err != nil {
			t.Fatalf("Posts table should exist: %v", err)
		}
	})

	t.Run("running Up again does nothing", func(t *testing.T) {
		err := m.Up(context.Background())
		if err != nil {
			t.Fatalf("Failed to run migrations again: %v", err)
		}

		var count int
		dialect := database.DatabaseType(db)
		countSQL, countParams, err := sb.NewBuilder(dialect).
			Table(DefaultTableName).
			Select([]string{"COUNT(*)"})
		if err != nil {
			t.Fatalf("Failed to build count SQL: %v", err)
		}
		err = db.QueryRow(countSQL, countParams...).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query migrations table: %v", err)
		}
		if count != 3 {
			t.Errorf("Expected still 3 migrations, got %d", count)
		}
	})

	t.Run("rolls back last migration", func(t *testing.T) {
		err := m.Down(context.Background())
		if err != nil {
			t.Fatalf("Failed to rollback: %v", err)
		}

		var count int
		dialect := database.DatabaseType(db)
		countSQL, countParams, err := sb.NewBuilder(dialect).
			Table(DefaultTableName).
			Select([]string{"COUNT(*)"})
		if err != nil {
			t.Fatalf("Failed to build count SQL: %v", err)
		}
		err = db.QueryRow(countSQL, countParams...).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query migrations table: %v", err)
		}
		if count != 2 {
			t.Errorf("Expected 2 migrations after rollback, got %d", count)
		}
	})

	t.Run("can re-run rolled back migration", func(t *testing.T) {
		// Create a fresh database for this test to avoid column conflicts
		db2 := setupTestDB(t)
		defer db2.Close()

		m2, err := New(db2, &Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
		if err != nil {
			t.Fatalf("Failed to create migrator: %v", err)
		}
		if err := m2.AddMigrations(userMigrations); err != nil {
			t.Fatalf("Failed to add user migrations: %v", err)
		}

		// Run all migrations
		if err := m2.Up(context.Background()); err != nil {
			t.Fatalf("Failed to run migrations: %v", err)
		}

		// Roll back last one
		if err := m2.Down(context.Background()); err != nil {
			t.Fatalf("Failed to rollback: %v", err)
		}

		// Re-run
		err = m2.Up(context.Background())
		if err != nil {
			t.Fatalf("Failed to re-run migration: %v", err)
		}

		var count int
		dialect := database.DatabaseType(db2)
		countSQL, countParams, err := sb.NewBuilder(dialect).
			Table(DefaultTableName).
			Select([]string{"COUNT(*)"})
		if err != nil {
			t.Fatalf("Failed to build count SQL: %v", err)
		}
		err = db2.QueryRow(countSQL, countParams...).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query migrations table: %v", err)
		}
		if count != 3 {
			t.Errorf("Expected 3 migrations after re-run, got %d", count)
		}
	})
}

func TestIntegrationMigrationOrdering(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := New(db, &Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}

	executionOrder := []string{}

	migrations := []MigrationInterface{
		&mockMigration{
			id:          "2026_03_22_0003_third",
			description: "Third",
			upFunc: func(ctx context.Context, tx *sql.Tx) error {
				executionOrder = append(executionOrder, "third")
				return nil
			},
			downFunc: func(ctx context.Context, tx *sql.Tx) error { return nil },
		},
		&mockMigration{
			id:          "2026_03_21_0001_first",
			description: "First",
			upFunc: func(ctx context.Context, tx *sql.Tx) error {
				executionOrder = append(executionOrder, "first")
				return nil
			},
			downFunc: func(ctx context.Context, tx *sql.Tx) error { return nil },
		},
		&mockMigration{
			id:          "2026_03_21_0002_second",
			description: "Second",
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

	err = m.Up(context.Background())
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	if len(executionOrder) != 3 {
		t.Fatalf("Expected 3 user migrations to run, got %d", len(executionOrder))
	}

	if executionOrder[0] != "first" {
		t.Errorf("Expected first migration to run first, got %s", executionOrder[0])
	}
	if executionOrder[1] != "second" {
		t.Errorf("Expected second migration to run second, got %s", executionOrder[1])
	}
	if executionOrder[2] != "third" {
		t.Errorf("Expected third migration to run third, got %s", executionOrder[2])
	}

	var firstID string
	err = db.QueryRow("SELECT id FROM " + DefaultTableName + " ORDER BY completed_at LIMIT 1").Scan(&firstID)
	if err != nil {
		t.Fatalf("Failed to query first migration: %v", err)
	}
	if firstID != BuiltinMigrationID {
		t.Errorf("Expected builtin migration to be first, got %s", firstID)
	}
}

func TestIntegrationWithCustomMigrationTableName(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	customMigrationTableName := "custom_migrations_table"
	m, err := New(db, &Options{
		MigrationTableName: customMigrationTableName,
		Logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}

	migration := &mockMigration{
		id:          "2026_03_21_0001_test",
		description: "Test",
		upFunc:      func(ctx context.Context, tx *sql.Tx) error { return nil },
		downFunc:    func(ctx context.Context, tx *sql.Tx) error { return nil },
	}

	if err := m.AddMigration(migration); err != nil {
		t.Fatalf("Failed to add migration: %v", err)
	}

	if err := m.Up(context.Background()); err != nil {
		t.Fatalf("Failed to run migration: %v", err)
	}

	var count int
	dialect := database.DatabaseType(db)
	countSQL, countParams, err := sb.NewBuilder(dialect).
		Table(customMigrationTableName).
		Select([]string{"COUNT(*)"})
	if err != nil {
		t.Fatalf("Failed to build count SQL: %v", err)
	}
	err = db.QueryRow(countSQL, countParams...).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query custom table: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 migrations (1 builtin + 1 user) in custom table, got %d", count)
	}
}

func TestIntegrationTransactionRollback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := New(db, &Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}

	migration1 := &mockMigration{
		id:          "2026_03_21_0001_create_table",
		description: "Create test table",
		upFunc: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.Exec("CREATE TABLE test_rollback (id INTEGER PRIMARY KEY)")
			return err
		},
		downFunc: func(ctx context.Context, tx *sql.Tx) error { return nil },
	}

	migration2 := &mockMigration{
		id:          "2026_03_21_0002_fail",
		description: "Failing migration",
		upFunc: func(ctx context.Context, tx *sql.Tx) error {
			_, err := tx.Exec("INSERT INTO test_rollback (id) VALUES (1)")
			if err != nil {
				return err
			}
			return sql.ErrConnDone
		},
		downFunc: func(ctx context.Context, tx *sql.Tx) error { return nil },
	}

	if err := m.AddMigrations([]MigrationInterface{migration1, migration2}); err != nil {
		t.Fatalf("Failed to add migrations: %v", err)
	}

	err = m.Up(context.Background())
	if err == nil {
		t.Error("Expected error from failing migration")
	}

	var count int
	dialect := database.DatabaseType(db)
	countSQL, countParams, err := sb.NewBuilder(dialect).
		Table(DefaultTableName).
		Where(&sb.Where{
			Column:   ColumnID,
			Operator: "=",
			Value:    migration2.ID(),
		}).
		Select([]string{"COUNT(*)"})
	if err != nil {
		t.Fatalf("Failed to build count SQL: %v", err)
	}
	err = db.QueryRow(countSQL, countParams...).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query migrations table: %v", err)
	}
	if count != 0 {
		t.Error("Failed migration should not be recorded")
	}

	err = db.QueryRow("SELECT COUNT(*) FROM test_rollback").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query test table: %v", err)
	}
	if count != 0 {
		t.Error("Transaction should have been rolled back, table should be empty")
	}
}
