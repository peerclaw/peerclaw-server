import { useState, useEffect, useRef } from "react"
import { ChevronDown, Search, Bot } from "lucide-react"
import { fetchDirectory } from "@/api/client"
import type { PublicAgentProfile } from "@/api/types"

interface AgentSelectorProps {
  selectedId: string | null
  onSelect: (id: string) => void
}

export function AgentSelector({ selectedId, onSelect }: AgentSelectorProps) {
  const [agents, setAgents] = useState<PublicAgentProfile[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState("")
  const [open, setOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    fetchDirectory({ page_size: 100 })
      .then((res) => setAgents(res.agents ?? []))
      .catch(() => setAgents([]))
      .finally(() => setLoading(false))
  }, [])

  // Close on outside click
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (
        containerRef.current &&
        !containerRef.current.contains(e.target as Node)
      ) {
        setOpen(false)
      }
    }
    document.addEventListener("mousedown", handleClick)
    return () => document.removeEventListener("mousedown", handleClick)
  }, [])

  const selectedAgent = agents.find((a) => a.id === selectedId)

  const filtered = agents.filter((a) => {
    if (!search) return true
    const q = search.toLowerCase()
    return (
      a.name.toLowerCase().includes(q) ||
      a.id.toLowerCase().includes(q) ||
      (a.description ?? "").toLowerCase().includes(q)
    )
  })

  return (
    <div ref={containerRef} className="relative">
      {/* Trigger */}
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex w-full items-center justify-between gap-2 rounded-lg border border-input bg-background px-3 py-2 text-sm transition-colors hover:bg-accent/50 focus:outline-none focus:ring-2 focus:ring-ring dark:bg-input/30"
      >
        <span className="flex items-center gap-2 truncate">
          <Bot className="size-4 shrink-0 text-muted-foreground" />
          {selectedAgent ? (
            <span className="truncate">{selectedAgent.name}</span>
          ) : (
            <span className="text-muted-foreground">Select an agent...</span>
          )}
        </span>
        <ChevronDown
          className={`size-4 shrink-0 text-muted-foreground transition-transform ${
            open ? "rotate-180" : ""
          }`}
        />
      </button>

      {/* Dropdown */}
      {open && (
        <div className="absolute left-0 top-full z-50 mt-1 w-full rounded-lg border border-border bg-card shadow-lg">
          {/* Search */}
          <div className="border-b border-border p-2">
            <div className="relative">
              <Search className="absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search agents..."
                className="w-full rounded-md border border-input bg-background py-1.5 pl-8 pr-3 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring dark:bg-input/30"
                autoFocus
              />
            </div>
          </div>

          {/* List */}
          <div className="max-h-64 overflow-auto p-1">
            {loading ? (
              <div className="px-3 py-4 text-center text-xs text-muted-foreground">
                Loading agents...
              </div>
            ) : filtered.length === 0 ? (
              <div className="px-3 py-4 text-center text-xs text-muted-foreground">
                No agents found
              </div>
            ) : (
              filtered.map((agent) => (
                <button
                  key={agent.id}
                  onClick={() => {
                    onSelect(agent.id)
                    setOpen(false)
                    setSearch("")
                  }}
                  className={`flex w-full items-start gap-3 rounded-md px-3 py-2 text-left text-sm transition-colors hover:bg-accent/50 ${
                    agent.id === selectedId
                      ? "bg-accent text-accent-foreground"
                      : "text-foreground"
                  }`}
                >
                  <div
                    className={`mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-md text-xs font-bold ${
                      agent.status === "online"
                        ? "bg-emerald-500/20 text-emerald-400"
                        : agent.status === "degraded"
                        ? "bg-amber-500/20 text-amber-400"
                        : "bg-muted text-muted-foreground"
                    }`}
                  >
                    <Bot className="size-3.5" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="truncate font-medium">
                        {agent.name}
                      </span>
                      <span
                        className={`shrink-0 rounded-full px-1.5 py-0.5 text-[10px] font-medium ${
                          agent.status === "online"
                            ? "bg-emerald-500/20 text-emerald-400"
                            : agent.status === "degraded"
                            ? "bg-amber-500/20 text-amber-400"
                            : "bg-muted text-muted-foreground"
                        }`}
                      >
                        {agent.status}
                      </span>
                    </div>
                    {agent.description && (
                      <p className="mt-0.5 truncate text-xs text-muted-foreground">
                        {agent.description}
                      </p>
                    )}
                  </div>
                </button>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  )
}
