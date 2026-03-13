import { useEffect, useState, useCallback } from "react"
import { useSearchParams, Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { useAuth } from "@/hooks/use-auth"
import { fetchConsoleDirectory, submitAccessRequest, fetchAccessRequestStatus } from "@/api/client"
import type { PublicAgentProfile } from "@/api/types"
import { CategoryFilter } from "@/components/public/CategoryFilter"
import { VerifiedBadge } from "@/components/public/VerifiedBadge"
import { ReputationMeter } from "@/components/public/ReputationMeter"
import { Button } from "@/components/ui/button"
import { useDebounce } from "@/hooks/use-debounce"
import { Search, SlidersHorizontal, UserPlus, Check } from "lucide-react"

type SortOption = "reputation" | "name" | "registered_at" | "popular"

const PAGE_SIZE = 24

const statusColors: Record<string, string> = {
  online: "bg-emerald-500 shadow-[0_0_6px_oklch(0.72_0.2_160_/_0.5)]",
  offline: "bg-zinc-500",
  degraded: "bg-amber-500 shadow-[0_0_6px_oklch(0.8_0.15_85_/_0.5)]",
}

export function DiscoverAgentsPage() {
  const { t } = useTranslation()
  const { accessToken } = useAuth()
  const [searchParams, setSearchParams] = useSearchParams()
  const [agents, setAgents] = useState<PublicAgentProfile[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [search, setSearch] = useState(searchParams.get("search") ?? "")
  const [sort, setSort] = useState<SortOption>(
    (searchParams.get("sort") as SortOption) ?? "reputation"
  )
  const [verifiedOnly, setVerifiedOnly] = useState(
    searchParams.get("verified") === "true"
  )
  const [category, setCategory] = useState<string | undefined>(
    searchParams.get("category") ?? undefined
  )
  const [nextPageToken, setNextPageToken] = useState<string | undefined>()
  const [requestedIds, setRequestedIds] = useState<Set<string>>(new Set())
  const [requestingId, setRequestingId] = useState<string | null>(null)

  const debouncedSearch = useDebounce(search, 300)

  // Fetch agents
  useEffect(() => {
    if (!accessToken) return
    setLoading(true)
    setNextPageToken(undefined)
    fetchConsoleDirectory(
      {
        search: debouncedSearch || undefined,
        sort,
        verified: verifiedOnly || undefined,
        category,
        page_size: PAGE_SIZE,
      },
      accessToken
    )
      .then((res) => {
        setAgents(res.agents ?? [])
        setTotal(res.total_count)
        setNextPageToken(res.next_page_token)
      })
      .catch(() => {
        setAgents([])
        setTotal(0)
        setNextPageToken(undefined)
      })
      .finally(() => setLoading(false))
  }, [debouncedSearch, sort, verifiedOnly, category, accessToken])

  const loadMore = () => {
    if (!nextPageToken || loadingMore || !accessToken) return
    setLoadingMore(true)
    fetchConsoleDirectory(
      {
        search: debouncedSearch || undefined,
        sort,
        verified: verifiedOnly || undefined,
        category,
        page_size: PAGE_SIZE,
        page_token: nextPageToken,
      },
      accessToken
    )
      .then((res) => {
        setAgents((prev) => [...prev, ...(res.agents ?? [])])
        setNextPageToken(res.next_page_token)
      })
      .catch(() => {})
      .finally(() => setLoadingMore(false))
  }

  const handleRequestAccess = useCallback(
    async (agentId: string) => {
      if (!accessToken || requestingId) return
      setRequestingId(agentId)
      try {
        // Check if already requested
        const status = await fetchAccessRequestStatus(agentId, accessToken)
        if (status.status !== "none" && status.id) {
          setRequestedIds((prev) => new Set(prev).add(agentId))
          return
        }
        await submitAccessRequest(agentId, "", accessToken)
        setRequestedIds((prev) => new Set(prev).add(agentId))
      } catch {
        // If 404 or error, just submit
        try {
          await submitAccessRequest(agentId, "", accessToken)
          setRequestedIds((prev) => new Set(prev).add(agentId))
        } catch {
          // ignore
        }
      } finally {
        setRequestingId(null)
      }
    },
    [accessToken, requestingId]
  )

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
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t('discover.title')}</h1>
        <p className="text-sm text-muted-foreground mt-1">
          {t('discover.description')}
        </p>
      </div>

      {/* Category Filter */}
      <CategoryFilter
        selected={category}
        onChange={(cat) => {
          setCategory(cat)
          updateFilter("category", cat)
        }}
      />

      {/* Filters bar */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative flex-1 min-w-[200px] max-w-md">
          <Search className="absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <input
            type="text"
            placeholder={t('directory.searchPlaceholder')}
            value={search}
            onChange={(e) => {
              setSearch(e.target.value)
              updateFilter("search", e.target.value || undefined)
            }}
            className="w-full rounded-xl border border-border/60 bg-card/50 py-2.5 pl-10 pr-3 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/40 focus:border-primary/40 transition-all"
          />
        </div>

        <div className="flex items-center gap-2">
          <SlidersHorizontal className="size-3.5 text-muted-foreground" />
          <select
            value={sort}
            onChange={(e) => {
              setSort(e.target.value as SortOption)
              updateFilter("sort", e.target.value)
            }}
            className="rounded-lg border border-border/60 bg-card px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/40 transition-all"
          >
            <option value="reputation">{t('directory.sortReputation')}</option>
            <option value="popular">{t('directory.sortPopular')}</option>
            <option value="name">{t('directory.sortName')}</option>
            <option value="registered_at">{t('directory.sortNewest')}</option>
          </select>
        </div>

        <label className="flex items-center gap-2 rounded-lg border border-border/60 bg-card px-3 py-2 text-sm text-muted-foreground cursor-pointer hover:border-primary/30 transition-all">
          <input
            type="checkbox"
            checked={verifiedOnly}
            onChange={(e) => {
              setVerifiedOnly(e.target.checked)
              updateFilter("verified", e.target.checked ? "true" : undefined)
            }}
            className="rounded border-input accent-primary"
          />
          {t('directory.verifiedOnly')}
        </label>
      </div>

      {/* Results */}
      {loading ? (
        <div className="flex h-48 items-center justify-center">
          <div className="flex flex-col items-center gap-3">
            <div className="size-6 animate-spin rounded-full border-2 border-primary/30 border-t-primary" />
            <p className="text-sm text-muted-foreground">{t('common.loading')}</p>
          </div>
        </div>
      ) : agents.length === 0 ? (
        <div className="flex h-48 flex-col items-center justify-center rounded-xl border border-dashed border-border/60">
          <div className="size-12 rounded-full bg-secondary/50 flex items-center justify-center mb-3">
            <Search className="size-5 text-muted-foreground" />
          </div>
          <p className="text-sm text-muted-foreground">{t('directory.noAgents')}</p>
          {(search || category) && (
            <button
              onClick={() => {
                setSearch("")
                setCategory(undefined)
                updateFilter("search", undefined)
                updateFilter("category", undefined)
              }}
              className="mt-2 text-xs text-primary hover:underline"
            >
              {t('directory.clearFilters')}
            </button>
          )}
        </div>
      ) : (
        <>
          <p className="text-xs text-muted-foreground">
            {t('directory.agentsRegistered', { count: total })}
          </p>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {agents.map((agent) => (
              <div
                key={agent.id}
                className="relative rounded-xl border border-border/60 bg-card p-4 transition-all duration-300 hover:border-primary/30 hover:shadow-[0_0_20px_oklch(0.72_0.15_192_/_0.06)]"
              >
                <Link to={`/agents/${agent.id}`} className="block group">
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <h3 className="truncate font-semibold text-sm text-foreground transition-colors group-hover:text-primary">
                          {agent.name}
                        </h3>
                        <span
                          className={`size-2 shrink-0 rounded-full transition-all ${statusColors[agent.status] ?? "bg-zinc-500"}`}
                          title={agent.status}
                        />
                        {agent.verified && <VerifiedBadge />}
                      </div>
                      {agent.description && (
                        <p className="mt-1.5 line-clamp-2 text-xs leading-relaxed text-muted-foreground">
                          {agent.description}
                        </p>
                      )}
                    </div>
                    <div className="shrink-0">
                      <ReputationMeter score={agent.reputation_score} size="sm" />
                    </div>
                  </div>

                  {/* Capabilities */}
                  <div className="mt-3 flex flex-wrap gap-1.5">
                    {agent.capabilities?.slice(0, 3).map((cap) => (
                      <span
                        key={cap}
                        className="rounded-md bg-secondary/80 px-1.5 py-0.5 text-[10px] font-medium text-secondary-foreground"
                      >
                        {cap}
                      </span>
                    ))}
                    {(agent.capabilities?.length ?? 0) > 3 && (
                      <span className="text-[10px] text-muted-foreground">
                        +{agent.capabilities!.length - 3}
                      </span>
                    )}
                  </div>

                  {/* Protocols */}
                  {agent.protocols && agent.protocols.length > 0 && (
                    <div className="mt-2 flex gap-1.5">
                      {agent.protocols.map((p) => (
                        <span
                          key={p}
                          className="rounded-md bg-primary/8 px-1.5 py-0.5 text-[10px] font-mono font-medium text-primary"
                        >
                          {p}
                        </span>
                      ))}
                    </div>
                  )}
                </Link>

                {/* Request Access button */}
                <div className="mt-3 pt-3 border-t border-border/40">
                  {requestedIds.has(agent.id) ? (
                    <Button variant="outline" size="sm" disabled className="w-full text-xs">
                      <Check className="size-3.5 text-emerald-500" />
                      {t('discover.accessRequested')}
                    </Button>
                  ) : (
                    <Button
                      variant="outline"
                      size="sm"
                      className="w-full text-xs"
                      disabled={requestingId === agent.id}
                      onClick={() => handleRequestAccess(agent.id)}
                    >
                      <UserPlus className="size-3.5" />
                      {t('discover.requestAccess')}
                    </Button>
                  )}
                </div>

                {/* Bottom accent */}
                <div className="absolute bottom-0 left-4 right-4 h-px bg-gradient-to-r from-transparent via-primary/0 to-transparent transition-all duration-300 hover:via-primary/40" />
              </div>
            ))}
          </div>

          {/* Load More */}
          {nextPageToken && (
            <div className="mt-8 flex justify-center">
              <button
                onClick={loadMore}
                disabled={loadingMore}
                className="rounded-xl border border-border/60 bg-card px-8 py-2.5 text-sm font-medium transition-all hover:border-primary/30 hover:text-primary disabled:opacity-50"
              >
                {loadingMore ? (
                  <span className="flex items-center gap-2">
                    <div className="size-3.5 animate-spin rounded-full border-2 border-primary/30 border-t-primary" />
                    {t('directory.loadingMore')}
                  </span>
                ) : (
                  t('directory.loadMore', { shown: agents.length, total })
                )}
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
