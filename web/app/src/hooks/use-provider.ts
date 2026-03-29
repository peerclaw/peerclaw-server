import { useCallback } from "react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { fetchWithAuth, fetchSDKVersion, type SDKVersionResponse } from "@/api/client"
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
  sdk_version?: string
  metadata?: Record<string, string>
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

// ----- Generic query helper -----

function useProviderQuery<T>(path: string, skip = false) {
  const { accessToken } = useAuth()
  return useQuery<T>({
    queryKey: ["provider", path],
    queryFn: () => fetchWithAuth<T>(path, accessToken!),
    enabled: !!accessToken && !skip,
  })
}

// ----- Hooks -----

export function useProviderAgents(
  search?: string,
  status?: string,
  sort?: string,
  limit = 50,
  offset = 0
) {
  const params = new URLSearchParams()
  if (search) params.set("search", search)
  if (status) params.set("status", status)
  if (sort) params.set("sort", sort)
  params.set("limit", String(limit))
  params.set("offset", String(offset))
  const qs = params.toString()
  return useProviderQuery<{ agents: ProviderAgent[]; total_count: number }>(
    `/provider/agents?${qs}`
  )
}

export function useProviderAgent(id: string | undefined) {
  return useProviderQuery<ProviderAgent>(
    `/provider/agents/${id}`,
    !id
  )
}

export function useProviderDashboard(since?: string) {
  const qs = since ? `?since=${encodeURIComponent(since)}` : ""
  return useProviderQuery<ProviderDashboardData>(`/provider/dashboard${qs}`)
}

export function useAgentAnalytics(agentId: string | undefined) {
  return useProviderQuery<AgentAnalytics>(
    `/provider/agents/${agentId}/analytics`,
    !agentId
  )
}

export function useProviderInvocations(page = 1, pageSize = 20) {
  const query = new URLSearchParams()
  query.set("limit", String(pageSize))
  query.set("offset", String((page - 1) * pageSize))
  const qs = query.toString()
  return useProviderQuery<InvocationListResponse>(`/invocations?${qs}`)
}

// ----- Mutations -----

export function useProviderMutations() {
  const { accessToken } = useAuth()
  const qc = useQueryClient()

  const registerAgent = useCallback(
    async (data: RegisterAgentData): Promise<ProviderAgent> => {
      if (!accessToken) throw new Error("Not authenticated")
      const result = await fetchWithAuth<ProviderAgent>("/provider/agents", accessToken, {
        method: "POST",
        body: JSON.stringify(data),
      })
      qc.invalidateQueries({ queryKey: ["provider"] })
      return result
    },
    [accessToken, qc]
  )

  const updateAgent = useCallback(
    async (id: string, data: Partial<RegisterAgentData>): Promise<ProviderAgent> => {
      if (!accessToken) throw new Error("Not authenticated")
      const result = await fetchWithAuth<ProviderAgent>(`/provider/agents/${id}`, accessToken, {
        method: "PUT",
        body: JSON.stringify(data),
      })
      qc.invalidateQueries({ queryKey: ["provider"] })
      return result
    },
    [accessToken, qc]
  )

  const deleteAgent = useCallback(
    async (id: string): Promise<void> => {
      if (!accessToken) throw new Error("Not authenticated")
      await fetchWithAuth<void>(`/provider/agents/${id}`, accessToken, {
        method: "DELETE",
      })
      qc.invalidateQueries({ queryKey: ["provider"] })
    },
    [accessToken, qc]
  )

  return { registerAgent, updateAgent, deleteAgent }
}

// ----- Agent Contacts Hooks -----

export function useAgentContacts(agentId: string | undefined) {
  return useProviderQuery<{ contacts: AgentContact[] }>(
    `/provider/agents/${agentId}/contacts`,
    !agentId
  )
}

export function useAgentContactMutations(agentId: string | undefined) {
  const { accessToken } = useAuth()
  const qc = useQueryClient()

  const addContact = useCallback(
    async (contactAgentId: string, alias = ""): Promise<AgentContact> => {
      if (!accessToken) throw new Error("Not authenticated")
      if (!agentId) throw new Error("Agent ID required")
      const result = await fetchWithAuth<AgentContact>(
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
      qc.invalidateQueries({ queryKey: ["provider", `/provider/agents/${agentId}/contacts`] })
      return result
    },
    [accessToken, agentId, qc]
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
      qc.invalidateQueries({ queryKey: ["provider", `/provider/agents/${agentId}/contacts`] })
    },
    [accessToken, agentId, qc]
  )

  return { addContact, removeContact }
}

// ----- Claim Token Hooks -----

export function useClaimTokens() {
  const { accessToken } = useAuth()
  return useQuery<{ tokens: ClaimToken[] }>({
    queryKey: ["claimTokens"],
    queryFn: () => listClaimTokens(accessToken!),
    enabled: !!accessToken,
  })
}

export function useGenerateClaimToken() {
  const { accessToken } = useAuth()
  const qc = useQueryClient()

  const generate = useCallback(
    async (
      params: GenerateClaimTokenRequest
    ): Promise<GenerateClaimTokenResponse> => {
      if (!accessToken) throw new Error("Not authenticated")
      const result = await generateClaimToken(accessToken, params)
      qc.invalidateQueries({ queryKey: ["claimTokens"] })
      return result
    },
    [accessToken, qc]
  )

  return { generate }
}

// ----- Access Request Hooks -----

export function useAgentAccessRequests(agentId: string | undefined) {
  return useProviderQuery<{ requests: AccessRequest[] }>(
    `/provider/agents/${agentId}/access-requests`,
    !agentId
  )
}

export function useAccessRequestMutations(agentId: string | undefined) {
  const { accessToken } = useAuth()
  const qc = useQueryClient()

  const invalidateRequests = () =>
    qc.invalidateQueries({
      queryKey: ["provider", `/provider/agents/${agentId}/access-requests`],
    })

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
      invalidateRequests()
    },
    [accessToken, agentId, qc]
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
      invalidateRequests()
    },
    [accessToken, agentId, qc]
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
      invalidateRequests()
    },
    [accessToken, agentId, qc]
  )

  return { approve, reject, revoke }
}

// ----- SDK Version -----

export function useSDKVersion() {
  const { accessToken } = useAuth()
  return useQuery<SDKVersionResponse>({
    queryKey: ["sdkVersion"],
    queryFn: () => fetchSDKVersion(accessToken!),
    enabled: !!accessToken,
  })
}
