import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  useMemo,
  type ReactNode,
} from "react"
import { createElement } from "react"
import type { AuthUser, AuthTokens } from "@/api/auth"
import * as authAPI from "@/api/auth"
import { setAuthRefreshHandler } from "@/api/client"

interface AuthContextValue {
  user: AuthUser | null
  accessToken: string | null
  loading: boolean
  pendingVerificationEmail: string | null
  login: (email: string, password: string) => Promise<void>
  register: (
    email: string,
    password: string,
    displayName?: string
  ) => Promise<void>
  logout: () => Promise<void>
  refreshAccessToken: () => Promise<string | null>
  updateProfile: (data: { display_name?: string; email?: string; description?: string }) => Promise<void>
  changePassword: (currentPassword: string, newPassword: string) => Promise<void>
  verifyEmail: (email: string, code: string) => Promise<void>
  resendVerification: (email: string) => Promise<void>
  clearPendingVerification: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

const TOKEN_KEY = "peerclaw_tokens"

function saveTokens(tokens: AuthTokens) {
  localStorage.setItem(TOKEN_KEY, JSON.stringify(tokens))
}

function loadTokens(): AuthTokens | null {
  const raw = localStorage.getItem(TOKEN_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as AuthTokens
  } catch {
    return null
  }
}

function clearTokens() {
  localStorage.removeItem(TOKEN_KEY)
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null)
  const [accessToken, setAccessToken] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [pendingVerificationEmail, setPendingVerificationEmail] = useState<string | null>(null)

  // Try to restore session on mount.
  useEffect(() => {
    const tokens = loadTokens()
    if (!tokens) {
      setLoading(false)
      return
    }

    setAccessToken(tokens.access_token)

    authAPI
      .getMe(tokens.access_token)
      .then((u) => setUser(u))
      .catch(() => {
        // Access token expired, try refresh.
        authAPI
          .refreshToken(tokens.refresh_token)
          .then((newTokens) => {
            saveTokens({
              ...newTokens,
              refresh_token:
                newTokens.refresh_token || tokens.refresh_token,
            })
            setAccessToken(newTokens.access_token)
            return authAPI.getMe(newTokens.access_token)
          })
          .then((u) => setUser(u))
          .catch(() => {
            clearTokens()
            setAccessToken(null)
          })
      })
      .finally(() => setLoading(false))
  }, [])

  // Auto-refresh timer.
  useEffect(() => {
    const tokens = loadTokens()
    if (!tokens || !accessToken) return

    // Refresh 1 minute before expiry.
    const refreshMs = (tokens.expires_in - 60) * 1000
    if (refreshMs <= 0) return

    const timer = setTimeout(() => {
      authAPI
        .refreshToken(tokens.refresh_token)
        .then((newTokens) => {
          saveTokens({
            ...newTokens,
            refresh_token:
              newTokens.refresh_token || tokens.refresh_token,
          })
          setAccessToken(newTokens.access_token)
        })
        .catch(() => {
          clearTokens()
          setAccessToken(null)
          setUser(null)
        })
    }, refreshMs)

    return () => clearTimeout(timer)
  }, [accessToken])

  const loginFn = useCallback(async (email: string, password: string) => {
    try {
      const res = await authAPI.login(email, password)
      if (res.tokens) {
        saveTokens(res.tokens)
        setAccessToken(res.tokens.access_token)
        setUser(res.user)
      }
    } catch (err: any) {
      // Check if the error response indicates email not verified.
      // The server returns 403 with requires_verification.
      if (err.message === "email not verified") {
        setPendingVerificationEmail(email)
      }
      throw err
    }
  }, [])

  const registerFn = useCallback(
    async (email: string, password: string, displayName?: string) => {
      const res = await authAPI.register(email, password, displayName)
      if (res.requires_verification) {
        setPendingVerificationEmail(email)
        return
      }
      if (res.tokens) {
        saveTokens(res.tokens)
        setAccessToken(res.tokens.access_token)
        setUser(res.user)
      }
    },
    []
  )

  const logoutFn = useCallback(async () => {
    const tokens = loadTokens()
    if (tokens) {
      await authAPI.logout(tokens.refresh_token).catch(() => {})
    }
    clearTokens()
    setAccessToken(null)
    setUser(null)
  }, [])

  const updateProfileFn = useCallback(async (data: { display_name?: string; email?: string; description?: string }) => {
    if (!accessToken) throw new Error("not authenticated")
    const updated = await authAPI.updateMe(accessToken, data)
    setUser(updated)
  }, [accessToken])

  const changePasswordFn = useCallback(async (currentPassword: string, newPassword: string) => {
    if (!accessToken) throw new Error("not authenticated")
    await authAPI.changePassword(accessToken, currentPassword, newPassword)
  }, [accessToken])

  const verifyEmailFn = useCallback(async (email: string, code: string) => {
    const res = await authAPI.verifyEmail(email, code)
    if (res.tokens) {
      saveTokens(res.tokens)
      setAccessToken(res.tokens.access_token)
      setUser(res.user)
      setPendingVerificationEmail(null)
    }
  }, [])

  const resendVerificationFn = useCallback(async (email: string) => {
    await authAPI.resendVerification(email)
  }, [])

  const clearPendingVerification = useCallback(() => {
    setPendingVerificationEmail(null)
  }, [])

  const refreshAccessToken = useCallback(async (): Promise<string | null> => {
    const tokens = loadTokens()
    if (!tokens) return null
    try {
      const newTokens = await authAPI.refreshToken(tokens.refresh_token)
      saveTokens({
        ...newTokens,
        refresh_token: newTokens.refresh_token || tokens.refresh_token,
      })
      setAccessToken(newTokens.access_token)
      return newTokens.access_token
    } catch {
      clearTokens()
      setAccessToken(null)
      setUser(null)
      return null
    }
  }, [])

  // Wire up the global refresh handler so fetchWithAuth can auto-retry on 401.
  useEffect(() => {
    setAuthRefreshHandler(refreshAccessToken)
  }, [refreshAccessToken])

  const value = useMemo(
    () => ({
      user,
      accessToken,
      loading,
      pendingVerificationEmail,
      login: loginFn,
      register: registerFn,
      logout: logoutFn,
      refreshAccessToken,
      updateProfile: updateProfileFn,
      changePassword: changePasswordFn,
      verifyEmail: verifyEmailFn,
      resendVerification: resendVerificationFn,
      clearPendingVerification,
    }),
    [user, accessToken, loading, pendingVerificationEmail, loginFn, registerFn, logoutFn, refreshAccessToken, updateProfileFn, changePasswordFn, verifyEmailFn, resendVerificationFn, clearPendingVerification]
  )

  return createElement(AuthContext.Provider, { value }, children)
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider")
  }
  return ctx
}
