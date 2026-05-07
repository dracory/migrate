package migrate

import "database/sql"

// migration represents a single database migration (internal use only)
type migration struct {
	ID          string
	Description string
	Up          func(*sql.Tx) error
	Down        func(*sql.Tx) error
}
