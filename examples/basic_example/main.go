package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/dracory/migrate"
	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	migrator := migrate.New(db, nil)

	if err := migrator.AddMigration(&CreateUsersTable{}); err != nil {
		log.Fatal(err)
	}

	if err := migrator.Up(context.Background()); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Migrations completed successfully")
}
