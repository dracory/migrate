package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
	"tap.com/pkg/migrate"
)

func main() {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Example with custom options
	migrator := migrate.New(db, &migrate.Options{
		MigrationTableName: "custom_migrations", // Custom table name
		Logger:             nil,                 // Disable logging
	})

	fmt.Printf("Migrator created: %T\n", migrator)
	fmt.Printf("Using custom table name: custom_migrations\n")

	// Note: The builtin migration now automatically creates the custom tracking table
	// with the new schema including batch, started_at, and completed_at columns
	fmt.Println("Builtin migration will create custom tracking table automatically")
	fmt.Println("This example demonstrates custom table name configuration")

	if err := migrator.AddMigration(&CreatePostsTable{}); err != nil {
		log.Fatal(err)
	}

	if err := migrator.Up(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Custom migration completed successfully")

	// Demonstrate rollback
	if err := migrator.Down(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Custom migration rolled back successfully")
}
