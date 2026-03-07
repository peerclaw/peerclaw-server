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
  public_endpoint?: boolean
  reputation_score?: number
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

// Public directory types

export interface PublicAgentProfile {
  id: string
  name: string
  description?: string
  version?: string
  public_key?: string
  capabilities?: string[]
  skills?: { name: string; description?: string }[]
  protocols?: string[]
  status: "online" | "offline" | "degraded"
  tags?: string[]
  verified: boolean
  verified_at?: string
  trusted: boolean
  reputation_score: number
  reputation_events: number
  endpoint_url?: string
  registered_at: string
  review_summary?: ReviewSummary
  categories?: string[]
}

export interface DirectoryResponse {
  agents: PublicAgentProfile[]
  next_page_token?: string
  total_count: number
}

export interface DirectoryParams {
  capability?: string
  protocol?: string
  status?: string
  verified?: boolean
  min_score?: number
  search?: string
  sort?: "reputation" | "name" | "registered_at"
  page_size?: number
  page_token?: string
  category?: string
}

export interface ReputationEvent {
  id: number
  agent_id: string
  event_type: string
  weight: number
  score_after: number
  metadata?: string
  created_at: string
}

// Review & Community types

export interface Review {
  id: string
  agent_id: string
  user_id: string
  rating: number
  comment: string
  created_at: string
  updated_at: string
}

export interface ReviewSummary {
  average_rating: number
  total_reviews: number
  distribution: number[]
}

export interface Category {
  id: string
  name: string
  slug: string
  description: string
  icon: string
  sort_order: number
}

export interface AbuseReport {
  id: string
  reporter_id: string
  target_type: string
  target_id: string
  reason: string
  details: string
  status: string
  created_at: string
}

// Invocation types

export interface InvokeRequest {
  message: string
  protocol?: string
  metadata?: Record<string, string>
  stream?: boolean
}

export interface InvokeResponse {
  invocation_id: string
  agent_id: string
  protocol: string
  response: string
  duration_ms: number
}

export interface InvocationRecord {
  id: string
  agent_id: string
  user_id: string
  protocol: string
  status_code: number
  duration_ms: number
  error: string
  created_at: string
}

// Provider types

export interface AgentInvocationStats {
  total_calls: number
  success_calls: number
  error_calls: number
  avg_duration_ms: number
  p95_duration_ms: number
}

export interface TimeSeriesPoint {
  bucket: string
  count: number
  errors: number
  avg_duration_ms: number
}

export interface ProviderDashboardStats {
  total_agents: number
  total_calls: number
  total_errors: number
  avg_duration_ms: number
  agents: Array<{
    id: string
    name: string
    total_calls: number
    error_rate: number
  }>
}
