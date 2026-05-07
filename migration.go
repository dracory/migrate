package migrate

import (
	"context"
	"database/sql"
)

// migration represents a single database migration (internal use only)
type migration struct {
	ID          string
	Description string
	Up          func(context.Context, *sql.Tx) error
	Down        func(context.Context, *sql.Tx) error
}
