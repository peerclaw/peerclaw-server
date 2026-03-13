import { useEffect, useState } from "react"
import { useSearchParams } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { fetchDirectory } from "@/api/client"
import type { PublicAgentProfile } from "@/api/types"
import { AgentDirectoryCard } from "@/components/public/AgentDirectoryCard"
import { CategoryFilter } from "@/components/public/CategoryFilter"
import { useDebounce } from "@/hooks/use-debounce"
import { Search, SlidersHorizontal } from "lucide-react"

type SortOption = "reputation" | "name" | "registered_at" | "popular"

const PAGE_SIZE = 24

export function DirectoryPage() {
  const { t } = useTranslation()
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

  const debouncedSearch = useDebounce(search, 300)

  // Reset and load first page when filters change
  useEffect(() => {
    setLoading(true)
    setNextPageToken(undefined)
    fetchDirectory({
      search: debouncedSearch || undefined,
      sort,
      verified: verifiedOnly || undefined,
      category,
      page_size: PAGE_SIZE,
    })
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
  }, [debouncedSearch, sort, verifiedOnly, category])

  const loadMore = () => {
    if (!nextPageToken || loadingMore) return
    setLoadingMore(true)
    fetchDirectory({
      search: debouncedSearch || undefined,
      sort,
      verified: verifiedOnly || undefined,
      category,
      page_size: PAGE_SIZE,
      page_token: nextPageToken,
    })
      .then((res) => {
        setAgents((prev) => [...prev, ...(res.agents ?? [])])
        setNextPageToken(res.next_page_token)
      })
      .catch(() => {
        // keep existing agents on error
      })
      .finally(() => setLoadingMore(false))
  }

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
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-2xl font-bold tracking-tight">{t('directory.title')}</h1>
        <p className="mt-1.5 text-sm text-muted-foreground">
          {t('directory.agentsRegistered', { count: total })}
        </p>
      </div>

      {/* Category Filter */}
      <div className="mb-5">
        <CategoryFilter
          selected={category}
          onChange={(cat) => {
            setCategory(cat)
            updateFilter("category", cat)
          }}
        />
      </div>

      {/* Filters bar */}
      <div className="mb-6 flex flex-wrap items-center gap-3">
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

      {/* Grid */}
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
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {agents.map((agent) => (
              <AgentDirectoryCard key={agent.id} agent={agent} />
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
