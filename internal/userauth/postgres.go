package userauth

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// PostgresStore implements Store using PostgreSQL.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgreSQL-backed user auth store.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// Migrate creates the required tables.
func (s *PostgresStore) Migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id            TEXT PRIMARY KEY,
			email         TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			display_name  TEXT NOT NULL DEFAULT '',
			role          TEXT NOT NULL DEFAULT 'user',
			created_at    TIMESTAMPTZ NOT NULL,
			updated_at    TIMESTAMPTZ NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id            TEXT PRIMARY KEY,
			user_id       TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			refresh_token TEXT NOT NULL,
			ip_address    TEXT DEFAULT '',
			user_agent    TEXT DEFAULT '',
			expires_at    TIMESTAMPTZ NOT NULL,
			created_at    TIMESTAMPTZ NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(refresh_token)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id)`,
		`CREATE TABLE IF NOT EXISTS user_api_keys (
			id         TEXT PRIMARY KEY,
			user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name       TEXT NOT NULL,
			key_hash   TEXT NOT NULL UNIQUE,
			prefix     TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			last_used  TIMESTAMPTZ,
			expires_at TIMESTAMPTZ,
			revoked    BOOLEAN DEFAULT FALSE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON user_api_keys(key_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_user ON user_api_keys(user_id)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("userauth migrate: %w", err)
		}
	}
	// Add description column if it doesn't exist.
	_, _ = s.db.ExecContext(ctx, "ALTER TABLE users ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT ''")
	// Add email_verified column if it doesn't exist.
	_, _ = s.db.ExecContext(ctx, "ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT FALSE")
	// Create email_verifications table.
	evStmts := []string{
		`CREATE TABLE IF NOT EXISTS email_verifications (
			id         TEXT PRIMARY KEY,
			email      TEXT NOT NULL,
			code_hash  TEXT NOT NULL,
			purpose    TEXT NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			used       BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ev_email ON email_verifications(email)`,
	}
	for _, stmt := range evStmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("userauth migrate email_verifications: %w", err)
		}
	}
	return nil
}

func (s *PostgresStore) CreateUser(ctx context.Context, user *User) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, email, password_hash, display_name, description, role, email_verified, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		user.ID, user.Email, user.PasswordHash, user.DisplayName, user.Description, user.Role, user.EmailVerified,
		user.CreatedAt.UTC(), user.UpdatedAt.UTC(),
	)
	return err
}

func (s *PostgresStore) GetUserByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, display_name, description, role, email_verified, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Description, &u.Role, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &u, nil
}

func (s *PostgresStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, display_name, description, role, email_verified, created_at, updated_at
		 FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Description, &u.Role, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &u, nil
}

func (s *PostgresStore) UpdateUser(ctx context.Context, user *User) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET email = $1, password_hash = $2, display_name = $3, description = $4, role = $5, email_verified = $6, updated_at = $7 WHERE id = $8`,
		user.Email, user.PasswordHash, user.DisplayName, user.Description, user.Role, user.EmailVerified, user.UpdatedAt.UTC(), user.ID,
	)
	return err
}

func (s *PostgresStore) CreateSession(ctx context.Context, session *Session) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, refresh_token, ip_address, user_agent, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		session.ID, session.UserID, session.RefreshToken,
		session.IPAddress, session.UserAgent,
		session.ExpiresAt.UTC(), session.CreatedAt.UTC(),
	)
	return err
}

func (s *PostgresStore) GetSessionByToken(ctx context.Context, tokenHash string) (*Session, error) {
	var sess Session
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, refresh_token, ip_address, user_agent, expires_at, created_at
		 FROM sessions WHERE refresh_token = $1`, tokenHash,
	).Scan(&sess.ID, &sess.UserID, &sess.RefreshToken, &sess.IPAddress, &sess.UserAgent, &sess.ExpiresAt, &sess.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, err
	}
	return &sess, nil
}

func (s *PostgresStore) DeleteSession(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE id = $1", id)
	return err
}

func (s *PostgresStore) DeleteExpiredSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at < $1", time.Now().UTC())
	return err
}

func (s *PostgresStore) CreateAPIKey(ctx context.Context, key *UserAPIKey) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO user_api_keys (id, user_id, name, key_hash, prefix, created_at, last_used, expires_at, revoked)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		key.ID, key.UserID, key.Name, key.KeyHash, key.Prefix,
		key.CreatedAt.UTC(), nilTimePtr(key.LastUsed), nilTimePtr(key.ExpiresAt), key.Revoked,
	)
	return err
}

func (s *PostgresStore) ListAPIKeys(ctx context.Context, userID string) ([]UserAPIKey, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, name, prefix, created_at, last_used, expires_at, revoked
		 FROM user_api_keys WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var keys []UserAPIKey
	for rows.Next() {
		var k UserAPIKey
		var lastUsed, expiresAt sql.NullTime
		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.Prefix, &k.CreatedAt, &lastUsed, &expiresAt, &k.Revoked); err != nil {
			return nil, err
		}
		if lastUsed.Valid {
			k.LastUsed = &lastUsed.Time
		}
		if expiresAt.Valid {
			k.ExpiresAt = &expiresAt.Time
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (s *PostgresStore) GetAPIKeyByHash(ctx context.Context, keyHash string) (*UserAPIKey, error) {
	var k UserAPIKey
	var lastUsed, expiresAt sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, name, key_hash, prefix, created_at, last_used, expires_at, revoked
		 FROM user_api_keys WHERE key_hash = $1`, keyHash,
	).Scan(&k.ID, &k.UserID, &k.Name, &k.KeyHash, &k.Prefix, &k.CreatedAt, &lastUsed, &expiresAt, &k.Revoked)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("API key not found")
		}
		return nil, err
	}
	if lastUsed.Valid {
		k.LastUsed = &lastUsed.Time
	}
	if expiresAt.Valid {
		k.ExpiresAt = &expiresAt.Time
	}
	return &k, nil
}

func (s *PostgresStore) RevokeAPIKey(ctx context.Context, keyID, userID string) error {
	res, err := s.db.ExecContext(ctx,
		"UPDATE user_api_keys SET revoked = TRUE WHERE id = $1 AND user_id = $2", keyID, userID,
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

func (s *PostgresStore) UpdateAPIKeyLastUsed(ctx context.Context, keyID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE user_api_keys SET last_used = $1 WHERE id = $2",
		time.Now().UTC(), keyID,
	)
	return err
}

func (s *PostgresStore) ListUsers(ctx context.Context, search, role string, limit, offset int) ([]User, int, error) {
	if limit <= 0 {
		limit = 50
	}

	where := "1=1"
	var args []interface{}
	argN := 1
	if search != "" {
		where += fmt.Sprintf(" AND (email ILIKE $%d OR display_name ILIKE $%d)", argN, argN+1)
		args = append(args, "%"+search+"%", "%"+search+"%")
		argN += 2
	}
	if role != "" {
		where += fmt.Sprintf(" AND role = $%d", argN)
		args = append(args, role)
		argN++
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE "+where, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf("SELECT id, email, password_hash, display_name, description, role, email_verified, created_at, updated_at FROM users WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d", where, argN, argN+1),
		args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Description, &u.Role, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (s *PostgresStore) DeleteUser(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (s *PostgresStore) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func (s *PostgresStore) CreateEmailVerification(ctx context.Context, ev *EmailVerification) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO email_verifications (id, email, code_hash, purpose, expires_at, used, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		ev.ID, ev.Email, ev.CodeHash, ev.Purpose, ev.ExpiresAt.UTC(), ev.Used, ev.CreatedAt.UTC(),
	)
	return err
}

func (s *PostgresStore) GetEmailVerification(ctx context.Context, email, codeHash, purpose string) (*EmailVerification, error) {
	var ev EmailVerification
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, code_hash, purpose, expires_at, used, created_at
		 FROM email_verifications WHERE email = $1 AND code_hash = $2 AND purpose = $3 AND used = FALSE
		 ORDER BY created_at DESC LIMIT 1`, email, codeHash, purpose,
	).Scan(&ev.ID, &ev.Email, &ev.CodeHash, &ev.Purpose, &ev.ExpiresAt, &ev.Used, &ev.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("verification not found")
		}
		return nil, err
	}
	return &ev, nil
}

func (s *PostgresStore) MarkEmailVerificationUsed(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE email_verifications SET used = TRUE WHERE id = $1", id)
	return err
}

func (s *PostgresStore) CountRecentVerifications(ctx context.Context, email string, since time.Time) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM email_verifications WHERE email = $1 AND created_at >= $2",
		email, since.UTC(),
	).Scan(&count)
	return count, err
}

func (s *PostgresStore) DeleteExpiredVerifications(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM email_verifications WHERE expires_at < $1", time.Now().UTC())
	return err
}

func (s *PostgresStore) SetEmailVerified(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE users SET email_verified = TRUE, updated_at = $1 WHERE id = $2",
		time.Now().UTC(), userID,
	)
	return err
}

// Close is a no-op since the db is shared.
func (s *PostgresStore) Close() error {
	return nil
}

func nilTimePtr(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.UTC()
}
