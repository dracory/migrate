package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/dracory/migrate"
	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Example with custom options
	migrator, err := migrate.New(db, &migrate.Options{
		MigrationTableName: "custom_migrations", // Custom table name
		Logger:             nil,                 // Disable logging
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Migrator created: %T\n", migrator)
	fmt.Printf("Using custom table name: custom_migrations\n")

	// Note: The builtin migration now automatically creates the custom tracking table
	// with the new schema including batch, started_at, and completed_at columns
	fmt.Println("Builtin migration will create custom tracking table automatically")
	fmt.Println("This example demonstrates custom table name configuration")

	if err := migrator.AddMigration(&CreatePostsTable{}); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := migrator.Up(ctx); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Custom migration completed successfully")

	// Demonstrate rollback
	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel2()

	if err := migrator.Down(ctx2); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Custom migration rolled back successfully")
}
