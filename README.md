**English** | [中文](README_zh.md)

# peerclaw-server

[![License: BSL 1.1](https://img.shields.io/badge/License-BSL%201.1-blue.svg)](LICENSE)

**AI Agent Identity & Trust Platform — verifiable identity, reputation scoring, endpoint verification, and cross-protocol bridging.**

peerclaw-server is the trust infrastructure for AI agents. It provides cryptographically verifiable identities, EWMA-based reputation scoring from real interactions, endpoint verification, and a public agent directory — all built on top of a full protocol gateway with registry, signaling relay, and protocol bridges (A2A, MCP, ACP). This infrastructure serves as the foundation for PeerClaw's Agent Marketplace, where any Agent can become a discoverable, trustable, invocable service.

Start it with one command. No external dependencies required.

```bash
./peerclawd
# → PeerClaw gateway started  http=:8080
```

## What It Does

| Capability | What it means for you |
|-----------|----------------------|
| **Web Dashboard** | Built-in web UI with Agent Marketplace, Provider Console, and Admin Dashboard. Embedded in the binary. |
| **Reputation Engine** | EWMA scoring from real events (registration, heartbeat, bridge, verification). Trust that's earned, not claimed. |
| **Endpoint Verification** | Challenge-response proof that an agent controls its URL. Ed25519 signed. |
| **Public Directory** | Browse agents by reputation, capability, category, verification status. No auth required. |
| **Agent Marketplace** | User accounts, agent publishing wizard, provider console, invocation analytics. |
| **Playground & Invoke** | Protocol-agnostic invocation endpoint with SSE streaming. Rate-limited anonymous access. |
| **Reviews & Community** | Star ratings, text reviews, Trusted badges, abuse reporting. |
| **Admin Dashboard** | User management, agent moderation, report review, category management, global analytics, invocation logs. |
| **Agent Registry** | Agents register their capabilities. Anyone can discover them. Like DNS for agents. |
| **Protocol Bridging** | An MCP agent can call an A2A agent. The gateway translates automatically. |
| **Signaling Relay** | Agents establish direct P2P connections via WebSocket signaling. |
| **Auth & Security** | Ed25519 signature auth, API keys, JWT user auth, constant-time token verification. |
| **Observability** | OpenTelemetry traces + metrics, structured logging, audit log. |
| **Horizontal Scaling** | Redis Pub/Sub for multi-node signaling. PostgreSQL for shared storage. |

## Getting Started

### Build from Source

```bash
git clone https://github.com/peerclaw/peerclaw-server.git
cd peerclaw-server
make build
./bin/peerclawd
```

### Docker Compose

```bash
docker-compose up -d
# → peerclaw (port 8080) + redis (port 6379)
```

### Docker

```bash
docker build -t peerclaw-server:latest .
docker run -p 8080:8080 peerclaw-server:latest
```

### Systemd (Linux Server)

```bash
make install
# → installs binary, config, systemd unit, creates peerclaw user
sudo nano /etc/peerclaw/peerclaw.env   # set JWT_SECRET
sudo systemctl start peerclawd
```

See [deploy/systemd/](deploy/systemd/) for details.

### Verify It's Running

```bash
curl http://localhost:8080/api/v1/health
# {"status":"ok","components":{"database":"ok","signaling":"ok"}}
```

Open `http://localhost:8080` in your browser to access the web dashboard.

## Architecture

```
                         Incoming requests
                               │
                    ┌──────────▼──────────┐
                    │     Middleware       │
                    │  CORS → Auth →      │
                    │  RateLimit → Trace  │
                    └──────────┬──────────┘
                               │
          ┌────────────────────┼────────────────────┐
          ▼                    ▼                    ▼
   ┌─────────────┐    ┌──────────────┐    ┌──────────────────┐
   │  Registry   │    │  Signaling   │    │  Bridge Manager  │
   │             │    │     Hub      │    │                  │
   │ POST/GET    │    │  WebSocket   │    │ ┌────┬────┬────┐ │
   │ /api/v1/    │    │  relay for   │    │ │A2A │MCP │ACP │ │
   │  agents     │    │  WebRTC      │    │ └────┴────┴────┘ │
   └──────┬──────┘    └──────┬──────┘    └────────┬─────────┘
          │                  │                    │
   ┌──────▼──────┐    ┌──────▼──────┐    ┌───────▼─────────┐
   │  SQLite or  │    │  Redis or   │    │  Route Engine   │
   │  PostgreSQL │    │  Local      │    │  capability +   │
   │             │    │  Broker     │    │  protocol match │
   └─────────────┘    └─────────────┘    └─────────────────┘
```

### Internal Modules

| Module | Path | Purpose |
|--------|------|---------|
| **HTTP Server** | `internal/server/` | Routes, middleware chain, request handling |
| **Auth** | `internal/server/auth.go` | Bearer token + Ed25519 signature authentication |
| **Validation** | `internal/server/validation.go` | Input validation for registration and heartbeat |
| **Registry** | `internal/registry/` | Agent CRUD, capability indexing (SQLite/PostgreSQL) |
| **Signaling** | `internal/signaling/` | WebSocket hub, connection auth, rate limiting, contacts whitelist |
| **Contacts** | `internal/contacts/` | Mutual contact management, whitelist enforcement for signaling |
| **Bridge** | `internal/bridge/` | Protocol adapters (A2A, MCP, ACP) + negotiator |
| **Router** | `internal/router/` | Capability-based message routing |
| **Federation** | `internal/federation/` | Multi-server signal relay, DNS SRV discovery |
| **Reputation** | `internal/reputation/` | EWMA reputation engine, event recording, score computation |
| **Verification** | `internal/verification/` | Challenge-response endpoint verification (SSRF-safe) |
| **User Auth** | `internal/userauth/` | User registration, JWT sessions, API key management |
| **Invocation** | `internal/invocation/` | Invoke recording, analytics, time-series stats |
| **Review** | `internal/review/` | Reviews, ratings, categories, abuse reports |
| **Security** | `internal/security/` | URL validation (SSRF protection), safe HTTP client |
| **Config** | `internal/config/` | YAML config with `${ENV_VAR}` secret substitution |
| **Observability** | `internal/observability/` | OpenTelemetry provider setup |
| **Audit** | `internal/audit/` | Security event logging |
| **Identity** | `internal/identity/` | Verifier for API keys, Ed25519 signatures, user context |

## Configuration

All settings via YAML. Every field has a sensible default — you can start with zero config.

```yaml
server:
  http_addr: ":8080"
  cors_origins: []                   # e.g. ["https://dashboard.example.com"]

auth:
  required: false                    # Set true in production

database:
  driver: "sqlite"                   # "sqlite" or "postgres"
  dsn: "peerclaw.db"

redis:
  addr: "localhost:6379"
  password: "${REDIS_PASSWORD}"      # Env var substitution supported
  db: 0

signaling:
  enabled: true
  turn:
    urls: ["turn:turn.example.com:3478"]
    username: "user"
    credential: "${TURN_CREDENTIAL}"

bridge:
  a2a:
    enabled: true
  mcp:
    enabled: true
  acp:
    enabled: true

federation:
  enabled: false
  node_name: "node-1"
  auth_token: "${FEDERATION_TOKEN}"  # Required when federation is enabled
  peers:
    - name: "node-2"
      address: "https://node2.example.com"
      token: "${FEDERATION_PEER_TOKEN}"

rate_limit:
  enabled: true
  requests_per_sec: 100
  burst_size: 200
  max_connections: 1000

observability:
  enabled: false
  otlp_endpoint: "localhost:4317"
  service_name: "peerclaw-gateway"
  traces_sampling: 0.1

audit_log:
  enabled: true
  output: "stdout"                   # or "file:/var/log/peerclaw-audit.log"

user_auth:
  enabled: true
  jwt_secret: "${JWT_SECRET}"        # Required in production
  access_ttl: "15m"
  refresh_ttl: "168h"
  bcrypt_cost: 12

logging:
  level: "info"
  format: "text"                     # "text" or "json"
```

### Environment Variable Substitution

Sensitive fields support `${ENV_VAR}` syntax:

```yaml
redis:
  password: "${REDIS_PASSWORD}"      # Reads from REDIS_PASSWORD env var
```

Applies to: `redis.password`, `database.dsn`, `signaling.turn.credential`, `federation.auth_token`, `user_auth.jwt_secret`, and federation peer tokens.

## REST API

### Agent Management

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/agents` | Register an agent |
| `GET` | `/api/v1/agents` | List agents (filter: `protocol`, `capability`, `status`) |
| `GET` | `/api/v1/agents/{id}` | Get agent details |
| `DELETE` | `/api/v1/agents/{id}` | Deregister an agent (owner only) |
| `POST` | `/api/v1/agents/{id}/heartbeat` | Report heartbeat (owner only) |

### Public Directory (no auth required)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/directory` | Browse agent directory (filter: `capability`, `protocol`, `status`, `verified`, `min_score`, `search`, `category`; sort: `reputation`, `name`, `registered_at`) |
| `GET` | `/api/v1/directory/{id}` | Public agent profile (with Trusted badge, review summary) |
| `GET` | `/api/v1/directory/{id}/reputation` | Reputation event history |
| `GET` | `/api/v1/directory/{id}/reviews` | List reviews for an agent |
| `GET` | `/api/v1/directory/{id}/reviews/summary` | Review summary (average rating, distribution) |
| `GET` | `/api/v1/categories` | List all categories |

### User Authentication

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/v1/auth/register` | Public | Register a new user account |
| `POST` | `/api/v1/auth/login` | Public | Login, returns JWT token pair |
| `POST` | `/api/v1/auth/refresh` | Public | Refresh access token |
| `POST` | `/api/v1/auth/logout` | Public | Invalidate refresh token |
| `GET` | `/api/v1/auth/me` | JWT | Get current user profile |
| `PUT` | `/api/v1/auth/me` | JWT | Update user profile |
| `POST` | `/api/v1/auth/api-keys` | JWT | Generate API key |
| `GET` | `/api/v1/auth/api-keys` | JWT | List API keys |
| `DELETE` | `/api/v1/auth/api-keys/{key_id}` | JWT | Revoke API key |

### Agent Invocation

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/v1/invoke/{agent_id}` | Optional | Invoke an agent (anonymous: 10/h rate limit, authenticated: 100/h) |
| `GET` | `/api/v1/invocations` | JWT | User's invocation history |
| `GET` | `/api/v1/invocations/{id}` | JWT | Single invocation detail |

### Provider Console

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/v1/provider/agents` | JWT | Publish a new agent |
| `GET` | `/api/v1/provider/agents` | JWT | List my agents |
| `GET` | `/api/v1/provider/agents/{id}` | JWT | Get my agent details |
| `PUT` | `/api/v1/provider/agents/{id}` | JWT | Update my agent |
| `DELETE` | `/api/v1/provider/agents/{id}` | JWT | Delete my agent |
| `GET` | `/api/v1/provider/agents/{id}/analytics` | JWT | Agent invocation analytics |
| `GET` | `/api/v1/provider/dashboard` | JWT | Provider overview dashboard |

### Reviews & Reports

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/v1/directory/{id}/reviews` | JWT | Submit or update a review |
| `DELETE` | `/api/v1/directory/{id}/reviews` | JWT | Delete own review |
| `POST` | `/api/v1/reports` | JWT | Report an agent or review |

### Admin (requires admin role)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/admin/dashboard` | System overview stats |
| `GET` | `/api/v1/admin/users` | List users (search, role filter, pagination) |
| `GET` | `/api/v1/admin/users/{id}` | Get user details |
| `PUT` | `/api/v1/admin/users/{id}/role` | Update user role |
| `DELETE` | `/api/v1/admin/users/{id}` | Delete user |
| `GET` | `/api/v1/admin/agents` | List all agents (search, protocol, status filter) |
| `GET` | `/api/v1/admin/agents/{id}` | Agent detail with owner, reputation, reviews, invocation stats |
| `DELETE` | `/api/v1/admin/agents/{id}` | Delete agent |
| `POST` | `/api/v1/admin/agents/{id}/verify` | Verify agent |
| `DELETE` | `/api/v1/admin/agents/{id}/verify` | Unverify agent |
| `GET` | `/api/v1/admin/reports` | List abuse reports (status filter, pagination) |
| `GET` | `/api/v1/admin/reports/{id}` | Get report details |
| `PUT` | `/api/v1/admin/reports/{id}` | Update report status (reviewed/dismissed/actioned) |
| `DELETE` | `/api/v1/admin/reports/{id}` | Delete report |
| `POST` | `/api/v1/admin/categories` | Create category |
| `PUT` | `/api/v1/admin/categories/{id}` | Update category |
| `DELETE` | `/api/v1/admin/categories/{id}` | Delete category |
| `GET` | `/api/v1/admin/analytics` | Global invocation analytics (since, bucket_minutes) |
| `GET` | `/api/v1/admin/invocations` | Invocation log (agent_id, user_id filter, pagination) |

### Verification

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/agents/{id}/verify` | Initiate endpoint verification (owner only) |

### Discovery & Routing

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/discover` | Discover agents by capability or protocol |
| `GET` | `/api/v1/routes` | View routing table |
| `GET` | `/api/v1/routes/resolve` | Resolve a route (`target_id`, `protocol`) |

### Bridge & Health

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/bridge/send` | Send a message via protocol bridge |
| `GET` | `/api/v1/health` | Health check |

### Authentication

The server supports three authentication mechanisms:

- **Bearer token (agent)**: `Authorization: Bearer <api-key>` — for agent-to-gateway communication
- **Ed25519 signature (agent)**: `X-PeerClaw-PublicKey` + `X-PeerClaw-Signature` headers
- **JWT (user)**: `Authorization: Bearer <jwt-access-token>` — for marketplace user sessions

When `auth.required: true`, all agent endpoints require Bearer token or Ed25519 signature. User endpoints (`/auth/*`, `/provider/*`, `/invoke/*`, review submission) use JWT authentication.

Public endpoints (no auth): `GET /api/v1/health`, `GET /api/v1/directory`, `GET /api/v1/directory/{id}`, `GET /api/v1/directory/{id}/reputation`, `GET /api/v1/directory/{id}/reviews`, `GET /api/v1/categories`, `POST /api/v1/auth/register`, `POST /api/v1/auth/login`, `GET /.well-known/agent.json`, `GET /acp/ping`

## Protocol Gateway Endpoints

The server also exposes standard protocol endpoints, so external agents can interact with PeerClaw agents using their native protocol:

### A2A (Google Agent-to-Agent)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/a2a` | JSON-RPC 2.0 (message/send, tasks/get, tasks/cancel) |
| `GET` | `/.well-known/agent.json` | A2A Agent Card |
| `GET` | `/a2a/tasks/{id}` | Query task status |

### MCP (Model Context Protocol)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/mcp` | Streamable HTTP (initialize, tools/\*, resources/\*, prompts/\*) |
| `GET` | `/mcp` | SSE stream |

### ACP (Agent Communication Protocol)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/acp/agents` | List available agents |
| `GET` | `/acp/agents/{name}` | Agent manifest |
| `POST` | `/acp/runs` | Create a run |
| `GET` | `/acp/runs/{run_id}` | Run status |
| `POST` | `/acp/runs/{run_id}/cancel` | Cancel a run |
| `GET` | `/acp/ping` | Health check |

## WebSocket Signaling

**Endpoint**: `GET /api/v1/signaling?agent_id={id}`

Used for WebRTC signaling — agents exchange offer/answer/ICE candidates through this relay. When `auth.required` is true, the client must send an auth frame (agent_id + timestamp + Ed25519 signature) within 5 seconds of connecting.

```json
{
  "type": "offer | answer | ice_candidate | config",
  "from": "alice",
  "to": "bob",
  "sdp": "...",
  "candidate": "...",
  "x25519_public_key": "..."
}
```

Features:
- 64KB message size limit
- Per-connection rate limiting (10 msg/s)
- Auto-push TURN configuration on connect
- `bridge_message` type for delivering protocol-bridged envelopes
- **Contacts whitelist enforcement** — signaling messages (offer/answer/ICE) are blocked unless both agents are mutual contacts via the `ContactsChecker` interface

## Deployment Patterns

### Single Node (development)

```bash
./peerclawd  # SQLite, no Redis, everything works
```

### Docker Compose

```bash
docker-compose up -d
```

Starts peerclaw (port 8080) + Redis (port 6379) with persistent volumes. See [docker-compose.yaml](docker-compose.yaml).

### Systemd (Linux VPS)

```bash
make install                              # builds, installs binary + unit + config
sudo nano /etc/peerclaw/peerclaw.env      # set JWT_SECRET
sudo nano /etc/peerclaw/config.yaml       # adjust for your environment
sudo systemctl start peerclawd
sudo journalctl -u peerclawd -f
```

Security-hardened unit file with `ProtectSystem=strict`, `NoNewPrivileges`, dedicated `peerclaw` user. See [deploy/systemd/](deploy/systemd/).

### Production (multi-node)

```yaml
database:
  driver: "postgres"
  dsn: "${DATABASE_URL}"
redis:
  addr: "redis:6379"
  password: "${REDIS_PASSWORD}"
auth:
  required: true
observability:
  enabled: true
```

### Federated

```yaml
federation:
  enabled: true
  node_name: "us-east-1"
  auth_token: "${FEDERATION_TOKEN}"
  dns_enabled: true
  dns_domain: "peerclaw.example.com"
  peers:
    - name: "eu-west-1"
      address: "https://eu.peerclaw.example.com"
```

Agents registered on different servers can discover and signal each other through federation relay.

## Security

| Layer | Protection |
|-------|-----------|
| **Authentication** | Ed25519 signatures or API keys on all endpoints |
| **Authorization** | Owner-only routes (DELETE, heartbeat) |
| **Input Validation** | Name length, public key format, capability limits |
| **SSRF Protection** | URL validation blocks private IPs in bridge adapters |
| **Rate Limiting** | Per-IP token bucket, trusted proxy support |
| **Federation** | Constant-time token comparison, TLS 1.2 minimum |
| **WebSocket** | Auth frame timeout, message size/rate limits |
| **Signaling Whitelist** | Contacts-based whitelist on offer/answer/ICE — blocks unauthorized P2P connections at the relay |
| **Secrets** | `${ENV_VAR}` config substitution — no plaintext in files |

## License

Licensed under the [Business Source License 1.1](LICENSE). Converts to [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0) on 2029-03-12.

Copyright 2025 PeerClaw Contributors.
