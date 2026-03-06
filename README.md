**English** | [中文](README_zh.md)

# peerclaw-server

The centralized platform service for PeerClaw. Provides agent registration and discovery, WebSocket signaling relay, and protocol bridging (A2A / ACP / MCP) -- serving as the coordination hub of the PeerClaw network.

## Features

- **Agent Registration & Discovery** -- RESTful API for managing Agent Cards, with discovery by capability, protocol, and more
- **WebSocket Signaling** -- Relays WebRTC offer/answer/ICE messages to help agents establish P2P connections
- **Protocol Bridging** -- Built-in adapters for A2A, ACP, and MCP that normalize everything into PeerClaw Envelopes
- **Route Engine** -- Intelligent message routing based on agent capabilities and protocols
- **gRPC Interface** -- High-performance gRPC endpoint (reserved for future use)

## Architecture

```
                    ┌──────────────────────────────────┐
                    │          peerclaw-server          │
                    │                                  │
  HTTP/REST ───────►│  ┌──────────┐  ┌──────────────┐  │
                    │  │ Registry │  │ Route Engine │  │
  WebSocket ───────►│  │ Service  │  │              │  │
                    │  └──────────┘  └──────────────┘  │
  gRPC ────────────►│  ┌──────────┐  ┌──────────────┐  │
                    │  │ Signaling│  │    Bridge     │  │
                    │  │   Hub    │  │   Manager     │  │
                    │  └──────────┘  └──────────────┘  │
                    │       │              │            │
                    │  ┌──────────┐  ┌─────┴────────┐  │
                    │  │  SQLite  │  │ A2A ACP MCP  │  │
                    │  └──────────┘  └──────────────┘  │
                    └──────────────────────────────────┘
```

## Getting Started

### Build from Source

```bash
# Build
cd server
go build -o peerclawd ./cmd/peerclawd

# Start with default settings (:8080 HTTP, :9090 gRPC)
./peerclawd

# Or specify a config file
./peerclawd -config config.yaml
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

## Configuration

All settings are configured via a YAML file. Every field has a sensible default:

```yaml
server:
  http_addr: ":8080"
  grpc_addr: ":9090"

database:
  driver: "sqlite"             # sqlite (default) or postgres
  dsn: "peerclaw.db"           # SQLite file path or PostgreSQL DSN

redis:
  addr: "localhost:6379"       # Redis address (for cross-node signaling)
  password: ""
  db: 0

logging:
  level: "info"                # debug / info / warn / error
  format: "text"               # text / json

bridge:
  a2a:
    enabled: true
  acp:
    enabled: true
  mcp:
    enabled: true

signaling:
  enabled: true
  turn:
    urls: []
    username: ""
    credential: ""

observability:
  enabled: false               # Enable OpenTelemetry
  otlp_endpoint: "localhost:4317"
  service_name: "peerclaw-gateway"
  traces_sampling: 0.1         # Sampling rate (0.0 - 1.0)

rate_limit:
  enabled: true
  requests_per_sec: 100        # Requests per second per IP
  burst_size: 200              # Token bucket burst size
  max_connections: 1000        # Max WebSocket connections
  max_message_bytes: 1048576   # Request body size limit (1 MB)

audit_log:
  enabled: true
  output: "stdout"             # "stdout" or "file:/var/log/peerclaw-audit.log"
```

### PostgreSQL Example

```yaml
database:
  driver: "postgres"
  dsn: "postgres://user:pass@localhost:5432/peerclaw?sslmode=disable"
```

### Cross-Node Signaling with Redis

When a Redis address is configured and the connection succeeds, signaling messages are automatically distributed across nodes via Redis Pub/Sub. If Redis is unavailable, the server falls back to local single-node mode.

### OpenTelemetry

When enabled, traces and metrics are pushed to a collector over OTLP gRPC. A Grafana dashboard template is available at `docs/grafana/peerclaw-overview.json`.

## REST API

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/agents` | Register an agent |
| `GET` | `/api/v1/agents` | List agents (filterable by `protocol`, `capability`, `status`) |
| `GET` | `/api/v1/agents/{id}` | Get a single agent |
| `DELETE` | `/api/v1/agents/{id}` | Deregister an agent |
| `POST` | `/api/v1/agents/{id}/heartbeat` | Report heartbeat |
| `POST` | `/api/v1/discover` | Discover agents by capability or protocol |
| `GET` | `/api/v1/routes` | View routing table |
| `GET` | `/api/v1/routes/resolve` | Resolve a route (`target_id`, `protocol`) |
| `POST` | `/api/v1/bridge/send` | Send a message to an external agent via protocol bridge |
| `GET` | `/api/v1/health` | Health check |

## Protocol Endpoints

PeerClaw Server also acts as a gateway for the A2A / MCP / ACP protocols, allowing external agents to interact with the PeerClaw network directly through standard protocol endpoints.

### A2A (Google Agent-to-Agent)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/a2a` | JSON-RPC endpoint (message/send, tasks/get, tasks/cancel) |
| `GET` | `/.well-known/agent.json` | A2A Agent Card |
| `GET` | `/a2a/tasks/{id}` | Query task status |

### MCP (Model Context Protocol)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/mcp` | Streamable HTTP endpoint (initialize, tools/*, resources/*, prompts/*) |
| `GET` | `/mcp` | SSE stream (server-sent events) |

### ACP (Agent Communication Protocol)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/acp/agents` | List available agents |
| `GET` | `/acp/agents/{name}` | Get an agent manifest |
| `POST` | `/acp/runs` | Create a run (sync or async) |
| `GET` | `/acp/runs/{run_id}` | Query run status |
| `POST` | `/acp/runs/{run_id}/cancel` | Cancel a run |
| `GET` | `/acp/ping` | ACP health check |

## WebSocket Signaling

Connection endpoint: `GET /api/v1/signaling?agent_id={id}`

Message format (JSON):

```json
{
  "type": "offer | answer | ice_candidate | config | ping | pong | bridge_message",
  "from": "agent-id",
  "to": "target-agent-id",
  "sdp": "...",
  "candidate": "...",
  "x25519_public_key": "...",
  "ice_servers": [{"urls": ["turn:turn.example.com:3478"], "username": "...", "credential": "..."}],
  "timestamp": "2025-01-01T00:00:00Z"
}
```

The signaling server automatically populates the `from` field and forwards messages to the target agent. Upon connection, the server pushes a `config` message containing TURN server configuration (if configured). The `offer` / `answer` messages may include an `x25519_public_key` field for establishing end-to-end encrypted sessions. The `bridge_message` type is used to deliver protocol-bridged messages (external agent -> PeerClaw Server -> PeerClaw agent), with the `payload` field carrying the Envelope JSON.

## License

MIT
