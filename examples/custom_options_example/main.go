package custom_options_example

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/dracory/migrate"
	_ "modernc.org/sqlite"
)

func RunMigrations(db *sql.DB) error {
	migrator, err := migrate.New(db, &migrate.Options{
		MigrationTableName: "custom_migrations",
		Logger:             nil,
	})
	if err != nil {
		return err
	}

	if err := migrator.AddMigration(&CreatePostsTable{}); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return migrator.Up(ctx)
}

func RollbackMigrations(db *sql.DB) error {
	migrator, err := migrate.New(db, &migrate.Options{
		MigrationTableName: "custom_migrations",
		Logger:             nil,
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return migrator.Down(ctx)
}

func main() {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Printf("Migrator created with custom table name: custom_migrations\n")
	fmt.Println("Builtin migration will create custom tracking table automatically")
	fmt.Println("This example demonstrates custom table name configuration")

	if err := RunMigrations(db); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Custom migration completed successfully")

	if err := RollbackMigrations(db); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Custom migration rolled back successfully")
}
