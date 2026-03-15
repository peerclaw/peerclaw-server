import { useState, useEffect, useCallback, useSyncExternalStore } from "react"
import { fetchWithAuth } from "@/api/client"
import { useAuth } from "@/hooks/use-auth"

export interface Notification {
  id: string
  user_id: string
  agent_id: string
  type: string
  severity: string
  title: string
  body: string
  metadata?: Record<string, string>
  read: boolean
  created_at: string
}

interface NotificationListResponse {
  notifications: Notification[]
  total: number
  unread_count: number
}

export function useNotifications(limit = 20) {
  const { accessToken } = useAuth()
  const [notifications, setNotifications] = useState<Notification[]>([])
  const [total, setTotal] = useState(0)
  const [unreadCount, setUnreadCount] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!accessToken) return
    try {
      setLoading(true)
      const data = await fetchWithAuth<NotificationListResponse>(
        `/provider/notifications?limit=${limit}`,
        accessToken
      )
      setNotifications(data.notifications ?? [])
      setTotal(data.total)
      setUnreadCount(data.unread_count)
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to fetch notifications")
    } finally {
      setLoading(false)
    }
  }, [accessToken, limit])

  useEffect(() => {
    load()
  }, [load])

  return { notifications, total, unreadCount, loading, error, reload: load }
}

// --- Shared singleton store for unread count ---
// All components using useUnreadCount() share the same state.
let _count = 0
let _listeners: Array<() => void> = []

function _getSnapshot() {
  return _count
}

function _subscribe(listener: () => void) {
  _listeners.push(listener)
  return () => {
    _listeners = _listeners.filter((l) => l !== listener)
  }
}

function _setCount(value: number | ((prev: number) => number)) {
  const next = typeof value === "function" ? value(_count) : value
  if (next !== _count) {
    _count = next
    _listeners.forEach((l) => l())
  }
}

// Track whether the WebSocket is already connected (singleton).
let _wsConnected = false
let _ws: WebSocket | null = null
let _wsToken: string | null = null

export function useUnreadCount() {
  const { accessToken } = useAuth()
  const count = useSyncExternalStore(_subscribe, _getSnapshot)

  // Initial fetch (only once per token change).
  useEffect(() => {
    if (!accessToken) return
    fetchWithAuth<{ unread_count: number }>(
      "/provider/notifications/count",
      accessToken
    ).then((data) => _setCount(data.unread_count))
     .catch(() => {})
  }, [accessToken])

  // WebSocket for real-time updates (singleton — only one connection).
  useEffect(() => {
    if (!accessToken) return
    // If a WebSocket is already running for this token, skip.
    if (_wsConnected && _wsToken === accessToken) return

    // Close previous if token changed.
    if (_ws) {
      _ws.close()
      _ws = null
    }

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
    const wsUrl = `${protocol}//${window.location.host}/api/v1/provider/notifications/ws?token=${encodeURIComponent(accessToken)}`

    const ws = new WebSocket(wsUrl)
    _ws = ws
    _wsToken = accessToken
    _wsConnected = true

    ws.onmessage = () => {
      _setCount((prev) => prev + 1)
    }

    ws.onerror = () => {}
    ws.onclose = () => {
      if (_ws === ws) {
        _wsConnected = false
        _ws = null
        _wsToken = null
      }
    }

    return () => {
      // Only close if this effect owns the current WebSocket.
      if (_ws === ws) {
        ws.close()
        _wsConnected = false
        _ws = null
        _wsToken = null
      }
    }
  }, [accessToken])

  const decrement = useCallback((n = 1) => {
    _setCount((prev) => Math.max(0, prev - n))
  }, [])

  const reset = useCallback(() => {
    _setCount(0)
  }, [])

  return { count, decrement, reset }
}

export function useNotificationMutations() {
  const { accessToken } = useAuth()

  const markRead = useCallback(async (id: string) => {
    if (!accessToken) return
    await fetchWithAuth(`/provider/notifications/${id}/read`, accessToken, {
      method: "PUT",
    })
  }, [accessToken])

  const markAllRead = useCallback(async () => {
    if (!accessToken) return
    await fetchWithAuth("/provider/notifications/read-all", accessToken, {
      method: "PUT",
    })
  }, [accessToken])

  return { markRead, markAllRead }
}
