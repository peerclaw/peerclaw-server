package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/peerclaw/peerclaw-core/agentcard"
	"github.com/peerclaw/peerclaw-core/protocol"
)

// SQLiteStore implements Store using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite-backed store.
func NewSQLiteStore(dsn string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// Enable WAL mode for better concurrent read performance.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *SQLiteStore) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS agents (
		id             TEXT PRIMARY KEY,
		name           TEXT NOT NULL,
		description    TEXT,
		version        TEXT,
		public_key     TEXT,
		capabilities   TEXT, -- JSON array
		endpoint_url   TEXT,
		endpoint_host  TEXT,
		endpoint_port  INTEGER,
		endpoint_transport TEXT,
		protocols      TEXT, -- JSON array
		auth_type      TEXT,
		auth_params    TEXT, -- JSON object
		metadata       TEXT, -- JSON object
		peerclaw_nat   TEXT,
		peerclaw_relay TEXT,
		peerclaw_priority INTEGER DEFAULT 0,
		peerclaw_tags  TEXT, -- JSON array
		skills         TEXT, -- JSON array
		tools          TEXT, -- JSON array
		status         TEXT DEFAULT 'online',
		registered_at  DATETIME NOT NULL,
		last_heartbeat DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);
	CREATE INDEX IF NOT EXISTS idx_agents_name ON agents(name);
	`
	_, err := s.db.Exec(schema)
	return err
}

func (s *SQLiteStore) Put(ctx context.Context, card *agentcard.Card) error {
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
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name, description=excluded.description, version=excluded.version,
			public_key=excluded.public_key, capabilities=excluded.capabilities,
			endpoint_url=excluded.endpoint_url,
			endpoint_host=excluded.endpoint_host, endpoint_port=excluded.endpoint_port,
			endpoint_transport=excluded.endpoint_transport, protocols=excluded.protocols,
			auth_type=excluded.auth_type, auth_params=excluded.auth_params,
			metadata=excluded.metadata, peerclaw_nat=excluded.peerclaw_nat,
			peerclaw_relay=excluded.peerclaw_relay, peerclaw_priority=excluded.peerclaw_priority,
			peerclaw_tags=excluded.peerclaw_tags, skills=excluded.skills, tools=excluded.tools,
			status=excluded.status, last_heartbeat=excluded.last_heartbeat
	`,
		card.ID, card.Name, card.Description, card.Version, card.PublicKey, string(caps),
		card.Endpoint.URL, card.Endpoint.Host, card.Endpoint.Port, string(card.Endpoint.Transport),
		string(protos), card.Auth.Type, string(authParams), string(meta),
		card.PeerClaw.NATType, card.PeerClaw.RelayPreference, card.PeerClaw.Priority, string(tags),
		string(skills), string(tools),
		string(card.Status), card.RegisteredAt.UTC().Format(time.RFC3339),
		card.LastHeartbeat.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) Get(ctx context.Context, id string) (*agentcard.Card, error) {
	row := s.db.QueryRowContext(ctx, `SELECT
		id, name, description, version, COALESCE(public_key, ''), capabilities,
		endpoint_url, endpoint_host, endpoint_port, endpoint_transport,
		protocols, auth_type, auth_params, metadata,
		peerclaw_nat, peerclaw_relay, peerclaw_priority, peerclaw_tags,
		COALESCE(skills, '[]'), COALESCE(tools, '[]'),
		status, registered_at, last_heartbeat
		FROM agents WHERE id = ?`, id)
	return s.scanCard(row)
}

func (s *SQLiteStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM agents WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("agent %s not found", id)
	}
	return nil
}

func (s *SQLiteStore) List(ctx context.Context, filter ListFilter) (*ListResult, error) {
	var conditions []string
	var args []any

	if filter.Protocol != "" {
		conditions = append(conditions, "protocols LIKE ?")
		args = append(args, "%"+filter.Protocol+"%")
	}
	if filter.Capability != "" {
		conditions = append(conditions, "capabilities LIKE ?")
		args = append(args, "%"+filter.Capability+"%")
	}
	if filter.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, string(filter.Status))
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
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
		fmt.Sscanf(filter.PageToken, "%d", &offset)
	}

	query := fmt.Sprintf(`SELECT
		id, name, description, version, COALESCE(public_key, ''), capabilities,
		endpoint_url, endpoint_host, endpoint_port, endpoint_transport,
		protocols, auth_type, auth_params, metadata,
		peerclaw_nat, peerclaw_relay, peerclaw_priority, peerclaw_tags,
		COALESCE(skills, '[]'), COALESCE(tools, '[]'),
		status, registered_at, last_heartbeat
		FROM agents %s ORDER BY registered_at DESC LIMIT ? OFFSET ?`, where)

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

func (s *SQLiteStore) UpdateHeartbeat(ctx context.Context, id string, status agentcard.AgentStatus) error {
	res, err := s.db.ExecContext(ctx,
		"UPDATE agents SET last_heartbeat = ?, status = ? WHERE id = ?",
		time.Now().UTC().Format(time.RFC3339), string(status), id,
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

func (s *SQLiteStore) FindByCapabilities(ctx context.Context, capabilities []string, proto string, maxResults int) ([]*agentcard.Card, error) {
	var conditions []string
	var args []any

	for _, cap := range capabilities {
		conditions = append(conditions, "capabilities LIKE ?")
		args = append(args, "%"+cap+"%")
	}
	if proto != "" {
		conditions = append(conditions, "protocols LIKE ?")
		args = append(args, "%"+proto+"%")
	}
	conditions = append(conditions, "status = ?")
	args = append(args, string(agentcard.StatusOnline))

	where := "WHERE " + strings.Join(conditions, " AND ")
	if maxResults <= 0 {
		maxResults = 20
	}

	query := fmt.Sprintf(`SELECT
		id, name, description, version, COALESCE(public_key, ''), capabilities,
		endpoint_url, endpoint_host, endpoint_port, endpoint_transport,
		protocols, auth_type, auth_params, metadata,
		peerclaw_nat, peerclaw_relay, peerclaw_priority, peerclaw_tags,
		COALESCE(skills, '[]'), COALESCE(tools, '[]'),
		status, registered_at, last_heartbeat
		FROM agents %s ORDER BY peerclaw_priority DESC LIMIT ?`, where)

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

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

type scanner interface {
	Scan(dest ...any) error
}

func (s *SQLiteStore) scanCard(row *sql.Row) (*agentcard.Card, error) {
	card := &agentcard.Card{}
	var caps, protos, authParams, meta, tags, skills, tools string
	var status, transport, regAt, hbAt string

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

func (s *SQLiteStore) scanCardFromRows(rows *sql.Rows) (*agentcard.Card, error) {
	card := &agentcard.Card{}
	var caps, protos, authParams, meta, tags, skills, tools string
	var status, transport, regAt, hbAt string

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

func (s *SQLiteStore) unmarshalCard(card *agentcard.Card, caps, protos, authParams, meta, tags, skills, tools, status, transport, regAt, hbAt string) {
	json.Unmarshal([]byte(caps), &card.Capabilities)
	json.Unmarshal([]byte(protos), &card.Protocols)
	json.Unmarshal([]byte(authParams), &card.Auth.Params)
	json.Unmarshal([]byte(meta), &card.Metadata)
	json.Unmarshal([]byte(tags), &card.PeerClaw.Tags)
	json.Unmarshal([]byte(skills), &card.Skills)
	json.Unmarshal([]byte(tools), &card.Tools)
	card.Status = agentcard.AgentStatus(status)
	card.Endpoint.Transport = protocol.Transport(transport)
	card.RegisteredAt, _ = time.Parse(time.RFC3339, regAt)
	card.LastHeartbeat, _ = time.Parse(time.RFC3339, hbAt)
}
