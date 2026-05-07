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
// Basic usage:
//
//	migrator := migrate.New(db, "postgres")
//	migrator.AddMigration(&MyMigration{})
//	if err := migrator.Up(); err != nil {
//	    log.Fatal(err)
//	}
package migrate
