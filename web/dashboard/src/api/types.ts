export interface Endpoint {
  url: string
  host: string
  port: number
  transport: string
}

export interface PeerClawExtension {
  nat_type?: string
  relay_preference?: string
  priority?: number
  tags?: string[]
}

export interface Agent {
  id: string
  name: string
  description: string
  version: string
  public_key: string
  capabilities: string[]
  protocols: string[]
  status: "online" | "offline" | "degraded"
  endpoint: Endpoint
  metadata: Record<string, string>
  peerclaw: PeerClawExtension
  registered_at: string
  last_heartbeat: string
}

export interface AgentSummary {
  id: string
  name: string
  status: string
  protocols: string[]
  last_heartbeat: string
}

export interface BridgeStats {
  protocol: string
  available: boolean
  task_count: number
}

export interface HealthStatus {
  status: string
  components: Record<string, string>
}

export interface DashboardStats {
  registered_agents: number
  connected_agents: number
  bridges: BridgeStats[]
  health: HealthStatus
  recent_agents: AgentSummary[]
}

export interface AgentListResponse {
  agents: Agent[]
  next_page_token: string
  total_count: number
}

export interface AgentFilters {
  protocol?: string
  status?: string
  search?: string
}
