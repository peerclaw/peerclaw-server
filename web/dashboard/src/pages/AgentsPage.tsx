import { useState, useMemo } from "react"
import { useAgents } from "@/hooks/use-agents"
import { AgentFilters } from "@/components/agents/AgentFilters"
import { AgentTable } from "@/components/agents/AgentTable"

export function AgentsPage() {
  const [search, setSearch] = useState("")
  const [protocol, setProtocol] = useState("all")
  const [status, setStatus] = useState("all")

  const filters = useMemo(
    () => ({
      protocol: protocol === "all" ? undefined : protocol,
      status: status === "all" ? undefined : status,
    }),
    [protocol, status]
  )

  const { agents, totalCount, error, loading } = useAgents(filters)

  // Client-side name filtering
  const filtered = useMemo(() => {
    if (!search) return agents
    const q = search.toLowerCase()
    return agents.filter(
      (a) =>
        a.name.toLowerCase().includes(q) ||
        a.id.toLowerCase().includes(q)
    )
  }, [agents, search])

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Agents</h1>
        <p className="text-sm text-muted-foreground mt-1">
          {totalCount} registered agent{totalCount !== 1 ? "s" : ""}
        </p>
      </div>

      <AgentFilters
        search={search}
        onSearchChange={setSearch}
        protocol={protocol}
        onProtocolChange={setProtocol}
        status={status}
        onStatusChange={setStatus}
      />

      {loading ? (
        <div className="flex h-40 items-center justify-center">
          <p className="text-sm text-muted-foreground">Loading agents...</p>
        </div>
      ) : error ? (
        <div className="flex h-40 items-center justify-center">
          <p className="text-sm text-destructive">{error}</p>
        </div>
      ) : (
        <AgentTable agents={filtered} />
      )}
    </div>
  )
}
