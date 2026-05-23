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

// migration represents a single database migration (internal use only)
type migration struct {
	ID          string
	Description string
	Up          func(context.Context, *sql.Tx) error
	Down        func(context.Context, *sql.Tx) error
}

// migratorImplementation handles database migrations
type migratorImplementation struct {
	db           *sql.DB
	migrations   []*migration
	tableName    string
	logger       *slog.Logger
	namingFormat NamingFormat
	mu           sync.Mutex
}

// AddMigration adds a new migration to the list
//
// Business Logic:
// - Acquires mutex lock to ensure thread-safe operation
// - Delegates to addMigrationInternal for validation and addition
// - Returns any validation errors (duplicate ID, invalid format, etc.)
func (m *migratorImplementation) AddMigration(mig MigrationInterface) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.addMigrationInternal(mig)
}

// addMigrationInternal adds a migration without locking (caller must hold mutex)
//
// Business Logic:
// - Validates migration is not nil
// - Validates migration ID is not empty
// - Validates migration ID format matches configured naming format
// - Checks for duplicate migration IDs to prevent conflicts
// - Converts MigrationInterface to internal migration struct
// - Appends to migrations list
func (m *migratorImplementation) addMigrationInternal(mig MigrationInterface) error {
	if mig == nil {
		return fmt.Errorf("migration cannot be nil")
	}

	if mig.ID() == "" {
		return fmt.Errorf("migration ID cannot be empty")
	}

	// Validate migration ID format based on naming format option
	if err := ValidateMigrationID(mig.ID(), m.namingFormat); err != nil {
		return fmt.Errorf("invalid migration ID: %w", err)
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
//
// Business Logic:
// - Acquires mutex lock to ensure thread-safe operation
// - Iterates through all provided migrations
// - Adds each migration using addMigrationInternal
// - Stops and returns error on first validation failure
// - All migrations are added atomically (all or nothing)
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
//
// Business Logic:
// - Detects database dialect from connection
// - Queries migration tracker table for all migration IDs
// - Orders by completion time (ascending) for consistency
// - Converts result slice to map[string]bool for O(1) lookups
// - Returns error if table doesn't exist or query fails
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

	// Convert slice of maps to map[string]bool for existence checks
	applied := make(map[string]bool, len(appliedStrings))
	for _, row := range appliedStrings {
		if id, ok := row[ColumnID]; ok {
			applied[id] = true
		}
	}

	return applied, nil
}

// runmigration executes a migration up or down
//
// Business Logic:
// - Begins transaction with context for cancellation support
// - Defers rollback in case of failure (ignores ErrTxDone if already committed)
// - Executes migration.Up() or migration.Down() based on direction
// - For Up direction: inserts record into tracker table with batch ID and timestamps
// - For Down direction: deletes record from tracker table
// - Commits transaction if all operations succeed
// - Logs rollback errors if they occur
func (m *migratorImplementation) runmigration(ctx context.Context, migration *migration, direction string) (err error) {
	// Start transaction with context for cancellation
	tx, txErr := m.db.BeginTx(ctx, nil)
	if txErr != nil {
		return txErr
	}
	defer func() {
		if rbErr := tx.Rollback(); rbErr != nil && rbErr != sql.ErrTxDone {
			// If there's already an error, wrap it with rollback error
			if err != nil {
				err = fmt.Errorf("migration error: %w, rollback error: %v", err, rbErr)
			} else {
				err = fmt.Errorf("failed to rollback transaction: %w", rbErr)
			}
			if m.logger != nil {
				m.logger.Error("Failed to rollback transaction", "error", rbErr)
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
		// Generate batch ID (YYYYMMDDHHMMSS) and timestamps after migration completes
		batchID := carbon.Now(carbon.UTC).Format("YmdHis")
		startedAt := carbon.Now(carbon.UTC).ToDateTimeString()
		completedAt := carbon.Now(carbon.UTC).ToDateTimeString()

		sql, params, err := sb.NewBuilder(dialect).
			Table(m.tableName).
			Insert(map[string]string{
				ColumnID:          migration.ID,
				ColumnBatch:       batchID,
				ColumnDescription: migration.Description,
				ColumnStartedAt:   startedAt,
				ColumnCompletedAt: completedAt,
			})

		if err != nil {
			return err
		}

		if _, execErr := tx.Exec(sql, params...); execErr != nil {
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
//
// Business Logic:
// - Gets the builtin migration ID for current naming format
// - Iterates through all registered migrations
// - Returns true if builtin migration ID is found
// - Returns false otherwise
func (m *migratorImplementation) hasBuiltinMigrations() bool {
	builtinID := GetBuiltinMigrationID(m.namingFormat)
	for _, migration := range m.migrations {
		if migration.ID == builtinID {
			return true
		}
	}
	return false
}

// sortMigrations sorts migrations by ID
// Note: Caller must hold m.mu lock
//
// Business Logic:
// - Sorts migrations lexicographically by ID
// - Ensures migrations are executed in chronological order
// - Uses string comparison for ordering
func (m *migratorImplementation) sortMigrations() {
	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].ID < m.migrations[j].ID
	})
}

// ensureBuiltinMigrations adds builtin migrations if not already present
// Note: Caller must hold m.mu lock
//
// Business Logic:
// - Checks if builtin migrations are already registered
// - If not present, retrieves all builtin migrations
// - Adds each builtin migration using addMigrationInternal
// - Returns error if any builtin migration fails to add
// - Ensures migration tracker table creation is always available
func (m *migratorImplementation) ensureBuiltinMigrations() error {
	if !m.hasBuiltinMigrations() {
		builtinMigrations := GetBuiltinMigrations(m.tableName, m.namingFormat)
		for _, mig := range builtinMigrations {
			if err := m.addMigrationInternal(mig); err != nil {
				return fmt.Errorf("failed to add builtin migrations: %w", err)
			}
		}
	}
	return nil
}

// Up runs all pending migrations
//
// Business Logic:
// - Acquires mutex lock for thread-safe operation
// - Ensures builtin migrations are registered
// - Retrieves list of already-applied migrations from database
// - If tracker table doesn't exist, assumes no migrations applied
// - Sorts all migrations by ID for lexicographical ordering
// - Iterates through migrations, running only those not yet applied
// - Logs migration start and completion if logger is configured
// - Stops and returns error on first migration failure
func (m *migratorImplementation) Up(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.ensureBuiltinMigrations(); err != nil {
		return err
	}

	// Get applied migrations (may be empty if migration_tracker doesn't exist yet)
	applied, err := m.getAppliedmigrations(ctx)
	if err != nil {
		// If table doesn't exist, assume no migrations applied
		// The builtin migration will create it
		applied = make(map[string]bool)
	}

	// Sort migrations by ID
	m.sortMigrations()

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
//
// Business Logic:
// - Acquires mutex lock for thread-safe operation
// - Ensures builtin migrations are registered
// - Retrieves list of applied migrations from database
// - Finds the last applied migration (most recent by ID)
// - If no migrations applied, logs and returns successfully
// - Logs rollback start if logger is configured
// - Executes migration Down() method
// - Logs rollback completion if logger is configured
// - Returns error if rollback fails
func (m *migratorImplementation) Down(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.ensureBuiltinMigrations(); err != nil {
		return err
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
//
// Business Logic:
// - Acquires mutex lock for thread-safe operation
// - Retrieves migration status from getStatusInternal
// - Prints formatted status table to stdout
// - Shows ID, description, and state (APPLIED/PENDING) for each migration
// - Returns error if status retrieval fails
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
//
// Business Logic:
// - Retrieves list of applied migrations from database
// - Sorts all migrations by ID for consistent ordering
// - Builds MigrationStatus slice for all registered migrations
// - Sets Applied flag based on existence in applied map
// - Returns error if applied migrations retrieval fails
func (m *migratorImplementation) getStatusInternal(ctx context.Context) ([]MigrationStatus, error) {
	applied, err := m.getAppliedmigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	m.sortMigrations()

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
//
// Business Logic:
// - Acquires mutex lock for thread-safe operation
// - Delegates to getStatusInternal for status retrieval
// - Returns structured data instead of printing to stdout
// - Useful for programmatic access to migration status
func (m *migratorImplementation) GetStatus(ctx context.Context) ([]MigrationStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.getStatusInternal(ctx)
}

// GetHistory returns the migration execution history from the database
//
// Business Logic:
// - Detects database dialect from connection
// - Queries migration tracker table for all columns
// - Orders by completion time (ascending) for chronological order
// - Converts result rows to MigrationRecord structs
// - Returns complete execution history including timestamps
// - Returns error if query fails
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
