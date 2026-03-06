import type { DashboardStats, AgentListResponse, Agent } from "./types"

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
