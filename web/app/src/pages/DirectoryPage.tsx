import { useEffect, useState } from "react"
import { useSearchParams } from "react-router-dom"
import { fetchDirectory } from "@/api/client"
import type { PublicAgentProfile } from "@/api/types"
import { AgentDirectoryCard } from "@/components/public/AgentDirectoryCard"
import { Search } from "lucide-react"

type SortOption = "reputation" | "name" | "registered_at"

export function DirectoryPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [agents, setAgents] = useState<PublicAgentProfile[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState(searchParams.get("search") ?? "")
  const [sort, setSort] = useState<SortOption>(
    (searchParams.get("sort") as SortOption) ?? "reputation"
  )
  const [verifiedOnly, setVerifiedOnly] = useState(
    searchParams.get("verified") === "true"
  )

  useEffect(() => {
    setLoading(true)
    fetchDirectory({
      search: search || undefined,
      sort,
      verified: verifiedOnly || undefined,
      page_size: 24,
    })
      .then((res) => {
        setAgents(res.agents ?? [])
        setTotal(res.total_count)
      })
      .catch(() => {
        setAgents([])
        setTotal(0)
      })
      .finally(() => setLoading(false))
  }, [search, sort, verifiedOnly])

  const updateFilter = (key: string, value: string | undefined) => {
    const params = new URLSearchParams(searchParams)
    if (value) {
      params.set(key, value)
    } else {
      params.delete(key)
    }
    setSearchParams(params, { replace: true })
  }

  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold">Agent Directory</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {total} agent{total !== 1 ? "s" : ""} registered
        </p>
      </div>

      {/* Filters */}
      <div className="mb-6 flex flex-wrap items-center gap-3">
        <div className="relative flex-1 min-w-[200px] max-w-md">
          <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search agents..."
            value={search}
            onChange={(e) => {
              setSearch(e.target.value)
              updateFilter("search", e.target.value || undefined)
            }}
            className="w-full rounded-lg border border-input bg-background py-2 pl-9 pr-3 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
          />
        </div>

        <select
          value={sort}
          onChange={(e) => {
            setSort(e.target.value as SortOption)
            updateFilter("sort", e.target.value)
          }}
          className="rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
        >
          <option value="reputation">Sort by Reputation</option>
          <option value="name">Sort by Name</option>
          <option value="registered_at">Sort by Newest</option>
        </select>

        <label className="flex items-center gap-2 text-sm text-muted-foreground">
          <input
            type="checkbox"
            checked={verifiedOnly}
            onChange={(e) => {
              setVerifiedOnly(e.target.checked)
              updateFilter("verified", e.target.checked ? "true" : undefined)
            }}
            className="rounded border-input"
          />
          Verified only
        </label>
      </div>

      {/* Grid */}
      {loading ? (
        <div className="flex h-48 items-center justify-center text-muted-foreground text-sm">
          Loading...
        </div>
      ) : agents.length === 0 ? (
        <div className="flex h-48 flex-col items-center justify-center text-muted-foreground">
          <p className="text-sm">No agents found</p>
          {search && (
            <button
              onClick={() => {
                setSearch("")
                updateFilter("search", undefined)
              }}
              className="mt-2 text-xs text-primary hover:underline"
            >
              Clear search
            </button>
          )}
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {agents.map((agent) => (
            <AgentDirectoryCard key={agent.id} agent={agent} />
          ))}
        </div>
      )}
    </div>
  )
}
