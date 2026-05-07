// Package migrate provides a database migration framework with support for
// versioned migrations, rollbacks, and migration tracking.
//
// The package is designed to be database-agnostic and can be used as a
// standalone library for managing database schema changes.
//
// Key features:
//   - Lexicographical migration ordering by ID
//   - Transactional migration execution
//   - Migration rollback support
//   - Customizable logging
//   - Duplicate migration detection
//
// Basic Usage:
//
//	db, err := sql.Open("sqlite", "database.db")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer db.Close()
//
//	migrator := migrate.New(db, nil)
//
//	if err := migrator.AddMigration(&YourMigration{}); err != nil {
//	    log.Fatal(err)
//	}
//
//	if err := migrator.Up(context.Background()); err != nil {
//	    log.Fatal(err)
//	}
//
// The Up() and Down() methods accept a context.Context parameter for cancellation support.
// Use context.Background() for non-cancelable operations, or a context with timeout
// for long-running migrations.
//
// Migration ID Format:
//
// Migrations must have an ID in the format: YYYY_MM_DD_description
// For example: 2026_03_21_create_users_table
//
// The date part must be a valid calendar date (e.g., February 30 will be rejected).
package migrate
