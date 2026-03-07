package verification

import "database/sql"

// NewStore creates a Store based on the driver name.
// Supported drivers: "sqlite" (default), "postgres".
func NewStore(driver string, db *sql.DB) Store {
	switch driver {
	case "postgres":
		return NewPostgresStore(db)
	default:
		return NewSQLiteStore(db)
	}
}
