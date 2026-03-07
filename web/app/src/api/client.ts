import type {
  DashboardStats,
  AgentListResponse,
  Agent,
  DirectoryResponse,
  DirectoryParams,
  PublicAgentProfile,
  ReputationEvent,
} from "./types"

const BASE = "/api/v1"

async function fetchJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`)
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export function fetchStats(): Promise<DashboardStats> {
  return fetchJSON<DashboardStats>("/dashboard/stats")
}

export function fetchAgents(params?: {
  protocol?: string
  status?: string
}): Promise<AgentListResponse> {
  const query = new URLSearchParams()
  if (params?.protocol) query.set("protocol", params.protocol)
  if (params?.status) query.set("status", params.status)
  const qs = query.toString()
  return fetchJSON<AgentListResponse>(`/agents${qs ? `?${qs}` : ""}`)
}

export function fetchAgent(id: string): Promise<Agent> {
  return fetchJSON<Agent>(`/agents/${id}`)
}

// Public directory API

export function fetchDirectory(
  params?: DirectoryParams
): Promise<DirectoryResponse> {
  const query = new URLSearchParams()
  if (params?.capability) query.set("capability", params.capability)
  if (params?.protocol) query.set("protocol", params.protocol)
  if (params?.status) query.set("status", params.status)
  if (params?.verified) query.set("verified", "true")
  if (params?.min_score) query.set("min_score", String(params.min_score))
  if (params?.search) query.set("search", params.search)
  if (params?.sort) query.set("sort", params.sort)
  if (params?.page_size) query.set("page_size", String(params.page_size))
  if (params?.page_token) query.set("page_token", params.page_token)
  const qs = query.toString()
  return fetchJSON<DirectoryResponse>(`/directory${qs ? `?${qs}` : ""}`)
}

export function fetchPublicProfile(id: string): Promise<PublicAgentProfile> {
  return fetchJSON<PublicAgentProfile>(`/directory/${id}`)
}

export function fetchReputationHistory(
  id: string,
  limit?: number
): Promise<{ events: ReputationEvent[] }> {
  const query = limit ? `?limit=${limit}` : ""
  return fetchJSON<{ events: ReputationEvent[] }>(
    `/directory/${id}/reputation${query}`
  )
}
