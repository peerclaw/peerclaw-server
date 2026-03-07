**English** | [中文](README_zh.md)

# peerclaw-server

**AI Agent Identity & Trust Platform — verifiable identity, reputation scoring, endpoint verification, and cross-protocol bridging.**

peerclaw-server is the trust infrastructure for AI agents. It provides cryptographically verifiable identities, EWMA-based reputation scoring from real interactions, endpoint verification, and a public agent directory — all built on top of a full protocol gateway with registry, signaling relay, and protocol bridges (A2A, MCP, ACP).

Start it with one command. No external dependencies required.

```bash
./peerclawd
# → PeerClaw gateway started  http=:8080  grpc=:9090
```

## What It Does

| Capability | What it means for you |
|-----------|----------------------|
| **Reputation Engine** | EWMA scoring from real events (registration, heartbeat, bridge, verification). Trust that's earned, not claimed. |
| **Endpoint Verification** | Challenge-response proof that an agent controls its URL. Ed25519 signed. |
| **Public Directory** | Browse agents by reputation, capability, verification status. No auth required. |
| **Agent Registry** | Agents register their capabilities. Anyone can discover them. Like DNS for agents. |
| **Protocol Bridging** | An MCP agent can call an A2A agent. The gateway translates automatically. |
| **Signaling Relay** | Agents establish direct P2P connections via WebSocket signaling. |
| **Auth & Security** | Ed25519 signature auth, API keys, constant-time token verification. |
| **Observability** | OpenTelemetry traces + metrics, structured logging, audit log. |
| **Horizontal Scaling** | Redis Pub/Sub for multi-node signaling. PostgreSQL for shared storage. |

## Getting Started

### Build from Source

```bash
cd server
go build -o peerclawd ./cmd/peerclawd
./peerclawd
```

### Docker

```dockerfile
FROM golang:1.26-alpine AS build
RUN apk add --no-cache gcc musl-dev
WORKDIR /src
COPY . .
RUN CGO_ENABLED=1 go build -o /peerclawd ./cmd/peerclawd

FROM alpine:3.19
COPY --from=build /peerclawd /usr/local/bin/
EXPOSE 8080 9090
CMD ["peerclawd"]
```

### Verify It's Running

```bash
curl http://localhost:8080/api/v1/health
# {"status":"ok","components":{"database":"ok","signaling":"ok"}}
```

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
| **Signaling** | `internal/signaling/` | WebSocket hub, connection auth, rate limiting |
| **Bridge** | `internal/bridge/` | Protocol adapters (A2A, MCP, ACP) + negotiator |
| **Router** | `internal/router/` | Capability-based message routing |
| **Federation** | `internal/federation/` | Multi-server signal relay, DNS SRV discovery |
| **Reputation** | `internal/reputation/` | EWMA reputation engine, event recording, score computation |
| **Verification** | `internal/verification/` | Challenge-response endpoint verification (SSRF-safe) |
| **Security** | `internal/security/` | URL validation (SSRF protection), safe HTTP client |
| **Config** | `internal/config/` | YAML config with `${ENV_VAR}` secret substitution |
| **Observability** | `internal/observability/` | OpenTelemetry provider setup |
| **Audit** | `internal/audit/` | Security event logging |
| **Identity** | `internal/identity/` | Verifier for API keys and Ed25519 signatures |

## Configuration

All settings via YAML. Every field has a sensible default — you can start with zero config.

```yaml
server:
  http_addr: ":8080"
  grpc_addr: ":9090"
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

Applies to: `redis.password`, `database.dsn`, `signaling.turn.credential`, `federation.auth_token`, and federation peer tokens.

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
| `GET` | `/api/v1/directory` | Browse agent directory (filter: `capability`, `protocol`, `status`, `verified`, `min_score`, `search`; sort: `reputation`, `name`, `registered_at`) |
| `GET` | `/api/v1/directory/{id}` | Public agent profile (sanitized, no auth params) |
| `GET` | `/api/v1/directory/{id}/reputation` | Reputation event history |

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

When `auth.required: true`, all non-public endpoints require one of:

- **Bearer token**: `Authorization: Bearer <api-key>`
- **Ed25519 signature**: `X-PeerClaw-PublicKey` + `X-PeerClaw-Signature` headers

Public endpoints (no auth): `GET /api/v1/health`, `GET /api/v1/directory`, `GET /api/v1/directory/{id}`, `GET /api/v1/directory/{id}/reputation`, `GET /.well-known/agent.json`, `GET /acp/ping`

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

## Deployment Patterns

### Single Node (development)

```bash
./peerclawd  # SQLite, no Redis, everything works
```

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
| **Secrets** | `${ENV_VAR}` config substitution — no plaintext in files |

## License

MIT
