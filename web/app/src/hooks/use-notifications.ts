import { useState, useEffect, useCallback, useRef } from "react"
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

export function useUnreadCount() {
  const { accessToken } = useAuth()
  const [count, setCount] = useState(0)
  const wsRef = useRef<WebSocket | null>(null)

  // Initial fetch.
  useEffect(() => {
    if (!accessToken) return
    fetchWithAuth<{ unread_count: number }>(
      "/provider/notifications/count",
      accessToken
    ).then((data) => setCount(data.unread_count))
     .catch(() => {})
  }, [accessToken])

  // WebSocket for real-time updates.
  useEffect(() => {
    if (!accessToken) return

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
    const wsUrl = `${protocol}//${window.location.host}/api/v1/provider/notifications/ws?token=${encodeURIComponent(accessToken)}`

    const ws = new WebSocket(wsUrl)
    wsRef.current = ws

    ws.onmessage = () => {
      setCount((prev) => prev + 1)
    }

    ws.onerror = () => {}
    ws.onclose = () => {}

    return () => {
      ws.close()
      wsRef.current = null
    }
  }, [accessToken])

  const decrement = useCallback((n = 1) => {
    setCount((prev) => Math.max(0, prev - n))
  }, [])

  const reset = useCallback(() => {
    setCount(0)
  }, [])

  return { count, setCount, decrement, reset }
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
