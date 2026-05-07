# Migration Package

A lightweight, database-agnostic migration framework for Go applications.

## Features

- **Lexicographical ordering**: Migrations are sorted and applied by ID
- **Transactional execution**: Each migration runs in a transaction
- **Rollback support**: Roll back the last applied migration
- **Customizable logging**: Inject your own logger or disable logging
- **Duplicate detection**: Prevents adding migrations with duplicate IDs
- **Automatic builtin migrations**: Creates schema tracking table automatically
- **Performance tracking**: Records start/end times for each migration
- **Batch grouping**: Groups migrations by execution batch
- **Custom table names**: Support for custom migration table names

## Installation

```go
import "github.com/dracory/migrate"
```

## Quick Start

### 1. Create a Migration

Implement the `MigrationInterface`:

```go
package migrations

import (
    "database/sql"
    
    "github.com/dracory/database"
    "github.com/dracory/sb"
)

type CreateUsersTable struct{}

func (m *CreateUsersTable) ID() string {
    return "2026_03_21_001_create_users_table"
}

func (m *CreateUsersTable) Description() string {
    return "Create users table with email and timestamps"
}

func (m *CreateUsersTable) Up(tx *sql.Tx) error {
    dialect := database.DatabaseType(tx)
    sql, err := sb.NewBuilder(dialect).
        Table("users").
        Column(sb.Column{
            Name:       "id",
            Type:       sb.COLUMN_TYPE_INTEGER,
            PrimaryKey: true,
        }).
        Column(sb.Column{
            Name:     "email",
            Type:     sb.COLUMN_TYPE_STRING,
            Length:   255,
            Nullable: false,
        }).
        Column(sb.Column{
            Name:     "created_at",
            Type:     sb.COLUMN_TYPE_DATETIME,
            Nullable: false,
        }).
        Create()
    if err != nil {
        return err
    }
    _, err = tx.Exec(sql)
    return err
}

func (m *CreateUsersTable) Down(tx *sql.Tx) error {
    dialect := database.DatabaseType(tx)
    sql, err := sb.NewBuilder(dialect).
        Table("users").
        Drop()
    if err != nil {
        return err
    }
    _, err = tx.Exec(sql)
    return err
}
```

### 2. Run Migrations

```go
package main

import (
    "context"
    "database/sql"
    "log"

    "github.com/dracory/migrate"
    "github.com/yourusername/yourproject/internal/migrations"
    _ "github.com/mattn/go-sqlite3"
)

func main() {
    db, err := sql.Open("sqlite", "app.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Create migrator with options
    migrator := migrate.New(db, nil)
    
    // Add your migrations (builtin migrations are added automatically)
    if err := migrator.AddMigration(&migrations.CreateUsersTable{}); err != nil {
        log.Fatal(err)
    }
    
    // Run all pending migrations
    if err := migrator.Up(context.Background()); err != nil {
        log.Fatal(err)
    }
    
    // Roll back the last migration
    if err := migrator.Down(context.Background()); err != nil {
        log.Fatal(err)
    }
    
    log.Println("Migrations completed successfully")
}
```

## Schema

The migration tracking table is automatically created with the following schema:

```sql
CREATE TABLE schema_migrations (
    id TEXT PRIMARY KEY,                    -- The immutable identifier for the change
    batch INTEGER NOT NULL,                 -- Timestamp ID (YYYYMMDDHHMMSS). Groups the run
    description TEXT NOT NULL,              -- What this specific change did
    started_at DATETIME NOT NULL,           -- When the SQL started
    completed_at DATETIME NOT NULL          -- When the SQL finished
);
```

### Automatic Builtin Migrations

The package includes a builtin migration that automatically creates the tracking table. This migration:

- Runs automatically when you call `Up()` or `Down()` for the first time
- Respects your custom table name configuration
- Uses the new schema with performance tracking columns
- Is always the first migration to run (ID: `2022_01_01_000_create_schema_migrations`)

## Migration ID Format

Migration IDs should follow this format for proper ordering:

```
YYYY_MM_DD_NNN_description
```

Examples:
- `2026_03_21_001_create_users_table`
- `2026_03_21_002_add_users_email_index`
- `2026_03_22_001_create_posts_table`

**Important**: Never change a migration ID after it has been applied to any environment.

## Configuration Options

### Custom Table Name

```go
migrator := migrate.New(db, &migrate.Options{
    MigrationTableName: "custom_migrations",
})
```

### Custom Logger

```go
import "log/slog"

migrator := migrate.New(db, &migrate.Options{
    Logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
})
```

### Disable Logging

```go
migrator := migrate.New(db, &migrate.Options{
    Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
})
```

### Default Options

```go
migrator := migrate.New(db, nil)  // Uses default table name and slog.Default()
```

## API Reference

### Migrator Methods

#### `Up() error`
Runs all pending migrations in lexicographical order by ID.

#### `Down() error`
Rolls back the last applied migration.

#### `Status() error`
Displays the status of all migrations (APPLIED or PENDING).

#### `AddMigration(migration MigrationInterface) error`
Adds a single migration to the migrator.

#### `AddMigrations(migrations []MigrationInterface) error`
Adds multiple migrations to the migrator.


### MigrationInterface

All migrations must implement this interface:

```go
type MigrationInterface interface {
    ID() string
    Description() string
    Up(tx *sql.Tx) error
    Down(tx *sql.Tx) error
}
```

## Best Practices

1. **Use descriptive IDs**: Include the date and a clear description
2. **Keep migrations small**: One logical change per migration
3. **Test rollbacks**: Ensure your `Down()` method properly reverses `Up()`
4. **Never modify applied migrations**: Create new migrations for changes
5. **Use transactions**: All migrations run in transactions automatically
6. **Handle errors**: Return errors from `Up()` and `Down()` methods
7. **Custom table names**: Use `MigrationTableName` option for custom tracking table names
8. **Performance monitoring**: Check `started_at` and `completed_at` columns for migration timing

## Migration Execution Order

Migrations are sorted lexicographically by their ID before execution:

```
2022_01_01_000_create_schema_migrations  # Builtin (always first)
2026_03_21_001_create_users_table        # User migration
2026_03_21_002_add_users_email_index     # User migration
2026_03_22_001_create_posts_table        # User migration
```

## Error Handling

If a migration fails:
- The transaction is automatically rolled back
- The migration is NOT recorded in `schema_migrations`
- The error is returned to the caller
- Subsequent migrations are NOT executed

## Testing

The package includes comprehensive tests. Run them with:

```bash
go test ./pkg/migrate/... -v
```

## Standalone Usage

This package is designed to be extracted as a standalone library. It has minimal dependencies and can be used in any Go project.

## License

[Your License Here]
