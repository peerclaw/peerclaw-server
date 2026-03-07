import { useState, useCallback, useRef } from "react"

const BASE = "/api/v1"
const TOKEN_KEY = "peerclaw_tokens"

export interface ChatMessage {
  id: string
  role: "user" | "assistant"
  content: string
  timestamp: Date
  raw?: unknown
}

function getAccessToken(): string | null {
  const raw = localStorage.getItem(TOKEN_KEY)
  if (!raw) return null
  try {
    const tokens = JSON.parse(raw)
    return tokens.access_token ?? null
  } catch {
    return null
  }
}

function generateId(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`
}

export function usePlayground() {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [lastRaw, setLastRaw] = useState<unknown>(null)
  const abortRef = useRef<AbortController | null>(null)

  const sendMessage = useCallback(
    async (agentId: string, message: string, stream = false) => {
      if (!message.trim()) return

      const accessToken = getAccessToken()
      if (!accessToken) {
        setError("Not authenticated. Please sign in first.")
        return
      }

      // Add user message
      const userMsg: ChatMessage = {
        id: generateId(),
        role: "user",
        content: message.trim(),
        timestamp: new Date(),
      }
      setMessages((prev) => [...prev, userMsg])
      setError(null)
      setLoading(true)
      setLastRaw(null)

      // Cancel any in-flight request
      if (abortRef.current) {
        abortRef.current.abort()
      }
      const controller = new AbortController()
      abortRef.current = controller

      try {
        if (stream) {
          // SSE streaming via fetch + ReadableStream
          const res = await fetch(`${BASE}/invoke/${agentId}`, {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
              Authorization: `Bearer ${accessToken}`,
              Accept: "text/event-stream",
            },
            body: JSON.stringify({ message: message.trim(), stream: true }),
            signal: controller.signal,
          })

          if (!res.ok) {
            const body = await res.json().catch(() => ({ error: res.statusText }))
            throw new Error(body.error || `API error: ${res.status}`)
          }

          const assistantId = generateId()
          setMessages((prev) => [
            ...prev,
            {
              id: assistantId,
              role: "assistant",
              content: "",
              timestamp: new Date(),
            },
          ])

          const reader = res.body?.getReader()
          if (!reader) throw new Error("No response body")

          const decoder = new TextDecoder()
          let accumulated = ""
          let rawChunks: string[] = []

          while (true) {
            const { done, value } = await reader.read()
            if (done) break

            const chunk = decoder.decode(value, { stream: true })
            rawChunks.push(chunk)

            // Parse SSE events
            const lines = chunk.split("\n")
            for (const line of lines) {
              if (line.startsWith("data: ")) {
                const data = line.slice(6)
                if (data === "[DONE]") break

                try {
                  const parsed = JSON.parse(data)
                  const content = parsed.content ?? parsed.text ?? parsed.delta ?? data
                  accumulated += typeof content === "string" ? content : JSON.stringify(content)
                } catch {
                  // Plain text data
                  accumulated += data
                }

                setMessages((prev) =>
                  prev.map((m) =>
                    m.id === assistantId
                      ? { ...m, content: accumulated }
                      : m
                  )
                )
              }
            }
          }

          setLastRaw(rawChunks)
        } else {
          // Standard request/response
          const res = await fetch(`${BASE}/invoke/${agentId}`, {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
              Authorization: `Bearer ${accessToken}`,
            },
            body: JSON.stringify({ message: message.trim(), stream: false }),
            signal: controller.signal,
          })

          if (!res.ok) {
            const body = await res.json().catch(() => ({ error: res.statusText }))
            throw new Error(body.error || `API error: ${res.status}`)
          }

          const data = await res.json()
          setLastRaw(data)

          const content =
            typeof data === "string"
              ? data
              : data.response ?? data.content ?? data.text ?? data.message ?? JSON.stringify(data, null, 2)

          const assistantMsg: ChatMessage = {
            id: generateId(),
            role: "assistant",
            content,
            timestamp: new Date(),
            raw: data,
          }
          setMessages((prev) => [...prev, assistantMsg])
        }
      } catch (err) {
        if ((err as Error).name === "AbortError") return
        const errMsg = err instanceof Error ? err.message : "Failed to invoke agent"
        setError(errMsg)

        // Add error as assistant message
        setMessages((prev) => [
          ...prev,
          {
            id: generateId(),
            role: "assistant",
            content: `Error: ${errMsg}`,
            timestamp: new Date(),
          },
        ])
      } finally {
        setLoading(false)
        abortRef.current = null
      }
    },
    []
  )

  const clearMessages = useCallback(() => {
    // Cancel in-flight request
    if (abortRef.current) {
      abortRef.current.abort()
      abortRef.current = null
    }
    setMessages([])
    setError(null)
    setLastRaw(null)
    setLoading(false)
  }, [])

  return { messages, loading, error, lastRaw, sendMessage, clearMessages }
}
