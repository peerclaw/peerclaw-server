package invocation

import "database/sql"

// NewStore creates a Store based on the driver name.
func NewStore(driver string, db *sql.DB) Store {
	switch driver {
	case "postgres":
		return NewPostgresStore(db)
	default:
		return NewSQLiteStore(db)
	}
}
