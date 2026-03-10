const BASE = "/api/v1"

export interface AuthTokens {
  access_token: string
  refresh_token: string
  expires_in: number
}

export interface AuthUser {
  id: string
  email: string
  display_name: string
  description: string
  role: string
  created_at: string
  updated_at: string
}

export interface AuthResponse {
  user: AuthUser
  tokens: AuthTokens
}

export interface APIKeyResponse {
  api_key: {
    id: string
    user_id: string
    name: string
    prefix: string
    created_at: string
    last_used?: string
    expires_at?: string
    revoked: boolean
  }
  key: string
}

export interface APIKey {
  id: string
  user_id: string
  name: string
  prefix: string
  created_at: string
  last_used?: string
  expires_at?: string
  revoked: boolean
}

async function authFetch<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const { headers, ...rest } = options
  const res = await fetch(`${BASE}${path}`, {
    ...rest,
    headers: { "Content-Type": "application/json", ...headers },
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error || `API error: ${res.status}`)
  }
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

export function register(
  email: string,
  password: string,
  displayName?: string
): Promise<AuthResponse> {
  return authFetch<AuthResponse>("/auth/register", {
    method: "POST",
    body: JSON.stringify({ email, password, display_name: displayName }),
  })
}

export function login(
  email: string,
  password: string
): Promise<AuthResponse> {
  return authFetch<AuthResponse>("/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  })
}

export function refreshToken(refresh_token: string): Promise<AuthTokens> {
  return authFetch<AuthTokens>("/auth/refresh", {
    method: "POST",
    body: JSON.stringify({ refresh_token }),
  })
}

export function logout(refresh_token: string): Promise<void> {
  return authFetch<void>("/auth/logout", {
    method: "POST",
    body: JSON.stringify({ refresh_token }),
  })
}

export function getMe(accessToken: string): Promise<AuthUser> {
  return authFetch<AuthUser>("/auth/me", {
    headers: { Authorization: `Bearer ${accessToken}` },
  })
}

export function updateMe(
  accessToken: string,
  data: { display_name?: string; email?: string; description?: string }
): Promise<AuthUser> {
  return authFetch<AuthUser>("/auth/me", {
    method: "PUT",
    headers: { Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(data),
  })
}

export function changePassword(
  accessToken: string,
  currentPassword: string,
  newPassword: string
): Promise<void> {
  return authFetch<void>("/auth/password", {
    method: "POST",
    headers: { Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
  })
}

export function createAPIKey(
  accessToken: string,
  name: string
): Promise<APIKeyResponse> {
  return authFetch<APIKeyResponse>("/auth/api-keys", {
    method: "POST",
    headers: { Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify({ name }),
  })
}

export function listAPIKeys(
  accessToken: string
): Promise<{ api_keys: APIKey[] }> {
  return authFetch<{ api_keys: APIKey[] }>("/auth/api-keys", {
    headers: { Authorization: `Bearer ${accessToken}` },
  })
}

export function revokeAPIKey(
  accessToken: string,
  keyId: string
): Promise<void> {
  return authFetch<void>(`/auth/api-keys/${keyId}`, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${accessToken}` },
  })
}
