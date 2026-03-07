import { useState, useEffect, useCallback } from "react"
import type { Agent } from "@/api/types"
import { fetchAgents } from "@/api/client"

export function useAgents(filters?: { protocol?: string; status?: string }) {
  const [agents, setAgents] = useState<Agent[]>([])
  const [totalCount, setTotalCount] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    try {
      setLoading(true)
      const data = await fetchAgents(filters)
      setAgents(data.agents ?? [])
      setTotalCount(data.total_count)
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to fetch agents")
    } finally {
      setLoading(false)
    }
  }, [filters?.protocol, filters?.status])

  useEffect(() => {
    load()
  }, [load])

  return { agents, totalCount, error, loading, reload: load }
}
