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

func createTestTable(db *sql.DB, tableName string) error {
	dialect := database.DatabaseType(db)
	sql, err := sb.NewBuilder(dialect).
		Table(tableName).
		Column(sb.Column{
			Name:       ColumnID,
			Type:       sb.COLUMN_TYPE_STRING,
			Length:     255,
			PrimaryKey: true,
		}).
		Column(sb.Column{
			Name:     ColumnBatch,
			Type:     sb.COLUMN_TYPE_INTEGER,
			Nullable: false,
		}).
		Column(sb.Column{
			Name:     ColumnDescription,
			Type:     sb.COLUMN_TYPE_TEXT,
			Nullable: false,
		}).
		Column(sb.Column{
			Name:     ColumnStartedAt,
			Type:     sb.COLUMN_TYPE_DATETIME,
			Nullable: false,
		}).
		Column(sb.Column{
			Name:     ColumnCompletedAt,
			Type:     sb.COLUMN_TYPE_DATETIME,
			Nullable: false,
		}).
		CreateIfNotExists()

	if err != nil {
		return err
	}

	_, err = db.Exec(sql)
	return err
}

func TestGetAppliedMigrations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	m, err := New(db, &Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	mImpl := m.(*migratorImplementation)

	t.Run("returns empty map when table doesn't exist", func(t *testing.T) {
		applied, err := mImpl.getAppliedmigrations(context.Background())
		if err == nil {
			t.Error("Expected error when table doesn't exist")
		}
		if applied != nil {
			t.Error("Expected nil map when error occurs")
		}
	})

	t.Run("returns empty map when no migrations applied", func(t *testing.T) {
		if err := createTestTable(db, mImpl.tableName); err != nil {
			t.Fatalf("Failed to create test table: %v", err)
		}

		applied, err := mImpl.getAppliedmigrations(context.Background())
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(applied) != 0 {
			t.Errorf("Expected empty map, got %d entries", len(applied))
		}
	})

	t.Run("returns applied migrations", func(t *testing.T) {
		dialect := database.DatabaseType(db)
		sql, params, err := sb.NewBuilder(dialect).
			Table(mImpl.tableName).
			Insert(map[string]string{
				ColumnID:          "2026_03_21_0001_test",
				ColumnBatch:       "20260321120000",
				ColumnDescription: "Test migration",
				ColumnStartedAt:   carbon.Now(carbon.UTC).ToDateTimeString(),
				ColumnCompletedAt: carbon.Now(carbon.UTC).ToDateTimeString(),
			})

		if err != nil {
			t.Fatalf("Failed to build insert SQL: %v", err)
		}

		_, err = db.Exec(sql, params...)
		if err != nil {
			t.Fatalf("Failed to insert test migration: %v", err)
		}

		applied, err := mImpl.getAppliedmigrations(context.Background())
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(applied) != 1 {
			t.Errorf("Expected 1 migration, got %d", len(applied))
		}

		if !applied["2026_03_21_0001_test"] {
			t.Error("Expected migration 2026_03_21_001_test to be applied")
		}
	})
}

func TestRunMigration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if err := createTestTable(db, DefaultTableName); err != nil {
		t.Fatalf("Failed to create migrations table: %v", err)
	}

	m, err := New(db, &Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	mImpl := m.(*migratorImplementation)

	t.Run("runs up migration successfully", func(t *testing.T) {
		executed := false
		migration := &migration{
			ID:          "2026_03_21_0001_test",
			Description: "Test migration",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				executed = true
				_, err := tx.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY)")
				return err
			},
			Down: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.Exec("DROP TABLE test_table")
				return err
			},
		}

		if err := mImpl.runmigration(context.Background(), migration, DirectionUp); err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !executed {
			t.Error("Expected migration Up to be executed")
		}

		var count int
		dialect := database.DatabaseType(db)
		countSQL, countParams, err := sb.NewBuilder(dialect).
			Table(mImpl.tableName).
			Where(&sb.Where{
				Column:   ColumnID,
				Operator: "=",
				Value:    migration.ID,
			}).
			Select([]string{"COUNT(*)"})
		if err != nil {
			t.Fatalf("Failed to build count SQL: %v", err)
		}
		if err := db.QueryRow(countSQL, countParams...).Scan(&count); err != nil {
			t.Fatalf("Failed to query migrations table: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1 migration record, got %d", count)
		}
	})

	t.Run("runs down migration successfully", func(t *testing.T) {
		executed := false
		migration := &migration{
			ID:          "2026_03_21_0001_test",
			Description: "Test migration",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				return nil
			},
			Down: func(ctx context.Context, tx *sql.Tx) error {
				executed = true
				_, err := tx.Exec("DROP TABLE test_table")
				return err
			},
		}

		if err := mImpl.runmigration(context.Background(), migration, DirectionDown); err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !executed {
			t.Error("Expected migration Down to be executed")
		}

		var count int
		dialect := database.DatabaseType(db)
		countSQL, countParams, err := sb.NewBuilder(dialect).
			Table(mImpl.tableName).
			Where(&sb.Where{
				Column:   ColumnID,
				Operator: "=",
				Value:    migration.ID,
			}).
			Select([]string{"COUNT(*)"})
		if err != nil {
			t.Fatalf("Failed to build count SQL: %v", err)
		}
		if err := db.QueryRow(countSQL, countParams...).Scan(&count); err != nil {
			t.Fatalf("Failed to query migrations table: %v", err)
		}
		if count != 0 {
			t.Errorf("Expected 0 migration records, got %d", count)
		}
	})

	t.Run("rolls back on migration error", func(t *testing.T) {
		migration := &migration{
			ID:          "2026_03_21_0002_test",
			Description: "Failing migration",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				return errors.New("migration failed")
			},
			Down: func(ctx context.Context, tx *sql.Tx) error { return nil },
		}

		if err := mImpl.runmigration(context.Background(), migration, DirectionUp); err == nil {
			t.Error("Expected error from failing migration")
		}

		var count int
		dialect := database.DatabaseType(db)
		countSQL, countParams, err := sb.NewBuilder(dialect).
			Table(mImpl.tableName).
			Where(&sb.Where{
				Column:   ColumnID,
				Operator: "=",
				Value:    migration.ID,
			}).
			Select([]string{"COUNT(*)"})
		if err != nil {
			t.Fatalf("Failed to build count SQL: %v", err)
		}
		if err := db.QueryRow(countSQL, countParams...).Scan(&count); err != nil {
			t.Fatalf("Failed to query migrations table: %v", err)
		}
		if count != 0 {
			t.Errorf("Expected 0 migration records after rollback, got %d", count)
		}
	})

	t.Run("rolls back on tracking error", func(t *testing.T) {
		db2 := setupTestDB(t)
		defer db2.Close()

		if err := createTestTable(db2, DefaultTableName); err != nil {
			t.Fatalf("Failed to create migrations table: %v", err)
		}

		m2, err := New(db2, &Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
		if err != nil {
			t.Fatalf("Failed to create migrator: %v", err)
		}
		m2Impl := m2.(*migratorImplementation)

		dialect := database.DatabaseType(db2)
		dropSQL, err := sb.NewBuilder(dialect).
			Table(m2Impl.tableName).
			Drop()
		if err != nil {
			t.Fatalf("Failed to build drop SQL: %v", err)
		}
		if _, err := db2.Exec(dropSQL); err != nil {
			t.Fatalf("Failed to drop table: %v", err)
		}

		migration := &migration{
			ID:          "2026_03_21_0003_test",
			Description: "Test migration",
			Up: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.Exec("CREATE TABLE another_test (id INTEGER)")
				return err
			},
			Down: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.Exec("DROP TABLE another_test")
				return err
			},
		}

		if err := m2Impl.runmigration(context.Background(), migration, DirectionUp); err == nil {
			t.Error("Expected error when tracking table doesn't exist")
		}

		var count int
		if err := db2.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='another_test'").Scan(&count); err != nil {
			t.Fatalf("Failed to query sqlite_master: %v", err)
		}
		if count != 0 {
			t.Error("Expected table to be rolled back")
		}
	})
}
