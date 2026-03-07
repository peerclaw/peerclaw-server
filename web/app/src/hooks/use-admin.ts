import { useState, useEffect, useCallback } from "react"
import { useAuth } from "@/hooks/use-auth"
import * as adminAPI from "@/api/admin"
import type {
  AdminDashboardStats,
  AdminUserListResponse,
  AdminAgentDetail,
  AdminReportListResponse,
  GlobalAnalytics,
  AdminInvocationListResponse,
  AgentListResponse,
  Category,
} from "@/api/types"

// ----- Generic query hook -----

interface UseQueryResult<T> {
  data: T | null
  loading: boolean
  error: string | null
  refetch: () => void
}

function useAdminQuery<T>(
  fetcher: (token: string) => Promise<T>,
  skip = false
): UseQueryResult<T> {
  const { accessToken } = useAuth()
  const [data, setData] = useState<T | null>(null)
  const [loading, setLoading] = useState(!skip)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!accessToken) return
    try {
      setLoading(true)
      setError(null)
      const result = await fetcher(accessToken)
      setData(result)
    } catch (e) {
      setError(e instanceof Error ? e.message : "Request failed")
    } finally {
      setLoading(false)
    }
  }, [accessToken, fetcher])

  useEffect(() => {
    if (!skip) {
      load()
    }
  }, [load, skip])

  return { data, loading, error, refetch: load }
}

// ----- Dashboard -----

export function useAdminDashboard(): UseQueryResult<AdminDashboardStats> {
  const fetcher = useCallback(
    (token: string) => adminAPI.fetchAdminDashboard(token),
    []
  )
  return useAdminQuery(fetcher)
}

// ----- Users -----

export function useAdminUsers(
  search?: string,
  role?: string,
  limit = 50,
  offset = 0
): UseQueryResult<AdminUserListResponse> {
  const fetcher = useCallback(
    (token: string) => adminAPI.fetchAdminUsers(token, { search, role, limit, offset }),
    [search, role, limit, offset]
  )
  return useAdminQuery(fetcher)
}

// ----- Agents -----

export function useAdminAgents(
  search?: string,
  protocol?: string,
  status?: string
): UseQueryResult<AgentListResponse> {
  const fetcher = useCallback(
    (token: string) => adminAPI.fetchAdminAgents(token, { search, protocol, status }),
    [search, protocol, status]
  )
  return useAdminQuery(fetcher)
}

export function useAdminAgent(id: string | undefined): UseQueryResult<AdminAgentDetail> {
  const fetcher = useCallback(
    (token: string) => adminAPI.fetchAdminAgent(token, id!),
    [id]
  )
  return useAdminQuery(fetcher, !id)
}

// ----- Reports -----

export function useAdminReports(
  status?: string,
  limit = 50,
  offset = 0
): UseQueryResult<AdminReportListResponse> {
  const fetcher = useCallback(
    (token: string) => adminAPI.fetchAdminReports(token, { status, limit, offset }),
    [status, limit, offset]
  )
  return useAdminQuery(fetcher)
}

// ----- Analytics -----

export function useAdminAnalytics(
  since?: string,
  bucketMinutes?: number
): UseQueryResult<GlobalAnalytics> {
  const fetcher = useCallback(
    (token: string) =>
      adminAPI.fetchAdminAnalytics(token, { since, bucket_minutes: bucketMinutes }),
    [since, bucketMinutes]
  )
  return useAdminQuery(fetcher)
}

// ----- Invocations -----

export function useAdminInvocations(
  agentId?: string,
  userId?: string,
  limit = 50,
  offset = 0
): UseQueryResult<AdminInvocationListResponse> {
  const fetcher = useCallback(
    (token: string) =>
      adminAPI.fetchAdminInvocations(token, {
        agent_id: agentId,
        user_id: userId,
        limit,
        offset,
      }),
    [agentId, userId, limit, offset]
  )
  return useAdminQuery(fetcher)
}

// ----- Categories (uses existing public endpoint + admin mutations) -----

export function useAdminCategories(): UseQueryResult<{ categories: Category[] }> {
  const [data, setData] = useState<{ categories: Category[] } | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const res = await fetch("/api/v1/categories")
      if (!res.ok) throw new Error("Failed to fetch categories")
      const result = await res.json()
      setData(result)
    } catch (e) {
      setError(e instanceof Error ? e.message : "Request failed")
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  return { data, loading, error, refetch: load }
}

// ----- Mutations -----

export function useAdminMutations() {
  const { accessToken } = useAuth()

  const updateUserRole = useCallback(
    async (id: string, role: string) => {
      if (!accessToken) throw new Error("Not authenticated")
      return adminAPI.updateAdminUserRole(accessToken, id, role)
    },
    [accessToken]
  )

  const deleteUser = useCallback(
    async (id: string) => {
      if (!accessToken) throw new Error("Not authenticated")
      return adminAPI.deleteAdminUser(accessToken, id)
    },
    [accessToken]
  )

  const deleteAgent = useCallback(
    async (id: string) => {
      if (!accessToken) throw new Error("Not authenticated")
      return adminAPI.deleteAdminAgent(accessToken, id)
    },
    [accessToken]
  )

  const verifyAgent = useCallback(
    async (id: string) => {
      if (!accessToken) throw new Error("Not authenticated")
      return adminAPI.verifyAdminAgent(accessToken, id)
    },
    [accessToken]
  )

  const unverifyAgent = useCallback(
    async (id: string) => {
      if (!accessToken) throw new Error("Not authenticated")
      return adminAPI.unverifyAdminAgent(accessToken, id)
    },
    [accessToken]
  )

  const updateReport = useCallback(
    async (id: string, status: string) => {
      if (!accessToken) throw new Error("Not authenticated")
      return adminAPI.updateAdminReport(accessToken, id, status)
    },
    [accessToken]
  )

  const deleteReport = useCallback(
    async (id: string) => {
      if (!accessToken) throw new Error("Not authenticated")
      return adminAPI.deleteAdminReport(accessToken, id)
    },
    [accessToken]
  )

  const createCategory = useCallback(
    async (data: Omit<Category, "id"> & { id?: string }) => {
      if (!accessToken) throw new Error("Not authenticated")
      return adminAPI.createAdminCategory(accessToken, data)
    },
    [accessToken]
  )

  const updateCategory = useCallback(
    async (id: string, data: Partial<Category>) => {
      if (!accessToken) throw new Error("Not authenticated")
      return adminAPI.updateAdminCategory(accessToken, id, data)
    },
    [accessToken]
  )

  const deleteCategory = useCallback(
    async (id: string) => {
      if (!accessToken) throw new Error("Not authenticated")
      return adminAPI.deleteAdminCategory(accessToken, id)
    },
    [accessToken]
  )

  return {
    updateUserRole,
    deleteUser,
    deleteAgent,
    verifyAgent,
    unverifyAgent,
    updateReport,
    deleteReport,
    createCategory,
    updateCategory,
    deleteCategory,
  }
}
