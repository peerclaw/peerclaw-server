import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { useAuth } from "@/hooks/use-auth"
import * as adminAPI from "@/api/admin"
import type {
  AdminDashboardStats,
  AdminUserListResponse,
  AdminAgentDetail,
  AdminReportListResponse,
  GlobalAnalytics,
  AdminInvocationListResponse,
  InvocationRecord,
  AgentListResponse,
  Category,
  AdminAuditListResponse,
} from "@/api/types"

// ----- Dashboard -----

export function useAdminDashboard() {
  const { accessToken } = useAuth()
  return useQuery<AdminDashboardStats>({
    queryKey: ["admin", "dashboard"],
    queryFn: () => adminAPI.fetchAdminDashboard(accessToken!),
    enabled: !!accessToken,
  })
}

// ----- Users -----

export function useAdminUsers(
  search?: string,
  role?: string,
  sort?: string,
  limit = 50,
  offset = 0
) {
  const { accessToken } = useAuth()
  return useQuery<AdminUserListResponse>({
    queryKey: ["admin", "users", { search, role, sort, limit, offset }],
    queryFn: () =>
      adminAPI.fetchAdminUsers(accessToken!, { search, role, sort, limit, offset }),
    enabled: !!accessToken,
  })
}

// ----- Agents -----

export function useAdminAgents(
  search?: string,
  protocol?: string,
  status?: string,
  sort?: string,
  limit = 50,
  offset = 0
) {
  const { accessToken } = useAuth()
  return useQuery<AgentListResponse>({
    queryKey: ["admin", "agents", { search, protocol, status, sort, limit, offset }],
    queryFn: () =>
      adminAPI.fetchAdminAgents(accessToken!, { search, protocol, status, sort, limit, offset }),
    enabled: !!accessToken,
  })
}

export function useAdminAgent(id: string | undefined) {
  const { accessToken } = useAuth()
  return useQuery<AdminAgentDetail>({
    queryKey: ["admin", "agent", id],
    queryFn: () => adminAPI.fetchAdminAgent(accessToken!, id!),
    enabled: !!accessToken && !!id,
  })
}

// ----- Reports -----

export function useAdminReports(
  status?: string,
  search?: string,
  sort?: string,
  limit = 50,
  offset = 0
) {
  const { accessToken } = useAuth()
  return useQuery<AdminReportListResponse>({
    queryKey: ["admin", "reports", { status, search, sort, limit, offset }],
    queryFn: () =>
      adminAPI.fetchAdminReports(accessToken!, { status, search, sort, limit, offset }),
    enabled: !!accessToken,
  })
}

// ----- Analytics -----

export function useAdminAnalytics(since?: string, bucketMinutes?: number) {
  const { accessToken } = useAuth()
  return useQuery<GlobalAnalytics>({
    queryKey: ["admin", "analytics", { since, bucketMinutes }],
    queryFn: () =>
      adminAPI.fetchAdminAnalytics(accessToken!, { since, bucket_minutes: bucketMinutes }),
    enabled: !!accessToken,
  })
}

// ----- Invocations -----

export function useAdminInvocations(
  agentId?: string,
  userId?: string,
  sort?: string,
  since?: string,
  limit = 50,
  offset = 0
) {
  const { accessToken } = useAuth()
  return useQuery<AdminInvocationListResponse>({
    queryKey: ["admin", "invocations", { agentId, userId, sort, since, limit, offset }],
    queryFn: () =>
      adminAPI.fetchAdminInvocations(accessToken!, {
        agent_id: agentId,
        user_id: userId,
        sort,
        since,
        limit,
        offset,
      }),
    enabled: !!accessToken,
  })
}

export function useAdminInvocation(id: string | undefined) {
  const { accessToken } = useAuth()
  return useQuery<InvocationRecord>({
    queryKey: ["admin", "invocation", id],
    queryFn: () => adminAPI.fetchAdminInvocation(accessToken!, id!),
    enabled: !!accessToken && !!id,
  })
}

// ----- Audit Log -----

export function useAdminAudit(
  adminUserID?: string,
  action?: string,
  targetType?: string,
  since?: string,
  limit = 50,
  offset = 0
) {
  const { accessToken } = useAuth()
  return useQuery<AdminAuditListResponse>({
    queryKey: ["admin", "audit", { adminUserID, action, targetType, since, limit, offset }],
    queryFn: () =>
      adminAPI.fetchAdminAudit(accessToken!, {
        admin_user_id: adminUserID,
        action,
        target_type: targetType,
        since,
        limit,
        offset,
      }),
    enabled: !!accessToken,
  })
}

// ----- Categories (public endpoint + admin mutations) -----

export function useAdminCategories() {
  return useQuery<{ categories: Category[] }>({
    queryKey: ["categories"],
    queryFn: async () => {
      const res = await fetch("/api/v1/categories")
      if (!res.ok) throw new Error("Failed to fetch categories")
      return res.json() as Promise<{ categories: Category[] }>
    },
  })
}

// ----- Mutations -----

export function useAdminMutations() {
  const { accessToken } = useAuth()
  const qc = useQueryClient()

  const deleteUser = useMutation({
    mutationFn: (id: string) => adminAPI.deleteAdminUser(accessToken!, id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "users"] }),
  })

  const updateUserRole = useMutation({
    mutationFn: ({ id, role }: { id: string; role: string }) =>
      adminAPI.updateAdminUserRole(accessToken!, id, role),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "users"] }),
  })

  const deleteAgent = useMutation({
    mutationFn: (id: string) => adminAPI.deleteAdminAgent(accessToken!, id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "agents"] }),
  })

  const verifyAgent = useMutation({
    mutationFn: (id: string) => adminAPI.verifyAdminAgent(accessToken!, id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "agents"] })
      qc.invalidateQueries({ queryKey: ["admin", "agent"] })
    },
  })

  const unverifyAgent = useMutation({
    mutationFn: (id: string) => adminAPI.unverifyAdminAgent(accessToken!, id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "agents"] })
      qc.invalidateQueries({ queryKey: ["admin", "agent"] })
    },
  })

  const updateReport = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      adminAPI.updateAdminReport(accessToken!, id, status),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "reports"] }),
  })

  const deleteReport = useMutation({
    mutationFn: (id: string) => adminAPI.deleteAdminReport(accessToken!, id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "reports"] }),
  })

  const createCategory = useMutation({
    mutationFn: (data: Omit<Category, "id"> & { id?: string }) =>
      adminAPI.createAdminCategory(accessToken!, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["categories"] }),
  })

  const updateCategory = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Category> }) =>
      adminAPI.updateAdminCategory(accessToken!, id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["categories"] }),
  })

  const deleteCategory = useMutation({
    mutationFn: (id: string) => adminAPI.deleteAdminCategory(accessToken!, id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["categories"] }),
  })

  const bulkAgentsAction = useMutation({
    mutationFn: ({ action, ids }: { action: string; ids: string[] }) =>
      adminAPI.bulkAgents(accessToken!, action, ids),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "agents"] }),
  })

  const bulkReportsAction = useMutation({
    mutationFn: ({ action, ids }: { action: string; ids: string[] }) =>
      adminAPI.bulkReports(accessToken!, action, ids),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "reports"] }),
  })

  const bulkUsersAction = useMutation({
    mutationFn: ({ action, ids, role }: { action: string; ids: string[]; role?: string }) =>
      adminAPI.bulkUsers(accessToken!, action, ids, role),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "users"] }),
  })

  return {
    deleteUser,
    updateUserRole,
    deleteAgent,
    verifyAgent,
    unverifyAgent,
    updateReport,
    deleteReport,
    createCategory,
    updateCategory,
    deleteCategory,
    bulkAgentsAction,
    bulkReportsAction,
    bulkUsersAction,
  }
}
