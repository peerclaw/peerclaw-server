package contacts

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SQLiteStore implements Store using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite-backed contacts store.
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) Migrate(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS agent_contacts (
		id               TEXT PRIMARY KEY,
		owner_agent_id   TEXT NOT NULL,
		contact_agent_id TEXT NOT NULL,
		alias            TEXT DEFAULT '',
		created_at       DATETIME NOT NULL,
		UNIQUE(owner_agent_id, contact_agent_id)
	);
	CREATE INDEX IF NOT EXISTS idx_contacts_owner ON agent_contacts(owner_agent_id);
	`
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

func (s *SQLiteStore) Add(ctx context.Context, contact *Contact) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agent_contacts (id, owner_agent_id, contact_agent_id, alias, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		contact.ID, contact.OwnerAgentID, contact.ContactAgentID, contact.Alias,
		contact.CreatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) Remove(ctx context.Context, ownerAgentID, contactAgentID string) error {
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM agent_contacts WHERE owner_agent_id = ? AND contact_agent_id = ?`,
		ownerAgentID, contactAgentID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("contact not found")
	}
	return nil
}

func (s *SQLiteStore) IsAllowed(ctx context.Context, ownerAgentID, contactAgentID string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM agent_contacts
		WHERE owner_agent_id = ? AND contact_agent_id = ?`,
		ownerAgentID, contactAgentID,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *SQLiteStore) ListByOwner(ctx context.Context, ownerAgentID string) ([]Contact, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, owner_agent_id, contact_agent_id, COALESCE(alias, ''), created_at
		FROM agent_contacts WHERE owner_agent_id = ? ORDER BY created_at DESC`, ownerAgentID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var contacts []Contact
	for rows.Next() {
		var c Contact
		var createdAt string
		if err := rows.Scan(&c.ID, &c.OwnerAgentID, &c.ContactAgentID, &c.Alias, &createdAt); err != nil {
			return nil, err
		}
		if parsed, err := time.Parse(time.RFC3339, createdAt); err == nil {
			c.CreatedAt = parsed
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}

func (s *SQLiteStore) Close() error {
	return nil // shared db, don't close
}
