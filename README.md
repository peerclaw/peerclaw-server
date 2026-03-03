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
FROM golang:1.24-alpine AS build
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
  dsn: "peerclaw.db"          # SQLite 数据库路径

redis:
  addr: "localhost:6379"       # 用于水平扩展（Phase 4）
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
```

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
| `GET` | `/api/v1/health` | 健康检查 |

## WebSocket 信令

连接端点：`GET /api/v1/signaling?agent_id={id}`

消息格式（JSON）：

```json
{
  "type": "offer | answer | ice_candidate | ping | pong",
  "from": "agent-id",
  "to": "target-agent-id",
  "sdp": "...",
  "candidate": "...",
  "timestamp": "2025-01-01T00:00:00Z"
}
```

信令服务器自动填充 `from` 字段并转发到目标 Agent。

## License

MIT
