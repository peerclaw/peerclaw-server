# Federation Multi-Node Architecture

## Overview

PeerClaw supports a federated multi-node architecture that enables multiple gateway nodes to forward signaling messages, discover agents from remote peers, and relay WebRTC signals across geographically distributed instances.

## Three-Layer Broker Architecture

Signal distribution uses a layered broker model:

```
┌──────────────────────────────────────────────────────┐
│  Agent A <-> WebSocket <-> Hub <-> Broker.Publish()  │
│                                     |                │
│                        +------------+------------+   │
│                        v            v            v   │
│                   LocalBroker  RedisBroker  FederationBroker
│                   (single)     (cluster)     (cross-region)
└──────────────────────────────────────────────────────┘
```

- **LocalBroker** -- In-memory forwarding, single-node development
- **RedisBroker** -- Redis Pub/Sub, multi-node within same cluster
- **FederationBroker** -- HTTP forwarding, cross-network/cross-organization relay

### Key Source Files

| File | Purpose |
|------|---------|
| `internal/signaling/broker.go` | Broker interface definition |
| `internal/signaling/local_broker.go` | Single-node in-memory broker |
| `internal/signaling/redis_broker.go` | Multi-node Redis Pub/Sub broker |
| `internal/signaling/federation_broker.go` | Hybrid local + federation broker |
| `internal/signaling/ws.go` | WebSocket hub and agent connection management |
| `internal/federation/federation.go` | Core federation service, peer management, signal forwarding |
| `internal/federation/dns.go` | DNS SRV-based peer discovery |

## Signal Forwarding Flow

```
Agent A (Node 1, Beijing)  sends signal to  Agent B (Node 2, Shanghai)

1. Agent A sends SignalMessage{To: "agent-b"} via WebSocket
2. Hub.Forward() -> Broker.Publish()
3. FederationBroker checks: hub.HasAgent("agent-b")?
   +-- YES -> Local WebSocket delivery
   +-- NO  -> Iterate all peer nodes
             POST /api/v1/federation/signal -> Node 2
             Header: Authorization: Bearer <peer-token>
4. Node 2 receives request:
   - Validates Bearer token (constant-time comparison)
   - Calls HandleIncomingSignal()
   - Hub.DeliverLocal() -> delivers to Agent B via WebSocket
```

Automatic fallback: if federation forwarding fails, it falls back to the local broker.

## Peer Discovery

### Method 1: Static Configuration

```yaml
federation:
  enabled: true
  node_name: "beijing-1"
  auth_token: "${FEDERATION_TOKEN}"
  peers:
    - name: "shanghai-1"
      address: "https://sh.example.com"
      token: "${PEER_SH_TOKEN}"
    - name: "guangzhou-1"
      address: "https://gz.example.com"
      token: "${PEER_GZ_TOKEN}"
```

### Method 2: DNS SRV Auto-Discovery

```yaml
federation:
  enabled: true
  dns_enabled: true
  dns_domain: "peerclaw.example.com"
```

Queries `_peerclaw._tcp.peerclaw.example.com` SRV records to automatically discover all node host:port pairs and join the federation.

Implementation in `internal/federation/dns.go`:
- `DiscoverPeers(domain)` performs `net.LookupSRV("peerclaw", "tcp", domain)`
- Constructs HTTP URLs from SRV target and port
- Called during initialization if `dns_enabled=true`

## Redis Multi-Node Mode (Intra-Cluster)

Unlike federation, Redis mode is for horizontal scaling of multiple PeerClaw instances **within the same cluster**:

```
         +--- Node 1 (peerclawd) ---+
Agent A -|                          |-- Redis Pub/Sub --+
         +--------------------------+   channel:        |
                                        "peerclaw:      |
         +--- Node 2 (peerclawd) ---+   signaling"     |
Agent B -|                          |-------------------+
         +--------------------------+
```

Key design decisions:
- **Echo prevention**: Each node has a unique `NodeID` (UUID). On receiving a message, it skips messages it sent itself (`env.NodeID == b.nodeID`).
- **256-message buffer**: Non-blocking channel prevents slow consumers from blocking.
- **Automatic fallback**: If Redis is unavailable, falls back to LocalBroker.

### Redis Envelope Format

```go
type redisEnvelope struct {
    NodeID  string                  `json:"node_id"`
    Message signaling.SignalMessage `json:"message"`
}
```

Published to Redis channel `peerclaw:signaling`.

## Federation API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/federation/signal` | POST | Receive signal forwarding from other nodes |
| `/api/v1/federation/discover` | POST | Receive agent discovery queries from other nodes |

Handlers are in `internal/server/http.go`:
- `handleFederationSignal` -- validates Bearer token, parses SignalMessage, calls `HandleIncomingSignal()`
- `handleFederationDiscover` -- queries local registry, returns matching agents

## Cross-Node Agent Discovery

```go
// Node 1 wants to find agents with "translation" capability
agents := fedService.QueryAgents(ctx, []string{"translation"})
// -> Concurrently queries all peers' /api/v1/federation/discover
// -> Aggregates results from all nodes
```

Implementation in `federation.go` `QueryAgents()`:
1. Queries `/api/v1/discover` on each peer
2. Filters by capabilities
3. Aggregates results from all peers
4. Returns combined agent list

## Security Model

| Layer | Mechanism |
|-------|-----------|
| Local signaling | Agent signature-based (Ed25519 public key verification) |
| Federation | Bearer token per peer (`crypto/subtle.ConstantTimeCompare`) |
| Redis | No authentication (internal trusted network) |
| Federation HTTP client | TLS 1.2 minimum, 10-second timeout |

## Core Data Structures

### FederationService

```go
type FederationService struct {
    mu        sync.RWMutex
    nodeName  string
    peers     map[string]*FederationPeer
    authToken string
    logger    *slog.Logger
    client    *http.Client
    onSignal  func(ctx context.Context, msg signaling.SignalMessage)
    stopCh    chan struct{}
}
```

### FederationPeer

```go
type FederationPeer struct {
    Name      string
    Address   string     // URL to peer node
    Token     string     // Auth token for this peer
    Connected bool
    LastSync  time.Time
}
```

### FederationBroker -- Hybrid Routing Logic

```go
func (fb *FederationBroker) Publish(ctx context.Context, msg SignalMessage) error {
    if fb.hub.HasAgent(msg.To) {
        return fb.local.Publish(ctx, msg)    // Local delivery
    }
    if err := fb.federation.ForwardSignal(ctx, msg); err != nil {
        return fb.local.Publish(ctx, msg)    // Fallback to local
    }
    return nil
}
```

## Deployment Topologies

### Topology 1: Load-Balanced Cluster (Redis Mode)

```
                  +--- Load Balancer ---+
                  |                     |
            +-----+-----+       +------+-----+
            |  Node 1   |       |  Node 2    |
            | peerclawd |       | peerclawd  |
            +-----+-----+       +------+-----+
                  |    Shared Redis     |
                  +--------+-----------+
                           |
                     +-----+-----+
                     |  Redis    |
                     +-----+-----+
                           |
                     +-----+------+
                     | PostgreSQL |
                     +------------+
```

### Topology 2: Cross-Region Federation

```
  Beijing DC                       Shanghai DC
+-------------+    HTTP/TLS    +-------------+
|  Node BJ-1  |<------------->|  Node SH-1  |
|  + Redis    |   Federation   |  + Redis    |
|  + PG       |   Signal       |  + PG       |
|  Agents:    |                |  Agents:    |
|  A, B, C    |                |  D, E, F    |
+-------------+                +-------------+
       ^                              ^
       | DNS SRV: _peerclaw._tcp.corp |
       +---------- Auto-discovery ----+
```

### Topology 3: Hybrid (Redis + Federation)

```
  Cluster A (3 nodes + Redis)       Cluster B (2 nodes + Redis)
+-------------------------+      +-------------------------+
| Node1 <-Redis-> Node2   |<--->| Node4 <-Redis-> Node5   |
|       ^                 | Fed  |                         |
|     Node3               |      +-------------------------+
+-------------------------+
```

Intra-cluster uses Redis broadcast, inter-cluster uses Federation HTTP forwarding.

## Initialization Flow

```
main.go
+-- Load config
+-- Initialize logger
+-- Initialize OpenTelemetry
+-- Initialize database
+-- Initialize signaling hub (if enabled)
|   +-- SetBroker()
|       +-- LocalBroker (if no Redis)
|       +-- RedisBroker (if Redis available)
+-- Initialize federation (if enabled)
|   +-- Create FederationService
|   +-- Add static peers from config
|   +-- Discover DNS SRV peers (if dns_enabled)
+-- Create HTTP server
|   +-- SetFederation()
|   +-- Register routes:
|       +-- POST /api/v1/federation/signal
|       +-- POST /api/v1/federation/discover
+-- Start servers and wait for shutdown
    +-- Clean shutdown: fedService.Close()
```

## Comparison

| Feature | LocalBroker | RedisBroker | FederationBroker |
|---------|------------|-------------|-----------------|
| Node count | 1 | N (same cluster) | N (cross-network) |
| Transport | In-memory | Redis Pub/Sub | HTTPS |
| Latency | ~0 | ~1ms | ~10-100ms |
| Auth | None | None (trusted LAN) | Bearer Token |
| Agent discovery | Local | Shared DB | HTTP query |
| Use case | Development | Production cluster | Multi-region/multi-org |

## Configuration Reference

```yaml
federation:
  enabled: false               # Enable federation
  node_name: "node-1"          # This node's unique name
  auth_token: "${FED_TOKEN}"   # Auth token (required when enabled)
  dns_enabled: false           # Enable DNS SRV discovery
  dns_domain: ""               # Domain for SRV lookup
  peers:                       # Static peer list
    - name: "node-2"
      address: "https://node2.example.com"
      token: "${PEER_TOKEN}"
```

Validation: `federation.auth_token` is required when `federation.enabled = true`.
Environment variable substitution (`${VAR}`) is supported for all token fields.
