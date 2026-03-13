package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-core/protocol"
)

// PostgresStore implements Store using PostgreSQL.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgreSQL-backed store.
func NewPostgresStore(dsn string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	s := &PostgresStore{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *PostgresStore) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS agents (
		id                 TEXT PRIMARY KEY,
		name               TEXT NOT NULL,
		description        TEXT DEFAULT '',
		version            TEXT DEFAULT '',
		public_key         TEXT DEFAULT '',
		capabilities       JSONB DEFAULT '[]',
		endpoint_url       TEXT DEFAULT '',
		endpoint_host      TEXT DEFAULT '',
		endpoint_port      INTEGER DEFAULT 0,
		endpoint_transport TEXT DEFAULT '',
		protocols          JSONB DEFAULT '[]',
		auth_type          TEXT DEFAULT '',
		auth_params        JSONB DEFAULT '{}',
		metadata           JSONB DEFAULT '{}',
		peerclaw_nat       TEXT DEFAULT '',
		peerclaw_relay     TEXT DEFAULT '',
		peerclaw_priority  INTEGER DEFAULT 0,
		peerclaw_tags      JSONB DEFAULT '[]',
		skills             JSONB DEFAULT '[]',
		tools              JSONB DEFAULT '[]',
		status             TEXT DEFAULT 'online',
		registered_at      TIMESTAMPTZ NOT NULL,
		last_heartbeat     TIMESTAMPTZ NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);
	CREATE INDEX IF NOT EXISTS idx_agents_name ON agents(name);
	`
	// Create GIN index on JSONB columns for capability/protocol queries.
	ginIndex := `
	CREATE INDEX IF NOT EXISTS idx_agents_capabilities ON agents USING GIN (capabilities);
	CREATE INDEX IF NOT EXISTS idx_agents_protocols ON agents USING GIN (protocols);
	`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}
	_, err := s.db.Exec(ginIndex)
	return err
}

func (s *PostgresStore) Put(ctx context.Context, card *agentcard.Card) error {
	caps, _ := json.Marshal(card.Capabilities)
	protos, _ := json.Marshal(card.Protocols)
	authParams, _ := json.Marshal(card.Auth.Params)
	meta, _ := json.Marshal(card.Metadata)
	tags, _ := json.Marshal(card.PeerClaw.Tags)
	skills, _ := json.Marshal(card.Skills)
	tools, _ := json.Marshal(card.Tools)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agents (
			id, name, description, version, public_key, capabilities,
			endpoint_url, endpoint_host, endpoint_port, endpoint_transport,
			protocols, auth_type, auth_params, metadata,
			peerclaw_nat, peerclaw_relay, peerclaw_priority, peerclaw_tags,
			skills, tools,
			status, registered_at, last_heartbeat
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
		ON CONFLICT(id) DO UPDATE SET
			name=EXCLUDED.name, description=EXCLUDED.description, version=EXCLUDED.version,
			public_key=EXCLUDED.public_key, capabilities=EXCLUDED.capabilities,
			endpoint_url=EXCLUDED.endpoint_url,
			endpoint_host=EXCLUDED.endpoint_host, endpoint_port=EXCLUDED.endpoint_port,
			endpoint_transport=EXCLUDED.endpoint_transport, protocols=EXCLUDED.protocols,
			auth_type=EXCLUDED.auth_type, auth_params=EXCLUDED.auth_params,
			metadata=EXCLUDED.metadata, peerclaw_nat=EXCLUDED.peerclaw_nat,
			peerclaw_relay=EXCLUDED.peerclaw_relay, peerclaw_priority=EXCLUDED.peerclaw_priority,
			peerclaw_tags=EXCLUDED.peerclaw_tags, skills=EXCLUDED.skills, tools=EXCLUDED.tools,
			status=EXCLUDED.status, last_heartbeat=EXCLUDED.last_heartbeat
	`,
		card.ID, card.Name, card.Description, card.Version, card.PublicKey, string(caps),
		card.Endpoint.URL, card.Endpoint.Host, card.Endpoint.Port, string(card.Endpoint.Transport),
		string(protos), card.Auth.Type, string(authParams), string(meta),
		card.PeerClaw.NATType, card.PeerClaw.RelayPreference, card.PeerClaw.Priority, string(tags),
		string(skills), string(tools),
		string(card.Status), card.RegisteredAt.UTC(), card.LastHeartbeat.UTC(),
	)
	return err
}

func (s *PostgresStore) Get(ctx context.Context, id string) (*agentcard.Card, error) {
	row := s.db.QueryRowContext(ctx, `SELECT
		id, name, COALESCE(description, ''), COALESCE(version, ''), COALESCE(public_key, ''),
		COALESCE(capabilities::text, '[]'),
		COALESCE(endpoint_url, ''), COALESCE(endpoint_host, ''), endpoint_port, COALESCE(endpoint_transport, ''),
		COALESCE(protocols::text, '[]'), COALESCE(auth_type, ''), COALESCE(auth_params::text, '{}'),
		COALESCE(metadata::text, '{}'),
		COALESCE(peerclaw_nat, ''), COALESCE(peerclaw_relay, ''), peerclaw_priority,
		COALESCE(peerclaw_tags::text, '[]'),
		COALESCE(skills::text, '[]'), COALESCE(tools::text, '[]'),
		status, registered_at, last_heartbeat
		FROM agents WHERE id = $1`, id)
	return s.scanCard(row)
}

func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM agents WHERE id = $1", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("agent %s not found", id)
	}
	return nil
}

func (s *PostgresStore) List(ctx context.Context, filter ListFilter) (*ListResult, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if filter.Protocol != "" {
		conditions = append(conditions, fmt.Sprintf("protocols @> $%d::jsonb", argIdx))
		v, _ := json.Marshal([]string{filter.Protocol})
		args = append(args, string(v))
		argIdx++
	}
	if filter.Capability != "" {
		conditions = append(conditions, fmt.Sprintf("capabilities @> $%d::jsonb", argIdx))
		v, _ := json.Marshal([]string{filter.Capability})
		args = append(args, string(v))
		argIdx++
	}
	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(filter.Status))
		argIdx++
	}
	if filter.PlaygroundOnly {
		conditions = append(conditions, "COALESCE(playground_enabled, FALSE) = TRUE")
	}
	if filter.IncludeOwnerUserID != "" {
		// Show public agents + this user's own agents (including private ones).
		conditions = append(conditions, fmt.Sprintf("(COALESCE(visibility, 'public') = 'public' OR owner_user_id = $%d)", argIdx))
		args = append(args, filter.IncludeOwnerUserID)
		argIdx++
	} else if filter.PublicOnly {
		conditions = append(conditions, "COALESCE(visibility, 'public') = 'public'")
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total.
	var total int
	countQuery := "SELECT COUNT(*) FROM agents " + where
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	pageSize := filter.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 50
	}

	offset := 0
	if filter.PageToken != "" {
		_, _ = fmt.Sscanf(filter.PageToken, "%d", &offset)
	}

	query := fmt.Sprintf(`SELECT
		id, name, COALESCE(description, ''), COALESCE(version, ''), COALESCE(public_key, ''),
		COALESCE(capabilities::text, '[]'),
		COALESCE(endpoint_url, ''), COALESCE(endpoint_host, ''), endpoint_port, COALESCE(endpoint_transport, ''),
		COALESCE(protocols::text, '[]'), COALESCE(auth_type, ''), COALESCE(auth_params::text, '{}'),
		COALESCE(metadata::text, '{}'),
		COALESCE(peerclaw_nat, ''), COALESCE(peerclaw_relay, ''), peerclaw_priority,
		COALESCE(peerclaw_tags::text, '[]'),
		COALESCE(skills::text, '[]'), COALESCE(tools::text, '[]'),
		status, registered_at, last_heartbeat
		FROM agents %s ORDER BY registered_at DESC LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)

	args = append(args, pageSize, offset)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []*agentcard.Card
	for rows.Next() {
		card, err := s.scanCardFromRows(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, card)
	}

	result := &ListResult{
		Agents:     agents,
		TotalCount: total,
	}
	nextOffset := offset + pageSize
	if nextOffset < total {
		result.NextPageToken = fmt.Sprintf("%d", nextOffset)
	}
	return result, rows.Err()
}

func (s *PostgresStore) UpdateHeartbeat(ctx context.Context, id string, status agentcard.AgentStatus) error {
	res, err := s.db.ExecContext(ctx,
		"UPDATE agents SET last_heartbeat = $1, status = $2 WHERE id = $3",
		time.Now().UTC(), string(status), id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("agent %s not found", id)
	}
	return nil
}

func (s *PostgresStore) FindByCapabilities(ctx context.Context, capabilities []string, proto string, maxResults int) ([]*agentcard.Card, error) {
	var conditions []string
	var args []any
	argIdx := 1

	for _, cap := range capabilities {
		conditions = append(conditions, fmt.Sprintf("capabilities @> $%d::jsonb", argIdx))
		v, _ := json.Marshal([]string{cap})
		args = append(args, string(v))
		argIdx++
	}
	if proto != "" {
		conditions = append(conditions, fmt.Sprintf("protocols @> $%d::jsonb", argIdx))
		v, _ := json.Marshal([]string{proto})
		args = append(args, string(v))
		argIdx++
	}
	conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
	args = append(args, string(agentcard.StatusOnline))
	argIdx++

	where := "WHERE " + strings.Join(conditions, " AND ")
	if maxResults <= 0 {
		maxResults = 20
	}

	query := fmt.Sprintf(`SELECT
		id, name, COALESCE(description, ''), COALESCE(version, ''), COALESCE(public_key, ''),
		COALESCE(capabilities::text, '[]'),
		COALESCE(endpoint_url, ''), COALESCE(endpoint_host, ''), endpoint_port, COALESCE(endpoint_transport, ''),
		COALESCE(protocols::text, '[]'), COALESCE(auth_type, ''), COALESCE(auth_params::text, '{}'),
		COALESCE(metadata::text, '{}'),
		COALESCE(peerclaw_nat, ''), COALESCE(peerclaw_relay, ''), peerclaw_priority,
		COALESCE(peerclaw_tags::text, '[]'),
		COALESCE(skills::text, '[]'), COALESCE(tools::text, '[]'),
		status, registered_at, last_heartbeat
		FROM agents %s ORDER BY peerclaw_priority DESC LIMIT $%d`, where, argIdx)

	args = append(args, maxResults)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []*agentcard.Card
	for rows.Next() {
		card, err := s.scanCardFromRows(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, card)
	}
	return agents, rows.Err()
}

func (s *PostgresStore) ListByOwner(ctx context.Context, userID string, filter ListFilter) (*ListResult, error) {
	filter.OwnerUserID = userID
	return s.List(ctx, filter)
}

func (s *PostgresStore) GetDB() interface{} {
	return s.db
}

func (s *PostgresStore) GetAccessFlags(ctx context.Context, id string) (*AccessFlags, error) {
	var playgroundEnabled bool
	var visibility string
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(playground_enabled, FALSE), COALESCE(visibility, 'public') FROM agents WHERE id = $1`, id,
	).Scan(&playgroundEnabled, &visibility)
	if err != nil {
		return nil, fmt.Errorf("get access flags: %w", err)
	}
	return &AccessFlags{
		PlaygroundEnabled: playgroundEnabled,
		Visibility:        visibility,
	}, nil
}

func (s *PostgresStore) GetAccessFlagsBatch(ctx context.Context, ids []string) (map[string]*AccessFlags, error) {
	if len(ids) == 0 {
		return map[string]*AccessFlags{}, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(
		`SELECT id, COALESCE(playground_enabled, FALSE), COALESCE(visibility, 'public') FROM agents WHERE id IN (%s)`,
		strings.Join(placeholders, ","),
	)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get access flags batch: %w", err)
	}
	defer rows.Close()

	result := make(map[string]*AccessFlags, len(ids))
	for rows.Next() {
		var id string
		var playgroundEnabled bool
		var visibility string
		if err := rows.Scan(&id, &playgroundEnabled, &visibility); err != nil {
			return nil, fmt.Errorf("scan access flags: %w", err)
		}
		result[id] = &AccessFlags{
			PlaygroundEnabled: playgroundEnabled,
			Visibility:        visibility,
		}
	}
	return result, rows.Err()
}

func (s *PostgresStore) SetAccessFlags(ctx context.Context, id string, flags *AccessFlags) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE agents SET playground_enabled = $1, visibility = $2 WHERE id = $3`,
		flags.PlaygroundEnabled, flags.Visibility, id,
	)
	return err
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}

func (s *PostgresStore) scanCard(row *sql.Row) (*agentcard.Card, error) {
	card := &agentcard.Card{}
	var caps, protos, authParams, meta, tags, skills, tools string
	var status, transport string
	var regAt, hbAt time.Time

	err := row.Scan(
		&card.ID, &card.Name, &card.Description, &card.Version, &card.PublicKey, &caps,
		&card.Endpoint.URL, &card.Endpoint.Host, &card.Endpoint.Port, &transport,
		&protos, &card.Auth.Type, &authParams, &meta,
		&card.PeerClaw.NATType, &card.PeerClaw.RelayPreference, &card.PeerClaw.Priority, &tags,
		&skills, &tools,
		&status, &regAt, &hbAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("agent not found")
		}
		return nil, err
	}
	s.unmarshalCard(card, caps, protos, authParams, meta, tags, skills, tools, status, transport, regAt, hbAt)
	return card, nil
}

func (s *PostgresStore) scanCardFromRows(rows *sql.Rows) (*agentcard.Card, error) {
	card := &agentcard.Card{}
	var caps, protos, authParams, meta, tags, skills, tools string
	var status, transport string
	var regAt, hbAt time.Time

	err := rows.Scan(
		&card.ID, &card.Name, &card.Description, &card.Version, &card.PublicKey, &caps,
		&card.Endpoint.URL, &card.Endpoint.Host, &card.Endpoint.Port, &transport,
		&protos, &card.Auth.Type, &authParams, &meta,
		&card.PeerClaw.NATType, &card.PeerClaw.RelayPreference, &card.PeerClaw.Priority, &tags,
		&skills, &tools,
		&status, &regAt, &hbAt,
	)
	if err != nil {
		return nil, err
	}
	s.unmarshalCard(card, caps, protos, authParams, meta, tags, skills, tools, status, transport, regAt, hbAt)
	return card, nil
}

func (s *PostgresStore) unmarshalCard(card *agentcard.Card, caps, protos, authParams, meta, tags, skills, tools, status, transport string, regAt, hbAt time.Time) {
	_ = json.Unmarshal([]byte(caps), &card.Capabilities)
	_ = json.Unmarshal([]byte(protos), &card.Protocols)
	_ = json.Unmarshal([]byte(authParams), &card.Auth.Params)
	_ = json.Unmarshal([]byte(meta), &card.Metadata)
	_ = json.Unmarshal([]byte(tags), &card.PeerClaw.Tags)
	_ = json.Unmarshal([]byte(skills), &card.Skills)
	_ = json.Unmarshal([]byte(tools), &card.Tools)
	card.Status = agentcard.AgentStatus(status)
	card.Endpoint.Transport = protocol.Transport(transport)
	card.RegisteredAt = regAt
	card.LastHeartbeat = hbAt
}
