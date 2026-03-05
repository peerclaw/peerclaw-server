package registry

import "fmt"

// NewStore creates a Store based on the driver name.
// Supported drivers: "sqlite" (default), "postgres".
func NewStore(driver, dsn string) (Store, error) {
	switch driver {
	case "", "sqlite":
		return NewSQLiteStore(dsn)
	case "postgres":
		return NewPostgresStore(dsn)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}
}
