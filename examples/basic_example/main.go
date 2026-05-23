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

	migrator, err := migrate.New(db, nil)
	if err != nil {
		log.Fatal(err)
	}

	if err := migrator.AddMigration(&CreateUsersTable{}); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := migrator.Up(ctx); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Migrations completed successfully")
}
