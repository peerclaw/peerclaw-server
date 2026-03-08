import { useState, useEffect, useCallback } from "react"
import { fetchWithAuth } from "@/api/client"
import { useAuth } from "@/hooks/use-auth"
import { generateClaimToken, listClaimTokens } from "@/api/claim"
import type {
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

export interface PublishAgentData {
  name: string
  description: string
  version: string
  capabilities: string[]
  protocols: string[]
  endpoint_url: string
  auth_type: string
  auth_config?: Record<string, string>
  tags: string[]
}

export interface Invocation {
  id: string
  agent_id: string
  agent_name: string
  caller_id: string
  status: "success" | "error" | "timeout"
  duration_ms: number
  created_at: string
  error_message?: string
}

export interface InvocationListResponse {
  invocations: Invocation[]
  next_page_token?: string
  total_count: number
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
  page?: number,
  pageSize = 20
): UseQueryResult<InvocationListResponse> {
  const query = new URLSearchParams()
  if (page) query.set("page", String(page))
  query.set("page_size", String(pageSize))
  const qs = query.toString()
  return useProviderQuery<InvocationListResponse>(
    `/provider/invocations${qs ? `?${qs}` : ""}`
  )
}

// ----- Mutations -----

export function useProviderMutations() {
  const { accessToken } = useAuth()

  const publishAgent = useCallback(
    async (data: PublishAgentData): Promise<ProviderAgent> => {
      if (!accessToken) throw new Error("Not authenticated")
      return fetchWithAuth<ProviderAgent>("/provider/agents", accessToken, {
        method: "POST",
        body: JSON.stringify(data),
      })
    },
    [accessToken]
  )

  const updateAgent = useCallback(
    async (id: string, data: Partial<PublishAgentData>): Promise<ProviderAgent> => {
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

  return { publishAgent, updateAgent, deleteAgent }
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
