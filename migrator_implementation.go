package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sort"
	"sync"

	"github.com/dracory/database"
	"github.com/dracory/sb"
	carbon "github.com/dromara/carbon/v2"
)

// migratorImplementation handles database migrations
type migratorImplementation struct {
	db         *sql.DB
	migrations []*migration
	tableName  string
	logger     *slog.Logger
	mu         sync.Mutex
}

// AddMigration adds a new migration to the list
func (m *migratorImplementation) AddMigration(mig MigrationInterface) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.addMigrationInternal(mig)
}

// addMigrationInternal adds a migration without locking (caller must hold mutex)
func (m *migratorImplementation) addMigrationInternal(mig MigrationInterface) error {
	if mig == nil {
		return fmt.Errorf("migration cannot be nil")
	}

	if mig.ID() == "" {
		return fmt.Errorf("migration ID cannot be empty")
	}

	// Validate migration ID format: YYYY_MM_DD_description
	if !isValidMigrationID(mig.ID()) {
		return fmt.Errorf("migration ID must follow format YYYY_MM_DD_description, got: %s", mig.ID())
	}

	// Check for duplicate IDs
	for _, existing := range m.migrations {
		if existing.ID == mig.ID() {
			return fmt.Errorf("migration with ID %s already exists", mig.ID())
		}
	}

	// Convert interface to internal struct
	internalMigration := &migration{
		ID:          mig.ID(),
		Description: mig.Description(),
		Up:          mig.Up,
		Down:        mig.Down,
	}
	m.migrations = append(m.migrations, internalMigration)
	return nil
}

// AddMigrations adds multiple migrations to the runner
func (m *migratorImplementation) AddMigrations(migrations []MigrationInterface) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, mig := range migrations {
		if err := m.addMigrationInternal(mig); err != nil {
			return err
		}
	}
	return nil
}

// getAppliedmigrations returns a map of applied migration IDs
func (m *migratorImplementation) getAppliedmigrations(ctx context.Context) (map[string]bool, error) {
	dialect := database.DatabaseType(m.db)
	sql, params, err := sb.NewBuilder(dialect).
		Table(m.tableName).
		OrderBy(ColumnCompletedAt, "asc").
		Select([]string{ColumnID})

	if err != nil {
		return nil, err
	}

	appliedStrings, err := database.SelectToMapString(database.NewQueryableContext(ctx, m.db), sql, params...)
	if err != nil {
		return nil, err
	}

	// Convert map[string]string to map[string]bool
	applied := make(map[string]bool)
	for _, migrationMap := range appliedStrings {
		if id, ok := migrationMap[ColumnID]; ok {
			applied[id] = true
		}
	}

	return applied, nil
}

// runmigration executes a migration up or down
func (m *migratorImplementation) runmigration(ctx context.Context, migration *migration, direction string) error {
	// Start transaction with context for cancellation
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil {
			if m.logger != nil {
				m.logger.Warn("Failed to rollback transaction", "error", err)
			}
		}
	}()

	// Run the migration
	var migrationErr error
	if direction == DirectionUp {
		migrationErr = migration.Up(ctx, tx)
	} else {
		migrationErr = migration.Down(ctx, tx)
	}

	if migrationErr != nil {
		return migrationErr
	}

	// Update migrations table
	if direction == DirectionUp {
		dialect := database.DatabaseType(tx)
		// Generate batch ID (YYYYMMDDHHMMSS)
		batchID := carbon.Now(carbon.UTC).Format("YmdHis")
		startedAt := carbon.Now(carbon.UTC).ToDateTimeString()

		sql, params, err := sb.NewBuilder(dialect).
			Table(m.tableName).
			Insert(map[string]string{
				ColumnID:          migration.ID,
				ColumnBatch:       batchID,
				ColumnDescription: migration.Description,
				ColumnStartedAt:   startedAt,
				ColumnCompletedAt: startedAt, // Will be updated after migration completes
			})

		if err != nil {
			return err
		}

		if _, execErr := tx.Exec(sql, params...); execErr != nil {
			return execErr
		}

		// Update completed_at timestamp after migration completes
		completedAt := carbon.Now(carbon.UTC).ToDateTimeString()
		updateSQL, updateParams, err := sb.NewBuilder(dialect).
			Table(m.tableName).
			Where(&sb.Where{
				Column:   ColumnID,
				Operator: "=",
				Value:    migration.ID,
			}).
			Update(map[string]string{
				ColumnCompletedAt: completedAt,
			})

		if err != nil {
			return err
		}

		if _, execErr := tx.Exec(updateSQL, updateParams...); execErr != nil {
			return execErr
		}
	} else {
		dialect := database.DatabaseType(tx)
		sql, params, err := sb.NewBuilder(dialect).
			Table(m.tableName).
			Where(&sb.Where{
				Column:   ColumnID,
				Operator: "=",
				Value:    migration.ID,
			}).
			Delete()

		if err != nil {
			return err
		}

		if _, execErr := tx.Exec(sql, params...); execErr != nil {
			return execErr
		}
	}

	// Commit transaction
	return tx.Commit()
}

// hasBuiltinMigrations checks if builtin migrations have been added
// Note: Caller must hold m.mu lock
func (m *migratorImplementation) hasBuiltinMigrations() bool {
	for _, migration := range m.migrations {
		if migration.ID == BuiltinMigrationID {
			return true
		}
	}
	return false
}

// Up runs all pending migrations
func (m *migratorImplementation) Up(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Automatically add builtin migrations if not already added
	if !m.hasBuiltinMigrations() {
		builtinMigrations := GetBuiltinMigrations(m.tableName)
		for _, mig := range builtinMigrations {
			if err := m.addMigrationInternal(mig); err != nil {
				return fmt.Errorf("failed to add builtin migrations: %w", err)
			}
		}
	}

	// Get applied migrations (may be empty if schema_migrations doesn't exist yet)
	applied, err := m.getAppliedmigrations(ctx)
	if err != nil {
		// If table doesn't exist, assume no migrations applied
		// The builtin migration will create it
		applied = make(map[string]bool)
	}

	// Sort migrations by ID
	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].ID < m.migrations[j].ID
	})

	// Run pending migrations
	for _, migration := range m.migrations {
		if _, exists := applied[migration.ID]; !exists {
			if m.logger != nil {
				m.logger.Info("Running migration", "id", migration.ID, "description", migration.Description)
			}

			if err := m.runmigration(ctx, migration, DirectionUp); err != nil {
				return fmt.Errorf("migration %s failed: %w", migration.ID, err)
			}

			if m.logger != nil {
				m.logger.Info("migration completed", "id", migration.ID)
			}
		}
	}

	return nil
}

// Down rolls back the last migration
func (m *migratorImplementation) Down(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Automatically add builtin migrations if not already added
	if !m.hasBuiltinMigrations() {
		builtinMigrations := GetBuiltinMigrations(m.tableName)
		for _, mig := range builtinMigrations {
			if err := m.addMigrationInternal(mig); err != nil {
				return fmt.Errorf("failed to add builtin migrations: %w", err)
			}
		}
	}

	// Get applied migrations
	applied, err := m.getAppliedmigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Find the last applied migration
	var lastmigration *migration
	for _, migration := range m.migrations {
		if _, exists := applied[migration.ID]; exists {
			lastmigration = migration
		}
	}

	if lastmigration == nil {
		if m.logger != nil {
			m.logger.Info("No migrations to rollback")
		}
		return nil
	}

	if m.logger != nil {
		m.logger.Info("Rolling back migration", "id", lastmigration.ID, "description", lastmigration.Description)
	}

	if err := m.runmigration(ctx, lastmigration, DirectionDown); err != nil {
		return fmt.Errorf("rollback %s failed: %w", lastmigration.ID, err)
	}

	if m.logger != nil {
		m.logger.Info("Rollback completed", "id", lastmigration.ID)
	}
	return nil
}

// Status shows migration status
func (m *migratorImplementation) Status(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	statuses, err := m.getStatusInternal(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("\nmigration Status:\n")
	fmt.Printf("================\n")

	for _, status := range statuses {
		state := "PENDING"
		if status.Applied {
			state = "APPLIED"
		}
		fmt.Printf("%s - %s - %s\n", status.ID, status.Description, state)
	}

	return nil
}

// getStatusInternal returns migration status without locking (caller must hold mutex)
func (m *migratorImplementation) getStatusInternal(ctx context.Context) ([]MigrationStatus, error) {
	applied, err := m.getAppliedmigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].ID < m.migrations[j].ID
	})

	statuses := make([]MigrationStatus, 0, len(m.migrations))
	for _, migration := range m.migrations {
		_, exists := applied[migration.ID]
		statuses = append(statuses, MigrationStatus{
			ID:          migration.ID,
			Description: migration.Description,
			Applied:     exists,
		})
	}

	return statuses, nil
}

// GetStatus returns migration status as structured data
func (m *migratorImplementation) GetStatus(ctx context.Context) ([]MigrationStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.getStatusInternal(ctx)
}

// GetHistory returns the migration execution history from the database
func (m *migratorImplementation) GetHistory(ctx context.Context) ([]MigrationRecord, error) {
	dialect := database.DatabaseType(m.db)
	sql, params, err := sb.NewBuilder(dialect).
		Table(m.tableName).
		OrderBy(ColumnCompletedAt, "asc").
		Select([]string{ColumnID, ColumnBatch, ColumnDescription, ColumnStartedAt, ColumnCompletedAt})

	if err != nil {
		return nil, err
	}

	rows, err := database.SelectToMapString(database.NewQueryableContext(ctx, m.db), sql, params...)
	if err != nil {
		return nil, err
	}

	records := make([]MigrationRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, MigrationRecord{
			ID:          row[ColumnID],
			Batch:       row[ColumnBatch],
			Description: row[ColumnDescription],
			StartedAt:   row[ColumnStartedAt],
			CompletedAt: row[ColumnCompletedAt],
		})
	}

	return records, nil
}
