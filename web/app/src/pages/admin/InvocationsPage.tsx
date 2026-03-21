import { useState } from "react"
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

const PAGE_SIZE = 50

export function InvocationsPage() {
  const { t } = useTranslation()
  const [agentFilter, setAgentFilter] = useState("")
  const [userFilter, setUserFilter] = useState("")
  const [sort, setSort] = useState("")
  const [page, setPage] = useState(0)

  const { data, loading, error } = useAdminInvocations(
    agentFilter || undefined,
    userFilter || undefined,
    sort || undefined,
    PAGE_SIZE,
    page * PAGE_SIZE
  )

  const total = data?.total ?? 0
  const totalPages = Math.ceil(total / PAGE_SIZE)
  const invocations = data?.invocations ?? []

  const statusBadgeVariant = (code: number) => {
    if (code >= 200 && code < 300) return "default"
    if (code >= 400 && code < 500) return "secondary"
    if (code >= 500) return "destructive"
    return "outline"
  }

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
                    <TableRow key={inv.id}>
                      <TableCell className="font-mono text-xs max-w-[80px] truncate">
                        {inv.id.slice(0, 8)}...
                      </TableCell>
                      <TableCell className="font-mono text-xs max-w-[100px] truncate">
                        {inv.agent_id}
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
    </div>
  )
}
