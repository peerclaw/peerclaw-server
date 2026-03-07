package userauth

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

// NewSQLiteStore creates a new SQLite-backed user auth store.
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

// Migrate creates the required tables.
func (s *SQLiteStore) Migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id            TEXT PRIMARY KEY,
			email         TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			display_name  TEXT NOT NULL DEFAULT '',
			role          TEXT NOT NULL DEFAULT 'user',
			created_at    DATETIME NOT NULL,
			updated_at    DATETIME NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id            TEXT PRIMARY KEY,
			user_id       TEXT NOT NULL,
			refresh_token TEXT NOT NULL,
			ip_address    TEXT DEFAULT '',
			user_agent    TEXT DEFAULT '',
			expires_at    DATETIME NOT NULL,
			created_at    DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(refresh_token)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id)`,
		`CREATE TABLE IF NOT EXISTS user_api_keys (
			id         TEXT PRIMARY KEY,
			user_id    TEXT NOT NULL,
			name       TEXT NOT NULL,
			key_hash   TEXT NOT NULL UNIQUE,
			prefix     TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			last_used  DATETIME,
			expires_at DATETIME,
			revoked    BOOLEAN DEFAULT 0,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON user_api_keys(key_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_user ON user_api_keys(user_id)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("userauth migrate: %w", err)
		}
	}
	return nil
}

func (s *SQLiteStore) CreateUser(ctx context.Context, user *User) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, email, password_hash, display_name, role, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Email, user.PasswordHash, user.DisplayName, user.Role,
		user.CreatedAt.UTC().Format(time.RFC3339),
		user.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) GetUserByID(ctx context.Context, id string) (*User, error) {
	var u User
	var createdAt, updatedAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, display_name, role, created_at, updated_at
		 FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Role, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	u.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &u, nil
}

func (s *SQLiteStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	var createdAt, updatedAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, display_name, role, created_at, updated_at
		 FROM users WHERE email = ?`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Role, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	u.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &u, nil
}

func (s *SQLiteStore) UpdateUser(ctx context.Context, user *User) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET display_name = ?, role = ?, updated_at = ? WHERE id = ?`,
		user.DisplayName, user.Role, user.UpdatedAt.UTC().Format(time.RFC3339), user.ID,
	)
	return err
}

func (s *SQLiteStore) CreateSession(ctx context.Context, session *Session) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, refresh_token, ip_address, user_agent, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		session.ID, session.UserID, session.RefreshToken,
		session.IPAddress, session.UserAgent,
		session.ExpiresAt.UTC().Format(time.RFC3339),
		session.CreatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) GetSessionByToken(ctx context.Context, tokenHash string) (*Session, error) {
	var sess Session
	var expiresAt, createdAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, refresh_token, ip_address, user_agent, expires_at, created_at
		 FROM sessions WHERE refresh_token = ?`, tokenHash,
	).Scan(&sess.ID, &sess.UserID, &sess.RefreshToken, &sess.IPAddress, &sess.UserAgent, &expiresAt, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, err
	}
	sess.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	sess.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &sess, nil
}

func (s *SQLiteStore) DeleteSession(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE id = ?", id)
	return err
}

func (s *SQLiteStore) DeleteExpiredSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM sessions WHERE expires_at < ?",
		time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) CreateAPIKey(ctx context.Context, key *UserAPIKey) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO user_api_keys (id, user_id, name, key_hash, prefix, created_at, last_used, expires_at, revoked)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		key.ID, key.UserID, key.Name, key.KeyHash, key.Prefix,
		key.CreatedAt.UTC().Format(time.RFC3339),
		nilTimeStr(key.LastUsed), nilTimeStr(key.ExpiresAt), key.Revoked,
	)
	return err
}

func (s *SQLiteStore) ListAPIKeys(ctx context.Context, userID string) ([]UserAPIKey, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, name, prefix, created_at, last_used, expires_at, revoked
		 FROM user_api_keys WHERE user_id = ? ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var keys []UserAPIKey
	for rows.Next() {
		var k UserAPIKey
		var createdAt string
		var lastUsed, expiresAt sql.NullString
		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.Prefix, &createdAt, &lastUsed, &expiresAt, &k.Revoked); err != nil {
			return nil, err
		}
		k.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		if lastUsed.Valid {
			t, _ := time.Parse(time.RFC3339, lastUsed.String)
			k.LastUsed = &t
		}
		if expiresAt.Valid {
			t, _ := time.Parse(time.RFC3339, expiresAt.String)
			k.ExpiresAt = &t
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (s *SQLiteStore) GetAPIKeyByHash(ctx context.Context, keyHash string) (*UserAPIKey, error) {
	var k UserAPIKey
	var createdAt string
	var lastUsed, expiresAt sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, name, key_hash, prefix, created_at, last_used, expires_at, revoked
		 FROM user_api_keys WHERE key_hash = ?`, keyHash,
	).Scan(&k.ID, &k.UserID, &k.Name, &k.KeyHash, &k.Prefix, &createdAt, &lastUsed, &expiresAt, &k.Revoked)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("API key not found")
		}
		return nil, err
	}
	k.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if lastUsed.Valid {
		t, _ := time.Parse(time.RFC3339, lastUsed.String)
		k.LastUsed = &t
	}
	if expiresAt.Valid {
		t, _ := time.Parse(time.RFC3339, expiresAt.String)
		k.ExpiresAt = &t
	}
	return &k, nil
}

func (s *SQLiteStore) RevokeAPIKey(ctx context.Context, keyID, userID string) error {
	res, err := s.db.ExecContext(ctx,
		"UPDATE user_api_keys SET revoked = 1 WHERE id = ? AND user_id = ?", keyID, userID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("API key not found")
	}
	return nil
}

func (s *SQLiteStore) UpdateAPIKeyLastUsed(ctx context.Context, keyID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE user_api_keys SET last_used = ? WHERE id = ?",
		time.Now().UTC().Format(time.RFC3339), keyID,
	)
	return err
}

func (s *SQLiteStore) ListUsers(ctx context.Context, search, role string, limit, offset int) ([]User, int, error) {
	if limit <= 0 {
		limit = 50
	}

	where := "1=1"
	var args []interface{}
	if search != "" {
		where += " AND (email LIKE ? OR display_name LIKE ?)"
		args = append(args, "%"+search+"%", "%"+search+"%")
	}
	if role != "" {
		where += " AND role = ?"
		args = append(args, role)
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE "+where, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, email, password_hash, display_name, role, created_at, updated_at FROM users WHERE "+where+" ORDER BY created_at DESC LIMIT ? OFFSET ?",
		args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	var users []User
	for rows.Next() {
		var u User
		var createdAt, updatedAt string
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Role, &createdAt, &updatedAt); err != nil {
			return nil, 0, err
		}
		u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		u.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (s *SQLiteStore) DeleteUser(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (s *SQLiteStore) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

// Close is a no-op since the db is shared.
func (s *SQLiteStore) Close() error {
	return nil
}

func nilTimeStr(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}
