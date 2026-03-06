[English](README.md) | **中文**

# peerclaw-server

PeerClaw 中心化平台服务。提供 Agent 注册与发现、WebSocket 信令中继、协议桥接（A2A / ACP / MCP），是 PeerClaw 网络的协调中枢。

## 功能

- **Agent 注册与发现** — RESTful API 管理 Agent Card，按能力、协议等维度发现 Agent
- **WebSocket 信令** — 中继 WebRTC offer/answer/ICE，帮助 Agent 建立 P2P 连接
- **协议桥接** — 内置 A2A、ACP、MCP 适配器，统一转换为 PeerClaw Envelope
- **路由引擎** — 基于 Agent 能力和协议的智能消息路由
- **gRPC 接口** — 高性能 gRPC 入口（预留）

## 架构

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

## 快速启动

### 本地编译

```bash
# 编译
cd server
go build -o peerclawd ./cmd/peerclawd

# 使用默认配置启动（:8080 HTTP, :9090 gRPC）
./peerclawd

# 或指定配置文件
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

## 配置

通过 YAML 文件配置，所有字段均有默认值：

```yaml
server:
  http_addr: ":8080"
  grpc_addr: ":9090"

database:
  driver: "sqlite"             # sqlite (default) 或 postgres
  dsn: "peerclaw.db"           # SQLite 路径或 PostgreSQL DSN

redis:
  addr: "localhost:6379"       # Redis 地址（跨节点信令）
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
  enabled: false               # 启用 OpenTelemetry
  otlp_endpoint: "localhost:4317"
  service_name: "peerclaw-gateway"
  traces_sampling: 0.1         # 采样率 (0.0 - 1.0)

rate_limit:
  enabled: true
  requests_per_sec: 100        # 每 IP 每秒请求数
  burst_size: 200              # 令牌桶突发大小
  max_connections: 1000        # WebSocket 最大连接数
  max_message_bytes: 1048576   # 请求体大小限制 (1MB)

audit_log:
  enabled: true
  output: "stdout"             # "stdout" 或 "file:/var/log/peerclaw-audit.log"
```

### PostgreSQL 配置示例

```yaml
database:
  driver: "postgres"
  dsn: "postgres://user:pass@localhost:5432/peerclaw?sslmode=disable"
```

### Redis 跨节点信令

当配置了 Redis 地址且连接成功时，信令消息自动通过 Redis Pub/Sub 跨节点分发。
Redis 不可用时自动回退到本地单节点模式。

### OpenTelemetry

启用后通过 OTLP gRPC 推送 traces 和 metrics 到 collector。Grafana dashboard 模板位于 `docs/grafana/peerclaw-overview.json`。

## REST API

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/agents` | 注册 Agent |
| `GET` | `/api/v1/agents` | 列出 Agent（支持 `protocol`、`capability`、`status` 过滤） |
| `GET` | `/api/v1/agents/{id}` | 获取单个 Agent |
| `DELETE` | `/api/v1/agents/{id}` | 注销 Agent |
| `POST` | `/api/v1/agents/{id}/heartbeat` | 心跳上报 |
| `POST` | `/api/v1/discover` | 按能力/协议发现 Agent |
| `GET` | `/api/v1/routes` | 查看路由表 |
| `GET` | `/api/v1/routes/resolve` | 解析路由（`target_id`、`protocol`） |
| `POST` | `/api/v1/bridge/send` | 通过协议桥接发送消息到外部 Agent |
| `GET` | `/api/v1/health` | 健康检查 |

## 协议端点

PeerClaw Server 同时作为 A2A / MCP / ACP 协议的网关，外部 Agent 可直接通过标准协议端点与 PeerClaw 网络交互。

### A2A (Google Agent-to-Agent)

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/a2a` | JSON-RPC 端点（message/send, tasks/get, tasks/cancel） |
| `GET` | `/.well-known/agent.json` | A2A Agent Card |
| `GET` | `/a2a/tasks/{id}` | 查询 Task 状态 |

### MCP (Model Context Protocol)

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/mcp` | Streamable HTTP 端点（initialize, tools/*, resources/*, prompts/*） |
| `GET` | `/mcp` | SSE 流（服务端推送） |

### ACP (Agent Communication Protocol)

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/acp/agents` | 列出可用 Agent |
| `GET` | `/acp/agents/{name}` | 获取 Agent Manifest |
| `POST` | `/acp/runs` | 创建 Run（同步/异步） |
| `GET` | `/acp/runs/{run_id}` | 查询 Run 状态 |
| `POST` | `/acp/runs/{run_id}/cancel` | 取消 Run |
| `GET` | `/acp/ping` | ACP 健康检查 |

## WebSocket 信令

连接端点：`GET /api/v1/signaling?agent_id={id}`

消息格式（JSON）：

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

信令服务器自动填充 `from` 字段并转发到目标 Agent。连接建立后，Server 自动推送 `config` 消息，携带 TURN 服务器配置（如已配置）。`offer` / `answer` 消息中可携带 `x25519_public_key` 字段用于建立端到端加密会话。`bridge_message` 类型用于投递协议桥接消息（外部 Agent → PeerClaw Server → PeerClaw Agent），`payload` 字段携带 Envelope JSON。

## License

MIT
