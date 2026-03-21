import { useState, useMemo } from "react"
import { useTranslation } from "react-i18next"
import { useAdminInvocations } from "@/hooks/use-admin"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Download } from "lucide-react"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { exportToCSV, exportToJSON } from "@/lib/export"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Card,
  CardContent,
} from "@/components/ui/card"
import { SortableHeader } from "@/components/ui/sortable-header"
import { TableSkeleton } from "@/components/ui/table-skeleton"
import { CopyButton } from "@/components/ui/copy-button"
import type { InvocationRecord } from "@/api/types"

const PAGE_SIZE = 50

type TimeRange = "today" | "7d" | "30d" | "all"

function getSinceISO(range: TimeRange): string | undefined {
  if (range === "all") return undefined
  const now = new Date()
  switch (range) {
    case "today": {
      const start = new Date(now.getFullYear(), now.getMonth(), now.getDate())
      return start.toISOString()
    }
    case "7d":
      return new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000).toISOString()
    case "30d":
      return new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000).toISOString()
  }
}

export function InvocationsPage() {
  const { t } = useTranslation()
  const [agentFilter, setAgentFilter] = useState("")
  const [userFilter, setUserFilter] = useState("")
  const [sort, setSort] = useState("")
  const [page, setPage] = useState(0)
  const [timeRange, setTimeRange] = useState<TimeRange>("all")
  const [detailRecord, setDetailRecord] = useState<InvocationRecord | null>(null)

  const since = useMemo(() => getSinceISO(timeRange), [timeRange])

  const { data, loading, error } = useAdminInvocations(
    agentFilter || undefined,
    userFilter || undefined,
    sort || undefined,
    since,
    PAGE_SIZE,
    page * PAGE_SIZE
  )

  const total = data?.total ?? 0
  const totalPages = Math.ceil(total / PAGE_SIZE)
  const invocations = data?.invocations ?? []

  const statusSummary = useMemo(() => {
    if (!invocations.length) return null
    const success = invocations.filter(i => i.status_code >= 200 && i.status_code < 400).length
    const errors = invocations.filter(i => i.status_code >= 400 || i.error).length
    const avgMs = Math.round(invocations.reduce((sum, i) => sum + i.duration_ms, 0) / invocations.length)
    return { success, errors, avgMs }
  }, [invocations])

  const statusBadgeVariant = (code: number) => {
    if (code >= 200 && code < 300) return "default"
    if (code >= 400 && code < 500) return "secondary"
    if (code >= 500) return "destructive"
    return "outline"
  }

  const timeRanges: { key: TimeRange; label: string }[] = [
    { key: "today", label: t('invocations.today') },
    { key: "7d", label: t('invocations.last7d') },
    { key: "30d", label: t('invocations.last30d') },
    { key: "all", label: t('invocations.allTime') },
  ]

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t('adminInvocations.title')}</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {t('adminInvocations.records', { count: total })}
          </p>
        </div>
        {invocations.length > 0 && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button size="sm" variant="outline">
                <Download className="size-4 mr-1.5" />
                {t('common.export')}
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => exportToCSV(
                invocations.map(inv => ({
                  id: inv.id,
                  agent_id: inv.agent_id,
                  user_id: inv.user_id || "",
                  protocol: inv.protocol || "",
                  status_code: String(inv.status_code),
                  duration_ms: String(inv.duration_ms),
                  created_at: inv.created_at,
                })),
                [
                  { key: "id", label: "ID" },
                  { key: "agent_id", label: "Agent ID" },
                  { key: "user_id", label: "User ID" },
                  { key: "protocol", label: "Protocol" },
                  { key: "status_code", label: "Status Code" },
                  { key: "duration_ms", label: "Duration (ms)" },
                  { key: "created_at", label: "Created At" },
                ],
                "invocations"
              )}>
                {t('common.exportCSV')}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => exportToJSON(invocations, "invocations")}>
                {t('common.exportJSON')}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>

      {/* Date range selector */}
      <div className="flex items-center gap-2">
        {timeRanges.map(({ key, label }) => (
          <Button
            key={key}
            size="sm"
            variant={timeRange === key ? "default" : "outline"}
            onClick={() => { setTimeRange(key); setPage(0) }}
          >
            {label}
          </Button>
        ))}
      </div>

      <div className="flex gap-3">
        <Input
          placeholder={t('adminInvocations.filterAgent')}
          value={agentFilter}
          onChange={(e) => {
            setAgentFilter(e.target.value)
            setPage(0)
          }}
          className="max-w-xs"
        />
        <Input
          placeholder={t('adminInvocations.filterUser')}
          value={userFilter}
          onChange={(e) => {
            setUserFilter(e.target.value)
            setPage(0)
          }}
          className="max-w-xs"
        />
      </div>

      {/* Status summary bar */}
      {statusSummary && !loading && (
        <div className="text-sm text-muted-foreground">
          {t('invocations.statusSummary', {
            total,
            success: statusSummary.success,
            errors: statusSummary.errors,
            avg: statusSummary.avgMs,
          })}
        </div>
      )}

      {loading ? (
        <Card>
          <CardContent className="p-0">
            <div aria-live="polite"><TableSkeleton rows={5} cols={8} /></div>
          </CardContent>
        </Card>
      ) : error ? (
        <div className="flex h-40 items-center justify-center">
          <p className="text-sm text-destructive" role="alert">{error}</p>
        </div>
      ) : (
        <>
          <Card>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('adminInvocations.id')}</TableHead>
                    <TableHead>{t('adminInvocations.agentId')}</TableHead>
                    <TableHead>{t('adminInvocations.userId')}</TableHead>
                    <TableHead>{t('adminInvocations.protocol')}</TableHead>
                    <SortableHeader field="status_code" label={t('adminInvocations.status')} currentSort={sort} onSort={(s) => { setSort(s); setPage(0) }} />
                    <SortableHeader field="duration_ms" label={t('adminInvocations.duration')} currentSort={sort} onSort={(s) => { setSort(s); setPage(0) }} />
                    <TableHead>{t('adminInvocations.error')}</TableHead>
                    <SortableHeader field="created_at" label={t('adminInvocations.createdAt')} currentSort={sort} onSort={(s) => { setSort(s); setPage(0) }} />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {invocations.map((inv) => (
                    <TableRow
                      key={inv.id}
                      className="cursor-pointer hover:bg-muted/50"
                      onClick={() => setDetailRecord(inv)}
                    >
                      <TableCell className="font-mono text-xs max-w-[80px]">
                        <div className="flex items-center gap-1">
                          <span className="truncate">{inv.id.slice(0, 8)}...</span>
                          <CopyButton value={inv.id} />
                        </div>
                      </TableCell>
                      <TableCell className="font-mono text-xs max-w-[100px]">
                        <div className="flex items-center gap-1">
                          <span className="truncate">{inv.agent_id}</span>
                          <CopyButton value={inv.agent_id} />
                        </div>
                      </TableCell>
                      <TableCell className="font-mono text-xs max-w-[100px] truncate">
                        {inv.user_id || "-"}
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className="text-xs">
                          {inv.protocol || "unknown"}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant={statusBadgeVariant(inv.status_code)}>
                          {inv.status_code || "N/A"}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-xs">
                        {inv.duration_ms}ms
                      </TableCell>
                      <TableCell className="text-xs text-red-500 max-w-[150px] truncate">
                        {inv.error || "-"}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground whitespace-nowrap">
                        {new Date(inv.created_at).toLocaleString()}
                      </TableCell>
                    </TableRow>
                  ))}
                  {invocations.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={8} className="text-center text-muted-foreground py-8">
                        {t('adminInvocations.noInvocations')}
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

      {/* Detail dialog */}
      <Dialog open={!!detailRecord} onOpenChange={(open) => { if (!open) setDetailRecord(null) }}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{t('adminInvocations.detail')}</DialogTitle>
          </DialogHeader>
          {detailRecord && (
            <div className="space-y-3 text-sm">
              <DetailRow label={t('adminInvocations.id')} value={detailRecord.id} mono copy />
              <DetailRow label={t('adminInvocations.agentId')} value={detailRecord.agent_id} mono copy />
              <DetailRow label={t('adminInvocations.userId')} value={detailRecord.user_id || "-"} mono />
              <DetailRow label={t('adminInvocations.protocol')} value={detailRecord.protocol || "unknown"} />
              <DetailRow label={t('adminInvocations.status')} value={String(detailRecord.status_code)} />
              <DetailRow label={t('adminInvocations.duration')} value={`${detailRecord.duration_ms}ms`} />
              <DetailRow label={t('adminInvocations.error')} value={detailRecord.error || "-"} />
              <DetailRow label={t('invocations.ipAddress')} value={detailRecord.ip_address || "-"} />
              <DetailRow label={t('adminInvocations.createdAt')} value={new Date(detailRecord.created_at).toLocaleString()} />
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}

function DetailRow({ label, value, mono, copy }: { label: string; value: string; mono?: boolean; copy?: boolean }) {
  return (
    <div className="flex items-start justify-between gap-4">
      <span className="text-muted-foreground shrink-0">{label}</span>
      <div className="flex items-center gap-1 min-w-0">
        <span className={`text-right break-all ${mono ? "font-mono text-xs" : ""}`}>{value}</span>
        {copy && value !== "-" && <CopyButton value={value} />}
      </div>
    </div>
  )
}
