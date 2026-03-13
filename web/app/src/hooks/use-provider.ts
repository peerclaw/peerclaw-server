import { useState, useEffect, useCallback } from "react"
import { fetchWithAuth } from "@/api/client"
import { useAuth } from "@/hooks/use-auth"
import { generateClaimToken, listClaimTokens } from "@/api/claim"
import type {
  AccessRequest,
  AgentContact,
  ClaimToken,
  GenerateClaimTokenRequest,
  GenerateClaimTokenResponse,
} from "@/api/types"

// ----- Types -----

export interface ProviderAgent {
  id: string
  name: string
  description: string
  version: string
  capabilities: string[]
  protocols: string[]
  status: "online" | "offline" | "degraded"
  endpoint_url: string
  auth_type: string
  tags: string[]
  total_calls: number
  success_rate: number
  avg_latency_ms: number
  created_at: string
  updated_at: string
  playground_enabled?: boolean
  visibility?: string
  public_key?: string
  skills?: Array<{ name: string; description?: string }>
  categories?: string[]
  verified?: boolean
  verified_at?: string
  registered_at?: string
  last_heartbeat?: string
  reputation_score?: number
  review_summary?: {
    average_rating: number
    total_reviews: number
    distribution: number[]
  }
}

export interface ProviderDashboardData {
  total_agents: number
  total_calls: number
  success_rate: number
  avg_latency_ms: number
  agents: ProviderAgent[]
}

export interface TimeSeriesPoint {
  timestamp: string
  count: number
}

export interface AgentAnalytics {
  total_calls: number
  success_rate: number
  avg_latency_ms: number
  time_series: TimeSeriesPoint[]
}

export interface RegisterAgentData {
  name: string
  description: string
  version: string
  capabilities: string[]
  protocols: string[]
  endpoint_url: string
  auth_type: string
  auth_config?: Record<string, string>
  tags: string[]
  playground_enabled?: boolean
  visibility?: string
}

export interface Invocation {
  id: string
  agent_id: string
  user_id?: string
  protocol: string
  status_code: number
  duration_ms: number
  error?: string
  created_at: string
}

export interface InvocationListResponse {
  invocations: Invocation[]
  total: number
}

// ----- Hook helpers -----

interface UseQueryResult<T> {
  data: T | null
  loading: boolean
  error: string | null
  refetch: () => void
}

function useProviderQuery<T>(path: string, skip = false): UseQueryResult<T> {
  const { accessToken } = useAuth()
  const [data, setData] = useState<T | null>(null)
  const [loading, setLoading] = useState(!skip)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!accessToken) return
    try {
      setLoading(true)
      setError(null)
      const result = await fetchWithAuth<T>(path, accessToken)
      setData(result)
    } catch (e) {
      setError(e instanceof Error ? e.message : "Request failed")
    } finally {
      setLoading(false)
    }
  }, [path, accessToken])

  useEffect(() => {
    if (!skip) {
      load()
    }
  }, [load, skip])

  return { data, loading, error, refetch: load }
}

// ----- Hooks -----

export function useProviderAgents(): UseQueryResult<{ agents: ProviderAgent[] }> {
  return useProviderQuery<{ agents: ProviderAgent[] }>("/provider/agents")
}

export function useProviderAgent(id: string | undefined): UseQueryResult<ProviderAgent> {
  return useProviderQuery<ProviderAgent>(
    `/provider/agents/${id}`,
    !id
  )
}

export function useProviderDashboard(): UseQueryResult<ProviderDashboardData> {
  return useProviderQuery<ProviderDashboardData>("/provider/dashboard")
}

export function useAgentAnalytics(agentId: string | undefined): UseQueryResult<AgentAnalytics> {
  return useProviderQuery<AgentAnalytics>(
    `/provider/agents/${agentId}/analytics`,
    !agentId
  )
}

export function useProviderInvocations(
  page = 1,
  pageSize = 20
): UseQueryResult<InvocationListResponse> {
  const query = new URLSearchParams()
  query.set("limit", String(pageSize))
  query.set("offset", String((page - 1) * pageSize))
  const qs = query.toString()
  return useProviderQuery<InvocationListResponse>(
    `/invocations?${qs}`
  )
}

// ----- Mutations -----

export function useProviderMutations() {
  const { accessToken } = useAuth()

  const registerAgent = useCallback(
    async (data: RegisterAgentData): Promise<ProviderAgent> => {
      if (!accessToken) throw new Error("Not authenticated")
      return fetchWithAuth<ProviderAgent>("/provider/agents", accessToken, {
        method: "POST",
        body: JSON.stringify(data),
      })
    },
    [accessToken]
  )

  const updateAgent = useCallback(
    async (id: string, data: Partial<RegisterAgentData>): Promise<ProviderAgent> => {
      if (!accessToken) throw new Error("Not authenticated")
      return fetchWithAuth<ProviderAgent>(`/provider/agents/${id}`, accessToken, {
        method: "PUT",
        body: JSON.stringify(data),
      })
    },
    [accessToken]
  )

  const deleteAgent = useCallback(
    async (id: string): Promise<void> => {
      if (!accessToken) throw new Error("Not authenticated")
      await fetchWithAuth<void>(`/provider/agents/${id}`, accessToken, {
        method: "DELETE",
      })
    },
    [accessToken]
  )

  return { registerAgent, updateAgent, deleteAgent }
}

// ----- Agent Contacts Hooks -----

export function useAgentContacts(
  agentId: string | undefined
): UseQueryResult<{ contacts: AgentContact[] }> {
  return useProviderQuery<{ contacts: AgentContact[] }>(
    `/provider/agents/${agentId}/contacts`,
    !agentId
  )
}

export function useAgentContactMutations(agentId: string | undefined) {
  const { accessToken } = useAuth()

  const addContact = useCallback(
    async (contactAgentId: string, alias = ""): Promise<AgentContact> => {
      if (!accessToken) throw new Error("Not authenticated")
      if (!agentId) throw new Error("Agent ID required")
      return fetchWithAuth<AgentContact>(
        `/provider/agents/${agentId}/contacts`,
        accessToken,
        {
          method: "POST",
          body: JSON.stringify({
            contact_agent_id: contactAgentId,
            alias,
          }),
        }
      )
    },
    [accessToken, agentId]
  )

  const removeContact = useCallback(
    async (contactAgentId: string): Promise<void> => {
      if (!accessToken) throw new Error("Not authenticated")
      if (!agentId) throw new Error("Agent ID required")
      await fetchWithAuth<void>(
        `/provider/agents/${agentId}/contacts/${contactAgentId}`,
        accessToken,
        { method: "DELETE" }
      )
    },
    [accessToken, agentId]
  )

  return { addContact, removeContact }
}

// ----- Claim Token Hooks -----

export function useClaimTokens(): UseQueryResult<{ tokens: ClaimToken[] }> {
  const { accessToken } = useAuth()
  const [data, setData] = useState<{ tokens: ClaimToken[] } | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!accessToken) return
    try {
      setLoading(true)
      setError(null)
      const result = await listClaimTokens(accessToken)
      setData(result)
    } catch (e) {
      setError(e instanceof Error ? e.message : "Request failed")
    } finally {
      setLoading(false)
    }
  }, [accessToken])

  useEffect(() => {
    load()
  }, [load])

  return { data, loading, error, refetch: load }
}

export function useGenerateClaimToken() {
  const { accessToken } = useAuth()

  const generate = useCallback(
    async (
      params: GenerateClaimTokenRequest
    ): Promise<GenerateClaimTokenResponse> => {
      if (!accessToken) throw new Error("Not authenticated")
      return generateClaimToken(accessToken, params)
    },
    [accessToken]
  )

  return { generate }
}

// ----- Access Request Hooks -----

export function useAgentAccessRequests(
  agentId: string | undefined
): UseQueryResult<{ requests: AccessRequest[] }> {
  return useProviderQuery<{ requests: AccessRequest[] }>(
    `/provider/agents/${agentId}/access-requests`,
    !agentId
  )
}

export function useAccessRequestMutations(agentId: string | undefined) {
  const { accessToken } = useAuth()

  const approve = useCallback(
    async (requestId: string, expiresAt?: string): Promise<void> => {
      if (!accessToken) throw new Error("Not authenticated")
      if (!agentId) throw new Error("Agent ID required")
      await fetchWithAuth<{ status: string }>(
        `/provider/agents/${agentId}/access-requests/${requestId}`,
        accessToken,
        {
          method: "PUT",
          body: JSON.stringify({
            action: "approve",
            ...(expiresAt ? { expires_at: expiresAt } : {}),
          }),
        }
      )
    },
    [accessToken, agentId]
  )

  const reject = useCallback(
    async (requestId: string, reason = ""): Promise<void> => {
      if (!accessToken) throw new Error("Not authenticated")
      if (!agentId) throw new Error("Agent ID required")
      await fetchWithAuth<{ status: string }>(
        `/provider/agents/${agentId}/access-requests/${requestId}`,
        accessToken,
        {
          method: "PUT",
          body: JSON.stringify({ action: "reject", reject_reason: reason }),
        }
      )
    },
    [accessToken, agentId]
  )

  const revoke = useCallback(
    async (requestId: string): Promise<void> => {
      if (!accessToken) throw new Error("Not authenticated")
      if (!agentId) throw new Error("Agent ID required")
      await fetchWithAuth<void>(
        `/provider/agents/${agentId}/access-requests/${requestId}`,
        accessToken,
        { method: "DELETE" }
      )
    },
    [accessToken, agentId]
  )

  return { approve, reject, revoke }
}
