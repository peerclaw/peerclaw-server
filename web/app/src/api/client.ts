import type {
  DashboardStats,
  AgentListResponse,
  Agent,
  DirectoryResponse,
  DirectoryParams,
  PublicAgentProfile,
  ReputationEvent,
  Review,
  ReviewSummary,
  Category,
  InvokeResponse,
} from "./types"

const BASE = "/api/v1"

async function fetchJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`)
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export async function fetchWithAuth<T>(
  path: string,
  accessToken: string,
  options: RequestInit = {}
): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${accessToken}`,
      ...options.headers,
    },
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error || `API error: ${res.status}`)
  }
  if (res.status === 204) return undefined as T
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
  if (params?.category) query.set("category", params.category)
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

// Reviews API

export function fetchReviews(
  agentId: string,
  limit?: number,
  offset?: number
): Promise<{ reviews: Review[]; total: number }> {
  const query = new URLSearchParams()
  if (limit) query.set("limit", String(limit))
  if (offset) query.set("offset", String(offset))
  const qs = query.toString()
  return fetchJSON<{ reviews: Review[]; total: number }>(
    `/directory/${agentId}/reviews${qs ? `?${qs}` : ""}`
  )
}

export function fetchReviewSummary(agentId: string): Promise<ReviewSummary> {
  return fetchJSON<ReviewSummary>(`/directory/${agentId}/reviews/summary`)
}

export function submitReview(
  agentId: string,
  rating: number,
  comment: string,
  accessToken: string
): Promise<Review> {
  return fetchWithAuth<Review>(`/directory/${agentId}/reviews`, accessToken, {
    method: "POST",
    body: JSON.stringify({ rating, comment }),
  })
}

export function deleteReview(
  agentId: string,
  accessToken: string
): Promise<void> {
  return fetchWithAuth<void>(`/directory/${agentId}/reviews`, accessToken, {
    method: "DELETE",
  })
}

// Categories API

export function fetchCategories(): Promise<{ categories: Category[] }> {
  return fetchJSON<{ categories: Category[] }>("/categories")
}

// Reports API

export function submitReport(
  targetType: string,
  targetId: string,
  reason: string,
  details: string,
  accessToken: string
): Promise<{ status: string }> {
  return fetchWithAuth<{ status: string }>("/reports", accessToken, {
    method: "POST",
    body: JSON.stringify({ target_type: targetType, target_id: targetId, reason, details }),
  })
}

// Invoke API

export function invokeAgent(
  agentId: string,
  message: string,
  options?: { protocol?: string; metadata?: Record<string, string>; accessToken?: string }
): Promise<InvokeResponse> {
  const headers: Record<string, string> = { "Content-Type": "application/json" }
  if (options?.accessToken) {
    headers["Authorization"] = `Bearer ${options.accessToken}`
  }
  return fetch(`${BASE}/invoke/${agentId}`, {
    method: "POST",
    headers,
    body: JSON.stringify({
      message,
      protocol: options?.protocol,
      metadata: options?.metadata,
      stream: false,
    }),
  }).then(async (res) => {
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: res.statusText }))
      throw new Error(body.error || `API error: ${res.status}`)
    }
    return res.json() as Promise<InvokeResponse>
  })
}

export function invokeAgentStream(
  agentId: string,
  message: string,
  onChunk: (text: string) => void,
  options?: { protocol?: string; metadata?: Record<string, string>; accessToken?: string }
): { cancel: () => void; done: Promise<void> } {
  const controller = new AbortController()
  const headers: Record<string, string> = { "Content-Type": "application/json" }
  if (options?.accessToken) {
    headers["Authorization"] = `Bearer ${options.accessToken}`
  }

  const done = fetch(`${BASE}/invoke/${agentId}`, {
    method: "POST",
    headers,
    body: JSON.stringify({
      message,
      protocol: options?.protocol,
      metadata: options?.metadata,
      stream: true,
    }),
    signal: controller.signal,
  }).then(async (res) => {
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: res.statusText }))
      throw new Error(body.error || `API error: ${res.status}`)
    }
    const reader = res.body?.getReader()
    if (!reader) return
    const decoder = new TextDecoder()
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      const text = decoder.decode(value, { stream: true })
      // Parse SSE lines
      for (const line of text.split("\n")) {
        if (line.startsWith("data: ")) {
          const data = line.slice(6)
          if (data === "[DONE]") return
          try {
            const parsed = JSON.parse(data)
            if (parsed.chunk) onChunk(parsed.chunk)
          } catch {
            onChunk(data)
          }
        }
      }
    }
  })

  return { cancel: () => controller.abort(), done }
}
