import { useState, useEffect } from "react"
import { useParams } from "react-router-dom"
import { fetchAgent } from "@/api/client"
import { AgentCardView } from "@/components/agents/AgentCardView"
import type { Agent } from "@/api/types"

export function AgentDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [agent, setAgent] = useState<Agent | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!id) return
    setLoading(true)
    fetchAgent(id)
      .then((data) => {
        setAgent(data)
        setError(null)
      })
      .catch((e) => setError(e instanceof Error ? e.message : "Failed to load agent"))
      .finally(() => setLoading(false))
  }, [id])

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <p className="text-sm text-muted-foreground">Loading agent details...</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex h-64 items-center justify-center">
        <p className="text-sm text-destructive">{error}</p>
      </div>
    )
  }

  if (!agent) return null

  return <AgentCardView agent={agent} />
}
