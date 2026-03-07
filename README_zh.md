[English](README.md) | **中文**

# peerclaw-server

**AI Agent 身份与信任平台 — 可验证身份、声誉评分、端点验证、跨协议桥接。**

peerclaw-server 是 AI Agent 的信任基础设施。它提供密码学可验证身份、基于真实交互的 EWMA 声誉评分、端点验证、以及公开的 Agent 目录 — 一切都构建在完整的协议网关之上，包括注册中心、信令中转和协议桥接（A2A、MCP、ACP）。

一行命令启动，零外部依赖。

```bash
./peerclawd
# → PeerClaw gateway started  http=:8080
```

## 核心能力

| 能力 | 对你意味着什么 |
|------|--------------|
| **声誉引擎** | 基于真实事件（注册、心跳、桥接、验证）的 EWMA 评分。信任是赢得的，不是声称的。 |
| **端点验证** | Challenge-Response 证明 Agent 控制其 URL，Ed25519 签名。 |
| **公开目录** | 按声誉、能力、验证状态浏览 Agent，无需认证。 |
| **Agent 注册中心** | Agent 注册自己的能力，任何人都能发现它。像 Agent 的 DNS。 |
| **协议桥接** | MCP Agent 可以调用 A2A Agent，网关自动翻译。 |
| **信令中转** | Agent 通过 WebSocket 信令建立 P2P 直连。 |
| **认证鉴权** | Ed25519 签名认证、API Key、恒时 token 比较。 |
| **可观测性** | OpenTelemetry 链路追踪 + 指标、结构化日志、审计日志。 |
| **水平扩展** | Redis Pub/Sub 多节点信令，PostgreSQL 共享存储。 |

## 快速开始

### 从源码构建

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
EXPOSE 8080
CMD ["peerclawd"]
```

### 验证运行

```bash
curl http://localhost:8080/api/v1/health
# {"status":"ok","components":{"database":"ok","signaling":"ok"}}
```

## 架构

```
                         传入请求
                           │
                ┌──────────▼──────────┐
                │      中间件链        │
                │  CORS → 认证 →      │
                │  限速 → 链路追踪    │
                └──────────┬──────────┘
                           │
      ┌────────────────────┼────────────────────┐
      ▼                    ▼                    ▼
┌─────────────┐    ┌──────────────┐    ┌──────────────────┐
│  注册中心    │    │   信令中心    │    │    协议桥接器      │
│             │    │              │    │                  │
│ POST/GET    │    │  WebSocket   │    │ ┌────┬────┬────┐ │
│ /api/v1/    │    │  中转 WebRTC │    │ │A2A │MCP │ACP │ │
│  agents     │    │  信令        │    │ └────┴────┴────┘ │
└──────┬──────┘    └──────┬──────┘    └────────┬─────────┘
       │                  │                    │
┌──────▼──────┐    ┌──────▼──────┐    ┌───────▼─────────┐
│  SQLite 或  │    │  Redis 或   │    │    路由引擎      │
│  PostgreSQL │    │  本地 Broker │    │  能力 + 协议     │
│             │    │             │    │  匹配            │
└─────────────┘    └─────────────┘    └─────────────────┘
```

### 内部模块

| 模块 | 路径 | 用途 |
|------|------|------|
| **HTTP 服务器** | `internal/server/` | 路由、中间件链、请求处理 |
| **认证** | `internal/server/auth.go` | Bearer Token + Ed25519 签名认证 |
| **校验** | `internal/server/validation.go` | 注册和心跳的输入校验 |
| **注册中心** | `internal/registry/` | Agent CRUD、能力索引（SQLite/PostgreSQL） |
| **信令** | `internal/signaling/` | WebSocket Hub、连接认证、消息限速 |
| **桥接** | `internal/bridge/` | 协议适配器（A2A、MCP、ACP）+ 协商器 |
| **路由** | `internal/router/` | 基于能力的消息路由 |
| **联邦** | `internal/federation/` | 多服务器信令中转、DNS SRV 发现 |
| **声誉** | `internal/reputation/` | EWMA 声誉引擎、事件记录、分数计算 |
| **验证** | `internal/verification/` | Challenge-Response 端点验证（SSRF 安全） |
| **安全** | `internal/security/` | URL 校验（SSRF 防护）、安全 HTTP 客户端 |
| **配置** | `internal/config/` | YAML 配置 + `${ENV_VAR}` 密钥替换 |
| **可观测** | `internal/observability/` | OpenTelemetry Provider 初始化 |
| **审计** | `internal/audit/` | 安全事件日志 |
| **身份** | `internal/identity/` | API Key 和 Ed25519 签名验证器 |

## 配置

所有配置通过 YAML 文件。每个字段都有合理的默认值 — 零配置即可启动。

```yaml
server:
  http_addr: ":8080"
  cors_origins: []                   # 如 ["https://dashboard.example.com"]

auth:
  required: false                    # 生产环境设为 true

database:
  driver: "sqlite"                   # "sqlite" 或 "postgres"
  dsn: "peerclaw.db"

redis:
  addr: "localhost:6379"
  password: "${REDIS_PASSWORD}"      # 支持环境变量替换
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
  auth_token: "${FEDERATION_TOKEN}"  # 联邦启用时必填
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
  output: "stdout"                   # 或 "file:/var/log/peerclaw-audit.log"

logging:
  level: "info"
  format: "text"                     # "text" 或 "json"
```

### 环境变量替换

敏感字段支持 `${ENV_VAR}` 语法：

```yaml
redis:
  password: "${REDIS_PASSWORD}"      # 从 REDIS_PASSWORD 环境变量读取
```

适用于：`redis.password`、`database.dsn`、`signaling.turn.credential`、`federation.auth_token` 及联邦 peer token。

## REST API

### Agent 管理

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/agents` | 注册 Agent |
| `GET` | `/api/v1/agents` | 列出 Agent（过滤：`protocol`、`capability`、`status`） |
| `GET` | `/api/v1/agents/{id}` | 获取 Agent 详情 |
| `DELETE` | `/api/v1/agents/{id}` | 注销 Agent（仅所有者） |
| `POST` | `/api/v1/agents/{id}/heartbeat` | 上报心跳（仅所有者） |

### 公开目录（免认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/directory` | 浏览 Agent 目录（过滤：`capability`、`protocol`、`status`、`verified`、`min_score`、`search`；排序：`reputation`、`name`、`registered_at`） |
| `GET` | `/api/v1/directory/{id}` | Agent 公开档案（脱敏，不含认证参数） |
| `GET` | `/api/v1/directory/{id}/reputation` | 声誉事件历史 |

### 端点验证

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/agents/{id}/verify` | 发起端点验证（仅所有者） |

### 发现与路由

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/discover` | 按能力或协议发现 Agent |
| `GET` | `/api/v1/routes` | 查看路由表 |
| `GET` | `/api/v1/routes/resolve` | 解析路由（`target_id`、`protocol`） |

### 桥接与健康

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/bridge/send` | 通过协议桥接发送消息 |
| `GET` | `/api/v1/health` | 健康检查 |

### 认证

当 `auth.required: true` 时，所有非公开端点需要以下认证方式之一：

- **Bearer Token**：`Authorization: Bearer <api-key>`
- **Ed25519 签名**：`X-PeerClaw-PublicKey` + `X-PeerClaw-Signature` 请求头

公开端点（免认证）：`GET /api/v1/health`、`GET /api/v1/directory`、`GET /api/v1/directory/{id}`、`GET /api/v1/directory/{id}/reputation`、`GET /.well-known/agent.json`、`GET /acp/ping`

## 协议网关端点

服务器同时暴露标准协议端点，外部 Agent 可以使用原生协议与 PeerClaw Agent 交互：

### A2A（Google Agent-to-Agent）

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/a2a` | JSON-RPC 2.0（message/send、tasks/get、tasks/cancel） |
| `GET` | `/.well-known/agent.json` | A2A Agent Card |
| `GET` | `/a2a/tasks/{id}` | 查询任务状态 |

### MCP（Model Context Protocol）

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/mcp` | Streamable HTTP（initialize、tools/\*、resources/\*、prompts/\*） |
| `GET` | `/mcp` | SSE 事件流 |

### ACP（Agent Communication Protocol）

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/acp/agents` | 列出可用 Agent |
| `GET` | `/acp/agents/{name}` | Agent 清单 |
| `POST` | `/acp/runs` | 创建运行 |
| `GET` | `/acp/runs/{run_id}` | 运行状态 |
| `POST` | `/acp/runs/{run_id}/cancel` | 取消运行 |
| `GET` | `/acp/ping` | 健康检查 |

## WebSocket 信令

**端点**：`GET /api/v1/signaling?agent_id={id}`

用于 WebRTC 信令 — Agent 通过此通道交换 offer/answer/ICE candidate。当 `auth.required` 为 true 时，客户端必须在连接后 5 秒内发送认证帧（agent_id + 时间戳 + Ed25519 签名）。

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

特性：
- 64KB 消息大小限制
- 每连接限速（10 条/秒）
- 连接时自动推送 TURN 配置
- `bridge_message` 类型用于投递协议桥接的 Envelope

## 部署模式

### 单节点（开发环境）

```bash
./peerclawd  # SQLite，无需 Redis，开箱即用
```

### 生产环境（多节点）

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

### 联邦模式

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

注册在不同服务器上的 Agent 可以通过联邦中转互相发现和通信。

## 安全

| 层级 | 防护 |
|------|------|
| **认证** | 所有端点的 Ed25519 签名或 API Key |
| **授权** | 所有者专属路由（DELETE、心跳） |
| **输入校验** | 名称长度、公钥格式、能力数量限制 |
| **SSRF 防护** | 桥接适配器中的 URL 校验，阻止私有 IP |
| **限速** | 每 IP 令牌桶、受信代理支持 |
| **联邦** | 恒时 token 比较、TLS 1.2 最低版本 |
| **WebSocket** | 认证帧超时、消息大小/速率限制 |
| **密钥** | `${ENV_VAR}` 配置替换 — 文件中无明文密钥 |

## 许可证

MIT
