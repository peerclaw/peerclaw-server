import { useState, useEffect } from "react"
import { Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { useProviderAgents } from "@/hooks/use-provider"
import { ClaimTokenSection } from "@/components/provider/ClaimTokenSection"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Table,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Card,
  CardContent,
} from "@/components/ui/card"
import { SortableHeader } from "@/components/ui/sortable-header"
import { TableSkeleton } from "@/components/ui/table-skeleton"
import { Bot, PlusCircle, ChevronDown, ChevronUp } from "lucide-react"

import { DEFAULT_PAGE_SIZE } from "@/lib/constants"

const PAGE_SIZE = DEFAULT_PAGE_SIZE

export function ProviderAgentsPage() {
  const { t } = useTranslation()
  const [search, setSearch] = useState("")
  const [status, setStatus] = useState("")
  const [sort, setSort] = useState("")
  const [page, setPage] = useState(0)
  const [showRegister, setShowRegister] = useState(false)

  const { data, isLoading, error } = useProviderAgents(
    search || undefined,
    status || undefined,
    sort || undefined,
    PAGE_SIZE,
    page * PAGE_SIZE
  )

  const agents = data?.agents ?? []
  const total = data?.total_count ?? 0
  const totalPages = Math.ceil(total / PAGE_SIZE)

  useEffect(() => {
    if (totalPages > 0 && page >= totalPages) setPage(totalPages - 1)
  }, [totalPages, page])

  const statusColor = (s: string) => {
    switch (s) {
      case "online":
        return "default" as const
      case "degraded":
        return "secondary" as const
      default:
        return "outline" as const
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t('nav.myAgents')}</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {t('directory.agentsRegistered', { count: total })}
          </p>
        </div>
        <button
          onClick={() => setShowRegister(!showRegister)}
          className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
        >
          <PlusCircle className="size-4" />
          {t('nav.registerAgent')}
          {showRegister ? <ChevronUp className="size-3.5" /> : <ChevronDown className="size-3.5" />}
        </button>
      </div>

      {showRegister && <ClaimTokenSection />}

      <div className="flex gap-3">
        <Input
          placeholder={t('provider.searchPlaceholder')}
          value={search}
          onChange={(e) => {
            setSearch(e.target.value)
            setPage(0)
          }}
          className="max-w-sm"
        />
        <select
          value={status}
          onChange={(e) => {
            setStatus(e.target.value)
            setPage(0)
          }}
          className="rounded-md border border-input bg-background px-3 py-2 text-sm"
        >
          <option value="">{t('provider.allStatus')}</option>
          <option value="online">{t('adminAgents.online')}</option>
          <option value="offline">{t('adminAgents.offline')}</option>
          <option value="degraded">{t('adminAgents.degraded')}</option>
        </select>
      </div>

      {isLoading ? (
        <Card>
          <CardContent className="p-0">
            <div aria-live="polite">
              <TableSkeleton rows={5} cols={5} />
            </div>
          </CardContent>
        </Card>
      ) : error ? (
        <div className="flex h-40 items-center justify-center">
          <p className="text-sm text-destructive" role="alert">{error?.message}</p>
        </div>
      ) : agents.length === 0 && !search && !status ? (
        <div className="flex flex-col items-center justify-center h-40 rounded-lg border border-dashed border-border">
          <Bot className="size-8 text-muted-foreground mb-2" />
          <p className="text-sm text-muted-foreground">{t('provider.noAgentsRegistered')}</p>
          {!showRegister && (
            <button
              onClick={() => setShowRegister(true)}
              className="text-sm text-primary hover:underline mt-1"
            >
              {t('provider.registerFirst')}
            </button>
          )}
        </div>
      ) : (
        <>
          <Card>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <SortableHeader field="name" label={t('provider.name')} currentSort={sort} onSort={(s) => { setSort(s); setPage(0) }} />
                    <SortableHeader field="status" label={t('provider.status')} currentSort={sort} onSort={(s) => { setSort(s); setPage(0) }} />
                    <SortableHeader field="total_calls" label={t('provider.calls')} currentSort={sort} onSort={(s) => { setSort(s); setPage(0) }} />
                    <SortableHeader field="success_rate" label={t('provider.successRate')} currentSort={sort} onSort={(s) => { setSort(s); setPage(0) }} />
                    <SortableHeader field="registered_at" label={t('provider.registered')} currentSort={sort} onSort={(s) => { setSort(s); setPage(0) }} />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {agents.map((agent) => (
                    <TableRow key={agent.id}>
                      <TableCell>
                        <Link
                          to={`/console/agents/${agent.id}`}
                          className="font-medium text-foreground hover:text-primary transition-colors"
                        >
                          {agent.name}
                        </Link>
                        <p className="text-xs text-muted-foreground mt-0.5 max-w-xs truncate">
                          {agent.description}
                        </p>
                      </TableCell>
                      <TableCell>
                        <Badge variant={statusColor(agent.status)}>
                          {agent.status}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {(agent.total_calls ?? 0).toLocaleString()}
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {(agent.success_rate ?? 0).toFixed(1)}%
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {agent.registered_at
                          ? new Date(agent.registered_at).toLocaleDateString()
                          : "-"}
                      </TableCell>
                    </TableRow>
                  ))}
                  {agents.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={5} className="text-center text-muted-foreground py-8">
                        {t('common.noResults')}
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>

          {totalPages > 1 && (
            <div className="flex items-center justify-between">
              <p className="text-sm text-muted-foreground">
                {t('common.page')} {page + 1} / {totalPages}
              </p>
              <div className="flex gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  disabled={page === 0}
                  onClick={() => setPage((p) => p - 1)}
                >
                  {t('common.previous')}
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={page >= totalPages - 1}
                  onClick={() => setPage((p) => p + 1)}
                >
                  {t('common.next')}
                </Button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  )
}
