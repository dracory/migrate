package basic_example

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
	migrator, err := migrate.New(db, nil)
	if err != nil {
		return err
	}

	if err := migrator.AddMigration(&CreateUsersTable{}); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return migrator.Up(ctx)
}

func main() {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := RunMigrations(db); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Migrations completed successfully")
}
