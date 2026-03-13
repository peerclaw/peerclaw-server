import { fetchWithAuth } from "./client"
import type {
  AdminDashboardStats,
  AdminUserListResponse,
  AdminUser,
  AdminAgentDetail,
  AdminReportListResponse,
  AbuseReport,
  Category,
  GlobalAnalytics,
  AdminInvocationListResponse,
  AgentListResponse,
} from "./types"

// Dashboard
export function fetchAdminDashboard(token: string): Promise<AdminDashboardStats> {
  return fetchWithAuth<AdminDashboardStats>("/admin/dashboard", token)
}

// User Management
export function fetchAdminUsers(
  token: string,
  params?: { search?: string; role?: string; limit?: number; offset?: number }
): Promise<AdminUserListResponse> {
  const query = new URLSearchParams()
  if (params?.search) query.set("search", params.search)
  if (params?.role) query.set("role", params.role)
  if (params?.limit) query.set("limit", String(params.limit))
  if (params?.offset) query.set("offset", String(params.offset))
  const qs = query.toString()
  return fetchWithAuth<AdminUserListResponse>(`/admin/users${qs ? `?${qs}` : ""}`, token)
}

export function fetchAdminUser(token: string, id: string): Promise<AdminUser> {
  return fetchWithAuth<AdminUser>(`/admin/users/${id}`, token)
}

export function updateAdminUserRole(
  token: string,
  id: string,
  role: string
): Promise<AdminUser> {
  return fetchWithAuth<AdminUser>(`/admin/users/${id}/role`, token, {
    method: "PUT",
    body: JSON.stringify({ role }),
  })
}

export function deleteAdminUser(token: string, id: string): Promise<void> {
  return fetchWithAuth<void>(`/admin/users/${id}`, token, { method: "DELETE" })
}

// Agent Management
export function fetchAdminAgents(
  token: string,
  params?: { search?: string; protocol?: string; status?: string; limit?: number; offset?: number }
): Promise<AgentListResponse> {
  const query = new URLSearchParams()
  if (params?.search) query.set("search", params.search)
  if (params?.protocol) query.set("protocol", params.protocol)
  if (params?.status) query.set("status", params.status)
  if (params?.limit) query.set("limit", String(params.limit))
  if (params?.offset) query.set("offset", String(params.offset))
  const qs = query.toString()
  return fetchWithAuth<AgentListResponse>(`/admin/agents${qs ? `?${qs}` : ""}`, token)
}

export function fetchAdminAgent(token: string, id: string): Promise<AdminAgentDetail> {
  return fetchWithAuth<AdminAgentDetail>(`/admin/agents/${id}`, token)
}

export function deleteAdminAgent(token: string, id: string): Promise<void> {
  return fetchWithAuth<void>(`/admin/agents/${id}`, token, { method: "DELETE" })
}

export function verifyAdminAgent(token: string, id: string): Promise<{ status: string }> {
  return fetchWithAuth<{ status: string }>(`/admin/agents/${id}/verify`, token, {
    method: "POST",
  })
}

export function unverifyAdminAgent(token: string, id: string): Promise<{ status: string }> {
  return fetchWithAuth<{ status: string }>(`/admin/agents/${id}/verify`, token, {
    method: "DELETE",
  })
}

// Report Moderation
export function fetchAdminReports(
  token: string,
  params?: { status?: string; limit?: number; offset?: number }
): Promise<AdminReportListResponse> {
  const query = new URLSearchParams()
  if (params?.status) query.set("status", params.status)
  if (params?.limit) query.set("limit", String(params.limit))
  if (params?.offset) query.set("offset", String(params.offset))
  const qs = query.toString()
  return fetchWithAuth<AdminReportListResponse>(`/admin/reports${qs ? `?${qs}` : ""}`, token)
}

export function fetchAdminReport(token: string, id: string): Promise<AbuseReport> {
  return fetchWithAuth<AbuseReport>(`/admin/reports/${id}`, token)
}

export function updateAdminReport(
  token: string,
  id: string,
  status: string
): Promise<{ status: string }> {
  return fetchWithAuth<{ status: string }>(`/admin/reports/${id}`, token, {
    method: "PUT",
    body: JSON.stringify({ status }),
  })
}

export function deleteAdminReport(token: string, id: string): Promise<void> {
  return fetchWithAuth<void>(`/admin/reports/${id}`, token, { method: "DELETE" })
}

// Category Management
export function createAdminCategory(
  token: string,
  data: Omit<Category, "id"> & { id?: string }
): Promise<Category> {
  return fetchWithAuth<Category>("/admin/categories", token, {
    method: "POST",
    body: JSON.stringify(data),
  })
}

export function updateAdminCategory(
  token: string,
  id: string,
  data: Partial<Category>
): Promise<Category> {
  return fetchWithAuth<Category>(`/admin/categories/${id}`, token, {
    method: "PUT",
    body: JSON.stringify(data),
  })
}

export function deleteAdminCategory(token: string, id: string): Promise<void> {
  return fetchWithAuth<void>(`/admin/categories/${id}`, token, { method: "DELETE" })
}

// Global Analytics
export function fetchAdminAnalytics(
  token: string,
  params?: { since?: string; bucket_minutes?: number }
): Promise<GlobalAnalytics> {
  const query = new URLSearchParams()
  if (params?.since) query.set("since", params.since)
  if (params?.bucket_minutes) query.set("bucket_minutes", String(params.bucket_minutes))
  const qs = query.toString()
  return fetchWithAuth<GlobalAnalytics>(`/admin/analytics${qs ? `?${qs}` : ""}`, token)
}

// Invocation Log
export function fetchAdminInvocations(
  token: string,
  params?: { agent_id?: string; user_id?: string; limit?: number; offset?: number }
): Promise<AdminInvocationListResponse> {
  const query = new URLSearchParams()
  if (params?.agent_id) query.set("agent_id", params.agent_id)
  if (params?.user_id) query.set("user_id", params.user_id)
  if (params?.limit) query.set("limit", String(params.limit))
  if (params?.offset) query.set("offset", String(params.offset))
  const qs = query.toString()
  return fetchWithAuth<AdminInvocationListResponse>(
    `/admin/invocations${qs ? `?${qs}` : ""}`,
    token
  )
}
